package execution

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

// ErrExecutionInterrupted indicates execution stopped due to context cancellation or signal.
var ErrExecutionInterrupted = errors.New("execution was cancelled or interrupted")

// compensationTimeout bounds each compensation call; it runs on a fresh context so that
// rollback still happens after the caller's context was cancelled.
const compensationTimeout = 45 * time.Second

// Task represents a discrete executable unit within an ExecutionPlan.
type Task struct {
	Step       domain.PlanStep
	Exclusive  bool     // If true, drains prior tasks and runs strictly alone (barrier)
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

// WithRollbackOnError toggles automatic compensation across all previously succeeded tasks when a step fails.
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

// Engine is a DAG scheduler: tasks run as soon as every dependency has succeeded, bounded by a worker
// limit and by resource locks, with compensation in reverse dependency order on failure.
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

type completion struct {
	idx int
	res Result
}

// run owns all mutable scheduling state. It is only touched by the coordinator goroutine
// (the caller of ExecuteTasks), so no mutex is needed; workers communicate solely via the done channel.
type run struct {
	tasks   []Task
	g       *graph
	indeg   []int
	ready   []int // task indices ready to start, ascending (deterministic order)
	held    map[string]struct{}
	results []Result
	done    chan completion
	running int
}

// ExecuteTasks schedules tasks over their dependency graph and guarantees every goroutine it
// started has finished before returning. On the first failure it stops dispatching, cancels the
// in-flight tasks, waits for them, and compensates completed tasks in reverse dependency order.
//
// Progress callbacks are invoked from the calling goroutine only, never concurrently.
func (e *Engine) ExecuteTasks(
	ctx context.Context,
	tasks []Task,
	onProgress func(Result),
) ([]Result, error) {
	if len(tasks) == 0 {
		return []Result{}, nil
	}

	g, err := buildGraph(tasks)
	if err != nil {
		return nil, err
	}

	notify := func(r Result) {
		if onProgress != nil {
			onProgress(r)
		}
	}

	r := newRun(tasks, g)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var execErr error
	completed := make([]int, 0, len(tasks))

	for {
		if execErr == nil {
			if cerr := ctx.Err(); cerr != nil {
				execErr = fmt.Errorf("%w: %w", ErrExecutionInterrupted, cerr)
				cancel()
			}
		}
		if execErr == nil {
			r.dispatch(runCtx, e, notify)
		}
		if r.running == 0 {
			break
		}

		// Workers observe runCtx (derived from ctx), so cancellation always reaches them and they
		// always report back; waiting on done alone cannot leak or hang beyond task cooperation.
		c := <-r.done
		r.running--
		r.release(c.idx)
		r.results[c.idx] = c.res
		notify(c.res)

		if c.res.Error != nil {
			if execErr == nil {
				step := &tasks[c.idx].Step
				execErr = fmt.Errorf("step %d (%s) failed: %w", step.ID, step.ResourceID, c.res.Error)
				cancel()
			}
			continue
		}

		completed = append(completed, c.idx)
		if execErr == nil {
			r.unblock(c.idx)
		}
	}

	if execErr != nil && e.rollbackOnError && len(completed) > 0 {
		execErr = errors.Join(execErr, e.compensate(g, tasks, completed, r.results, notify))
	}

	return r.results, execErr
}

func newRun(tasks []Task, g *graph) *run {
	r := &run{
		tasks:   tasks,
		g:       g,
		indeg:   slices.Clone(g.indeg),
		ready:   make([]int, 0, len(tasks)),
		held:    make(map[string]struct{}),
		results: make([]Result, len(tasks)),
		done:    make(chan completion, len(tasks)), // buffered: a worker never blocks on reporting
	}
	for i := range tasks {
		r.results[i] = Result{StepID: tasks[i].Step.ID, Status: domain.StepPending}
		if r.indeg[i] == 0 {
			r.ready = append(r.ready, i)
		}
	}
	return r
}

// dispatch starts ready tasks while worker slots and their locks are available. Tasks whose locks
// are busy are skipped (not blocked on), so one contended resource never starves unrelated work.
func (r *run) dispatch(ctx context.Context, e *Engine, notify func(Result)) {
	for r.running < e.maxWorkers {
		pos := r.firstRunnable()
		if pos < 0 {
			return
		}
		idx := r.ready[pos]
		r.ready = slices.Delete(r.ready, pos, pos+1)

		for _, l := range r.g.locks[idx] {
			r.held[l] = struct{}{}
		}
		r.running++
		r.results[idx] = Result{StepID: r.tasks[idx].Step.ID, Status: domain.StepRunning}
		if notify != nil {
			notify(r.results[idx])
		}

		go func() {
			r.done <- completion{idx: idx, res: e.runSingleTask(ctx, &r.tasks[idx])}
		}()
	}
}

// firstRunnable returns the position in ready of the lowest-index task whose locks are all free.
func (r *run) firstRunnable() int {
	for pos, idx := range r.ready {
		if r.locksFree(r.g.locks[idx]) {
			return pos
		}
	}
	return -1
}

func (r *run) locksFree(locks []string) bool {
	for _, l := range locks {
		if _, busy := r.held[l]; busy {
			return false
		}
	}
	return true
}

func (r *run) release(idx int) {
	for _, l := range r.g.locks[idx] {
		delete(r.held, l)
	}
}

// unblock decrements successor in-degrees and queues the ones that became ready.
func (r *run) unblock(idx int) {
	for _, s := range r.g.succ[idx] {
		r.indeg[s]--
		if r.indeg[s] == 0 {
			at, _ := slices.BinarySearch(r.ready, s)
			r.ready = slices.Insert(r.ready, at, s)
		}
	}
}

// compensate undoes completed tasks (dependents before their dependencies) and reports failures,
// so a broken rollback is never silent.
func (e *Engine) compensate(
	g *graph,
	tasks []Task,
	completed []int,
	results []Result,
	notify func(Result),
) error {
	var errs []error
	for _, idx := range g.compensationOrder(completed) {
		fn := tasks[idx].Compensate
		if fn == nil {
			continue
		}

		ctx, cancel := context.WithTimeout(context.Background(), compensationTimeout)
		err := fn(ctx)
		cancel()
		if err != nil {
			errs = append(errs, fmt.Errorf("compensating step %d (%s): %w",
				tasks[idx].Step.ID, tasks[idx].Step.ResourceID, err))
			continue
		}

		results[idx].Status = domain.StepCompensated
		notify(results[idx])
	}
	return errors.Join(errs...)
}

// runSingleTask executes one task under its timeout and converts a panic into a failed Result:
// an unrecovered panic in a worker goroutine would kill the process and skip rollback entirely.
func (e *Engine) runSingleTask(ctx context.Context, task *Task) (res Result) {
	res = Result{StepID: task.Step.ID, Status: domain.StepRunning}

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

	defer func() {
		if p := recover(); p != nil {
			res.Status = domain.StepFailed
			res.Error = fmt.Errorf("task panicked: %v", p)
		}
	}()

	if err := task.Execute(taskCtx); err != nil {
		res.Status = domain.StepFailed
		res.Error = err
		return res
	}

	res.Status = domain.StepSuccess
	return res
}
