package execution

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/arturonavax/env/internal/domain"
)

// ErrExecutionInterrupted indicates execution stopped due to context cancellation or signal.
var ErrExecutionInterrupted = errors.New("execution was cancelled or interrupted")

// Task represents a discrete executable unit within an ExecutionPlan.
type Task struct {
	Step       domain.PlanStep
	Exclusive  bool     // If true, drains prior parallel tasks and runs strictly alone
	Locks      []string // Fine-grained lock domains (e.g. "pkg:apt", "file:~/.config/nvim", "service:daemon")
	Timeout    time.Duration
	Execute    func(ctx context.Context) error
	Compensate func(ctx context.Context) error
}

// Result captures the execution outcome of a Task.
type Result struct {
	StepID int
	Status domain.StepStatus
	Error  error
}

// EngineOption configures an Engine instance.
type EngineOption func(*Engine)

// WithRollbackOnError toggles automatic LIFO compensation across all previously succeeded tasks when a step fails.
func WithRollbackOnError(enabled bool) EngineOption {
	return func(e *Engine) {
		e.rollbackOnError = enabled
	}
}

// WithDefaultTaskTimeout sets the fallback per-task timeout when Task.Timeout is unset.
func WithDefaultTaskTimeout(d time.Duration) EngineOption {
	return func(e *Engine) {
		if d > 0 {
			e.defaultTimeout = d
		}
	}
}

// Engine coordinates structured concurrent and exclusive execution of tasks with LIFO rollback.
type Engine struct {
	maxWorkers      int
	rollbackOnError bool
	defaultTimeout  time.Duration
}

// NewEngine creates an Engine with bounded concurrency and functional options.
func NewEngine(maxWorkers int, opts ...EngineOption) *Engine {
	if maxWorkers <= 0 {
		maxWorkers = 4
	}
	e := &Engine{
		maxWorkers:      maxWorkers,
		rollbackOnError: true,
		defaultTimeout:  300 * time.Second,
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// ExecuteTasks partitions tasks into parallel batches separated by exclusive barriers,
// guarantees all goroutines finish before returning, and executes LIFO compensation on error.
func (e *Engine) ExecuteTasks(
	ctx context.Context,
	tasks []Task,
	onProgress func(Result),
) ([]Result, error) {
	if len(tasks) == 0 {
		return []Result{}, nil
	}

	results := make([]Result, len(tasks))
	for i := range tasks {
		results[i] = Result{
			StepID: tasks[i].Step.ID,
			Status: domain.StepPending,
		}
	}

	var completedMu sync.Mutex
	completedIndices := make([]int, 0, len(tasks))

	notify := func(r Result) {
		if onProgress != nil {
			onProgress(r)
		}
	}

	var execErr error
	i := 0
	for i < len(tasks) {
		if err := ctx.Err(); err != nil {
			execErr = fmt.Errorf("%w: %w", ErrExecutionInterrupted, err)
			break
		}

		if tasks[i].Exclusive || tasks[i].Step.IsExclusive {
			res := e.runSingleTask(ctx, &tasks[i])
			results[i] = res
			notify(res)
			if res.Error != nil {
				execErr = fmt.Errorf("step %d (%s) failed: %w", tasks[i].Step.ID, tasks[i].Step.ResourceID, res.Error)
				break
			}
			completedMu.Lock()
			completedIndices = append(completedIndices, i)
			completedMu.Unlock()
			i++
			continue
		}

		// Collect contiguous non-exclusive batch up to the next exclusive barrier
		batchStart := i
		for i < len(tasks) && !tasks[i].Exclusive && !tasks[i].Step.IsExclusive {
			i++
		}
		batchEnd := i

		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(e.maxWorkers)
		lm := newLockManager()

		var progressMu sync.Mutex
		for idx := batchStart; idx < batchEnd; idx++ {
			taskIdx := idx
			g.Go(func() error {
				if err := gCtx.Err(); err != nil {
					return err
				}
				locks := tasks[taskIdx].Locks
				if len(locks) == 0 {
					locks = tasks[taskIdx].Step.Locks
				}
				if err := lm.Acquire(gCtx, locks); err != nil {
					return err
				}
				defer lm.Release(locks)

				res := e.runSingleTask(gCtx, &tasks[taskIdx])
				results[taskIdx] = res

				progressMu.Lock()
				notify(res)
				progressMu.Unlock()

				if res.Error != nil {
					return fmt.Errorf("step %d (%s) failed: %w", tasks[taskIdx].Step.ID, tasks[taskIdx].Step.ResourceID, res.Error)
				}

				completedMu.Lock()
				completedIndices = append(completedIndices, taskIdx)
				completedMu.Unlock()
				return nil
			})
		}

		if batchErr := g.Wait(); batchErr != nil {
			execErr = batchErr
			break
		}
	}

	if execErr != nil && e.rollbackOnError && len(completedIndices) > 0 {
		// Rollback in inverse topological order (dependencies are undone after dependents)
		rollbackOrder := computeInverseTopologicalOrder(tasks, completedIndices)
		for _, idx := range rollbackOrder {
			compFn := tasks[idx].Compensate
			if compFn == nil {
				continue
			}
			compCtx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
			compErr := compFn(compCtx)
			cancel()
			if compErr == nil {
				results[idx].Status = domain.StepCompensated
				notify(results[idx])
			}
		}
	}

	return results, execErr
}

func (e *Engine) runSingleTask(ctx context.Context, task *Task) Result {
	res := Result{
		StepID: task.Step.ID,
		Status: domain.StepRunning,
	}

	if task.Execute == nil {
		res.Status = domain.StepSuccess
		return res
	}

	timeout := e.defaultTimeout
	if task.Timeout > 0 {
		timeout = task.Timeout
	}

	taskCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	if err := task.Execute(taskCtx); err != nil {
		res.Status = domain.StepFailed
		res.Error = err
		return res
	}

	res.Status = domain.StepSuccess
	return res
}

// computeInverseTopologicalOrder orders completed tasks such that if task B depends on task A,
// task B is compensated before task A.
func computeInverseTopologicalOrder(tasks []Task, completedIndices []int) []int {
	if len(completedIndices) <= 1 {
		return completedIndices
	}

	stepIDToIdx := make(map[int]int, len(tasks))
	for _, idx := range completedIndices {
		stepIDToIdx[tasks[idx].Step.ID] = idx
	}

	inDegree := make(map[int]int, len(completedIndices))
	dependents := make(map[int][]int, len(completedIndices))
	for _, bIdx := range completedIndices {
		inDegree[bIdx] = 0
	}

	for _, bIdx := range completedIndices {
		for _, depStepID := range tasks[bIdx].Step.DependsOn {
			if aIdx, ok := stepIDToIdx[depStepID]; ok && aIdx != bIdx {
				dependents[bIdx] = append(dependents[bIdx], aIdx)
				inDegree[aIdx]++
			}
		}
	}

	ready := make([]int, 0, len(completedIndices))
	for pos := len(completedIndices) - 1; pos >= 0; pos-- {
		idx := completedIndices[pos]
		if inDegree[idx] == 0 {
			ready = append(ready, idx)
		}
	}

	result := make([]int, 0, len(completedIndices))
	for len(ready) > 0 {
		curr := ready[0]
		ready = ready[1:]
		result = append(result, curr)

		for _, aIdx := range dependents[curr] {
			inDegree[aIdx]--
			if inDegree[aIdx] == 0 {
				ready = append(ready, aIdx)
			}
		}
	}

	if len(result) < len(completedIndices) {
		seen := make(map[int]bool, len(result))
		for _, idx := range result {
			seen[idx] = true
		}
		for pos := len(completedIndices) - 1; pos >= 0; pos-- {
			idx := completedIndices[pos]
			if !seen[idx] {
				result = append(result, idx)
			}
		}
	}

	return result
}

type lockManager struct {
	mu     sync.Mutex
	active map[string]bool
	cond   *sync.Cond
}

func newLockManager() *lockManager {
	lm := &lockManager{
		active: make(map[string]bool),
	}
	lm.cond = sync.NewCond(&lm.mu)
	return lm
}

func (lm *lockManager) Acquire(ctx context.Context, locks []string) error {
	if len(locks) == 0 {
		return nil
	}

	stop := context.AfterFunc(ctx, func() {
		lm.mu.Lock()
		lm.cond.Broadcast()
		lm.mu.Unlock()
	})
	defer stop()

	lm.mu.Lock()
	defer lm.mu.Unlock()

	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		conflict := false
		for _, l := range locks {
			if lm.active[l] {
				conflict = true
				break
			}
		}
		if !conflict {
			for _, l := range locks {
				lm.active[l] = true
			}
			return nil
		}
		lm.cond.Wait()
	}
}

func (lm *lockManager) Release(locks []string) {
	if len(locks) == 0 {
		return
	}
	lm.mu.Lock()
	for _, l := range locks {
		delete(lm.active, l)
	}
	lm.cond.Broadcast()
	lm.mu.Unlock()
}
