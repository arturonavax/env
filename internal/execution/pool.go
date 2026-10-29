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
	Exclusive  bool // If true, drains prior parallel tasks and runs strictly alone
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

		var progressMu sync.Mutex
		for idx := batchStart; idx < batchEnd; idx++ {
			taskIdx := idx
			g.Go(func() error {
				if err := gCtx.Err(); err != nil {
					return err
				}
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
		// LIFO compensation across all previously succeeded steps
		for pos := len(completedIndices) - 1; pos >= 0; pos-- {
			idx := completedIndices[pos]
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
