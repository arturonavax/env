package execution

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

func task(id int, deps []int, fn func(ctx context.Context) error) Task {
	return Task{
		Step:    domain.PlanStep{ID: id, ResourceID: fmt.Sprintf("r%d", id), DependsOn: deps},
		Execute: fn,
	}
}

// DAG-01: randomized DAGs, shuffled submission order; every dependency must finish before its
// dependent starts, whatever the slice order and worker count.
func TestDAGRespectsDependenciesInRandomOrder(t *testing.T) {
	for seed := uint64(1); seed <= 25; seed++ {
		rng := rand.New(rand.NewPCG(seed, seed*7919))
		const n = 40

		deps := make([][]int, n+1)
		for id := 2; id <= n; id++ {
			for range rng.IntN(4) {
				deps[id] = append(deps[id], 1+rng.IntN(id-1)) // edges only to lower ids: acyclic
			}
		}

		// rand.PCG is not goroutine-safe: draw every random value before tasks run concurrently.
		sleeps := make([]time.Duration, n+1)
		for id := range sleeps {
			sleeps[id] = time.Duration(rng.IntN(3)) * time.Millisecond
		}

		var mu sync.Mutex
		finished := map[int]bool{}
		var violations []string

		build := func(id int) Task {
			return task(id, deps[id], func(ctx context.Context) error {
				mu.Lock()
				for _, d := range deps[id] {
					if !finished[d] {
						violations = append(violations, fmt.Sprintf("seed %d: step %d started before dep %d", seed, id, d))
					}
				}
				mu.Unlock()
				time.Sleep(sleeps[id])
				mu.Lock()
				finished[id] = true
				mu.Unlock()
				return nil
			})
		}

		order := rng.Perm(n)
		tasks := make([]Task, 0, n)
		for _, p := range order {
			tasks = append(tasks, build(p+1))
		}

		results, err := NewEngine(1+int(seed%6)).ExecuteTasks(context.Background(), tasks, nil)
		if err != nil {
			t.Fatalf("seed %d: %v", seed, err)
		}
		for _, v := range violations {
			t.Error(v)
		}
		for _, r := range results {
			if r.Status != domain.StepSuccess {
				t.Fatalf("seed %d: step %d status %v", seed, r.StepID, r.Status)
			}
		}
	}
}

// A dependent in the middle of a batch must wait; independent tasks still overlap (real DAG, not batches).
func TestDAGOverlapsIndependentWorkAndWaitsForDependency(t *testing.T) {
	aDone := make(chan struct{})
	var bStartedBeforeA atomic.Bool

	tasks := []Task{
		task(1, nil, func(ctx context.Context) error {
			time.Sleep(60 * time.Millisecond)
			close(aDone)
			return nil
		}),
		task(2, []int{1}, func(ctx context.Context) error { // contiguous with 1, but depends on it
			select {
			case <-aDone:
			default:
				bStartedBeforeA.Store(true)
			}
			return nil
		}),
		task(3, nil, func(ctx context.Context) error { // independent: must run while 1 is still sleeping
			select {
			case <-aDone:
				return errors.New("independent task serialized behind unrelated task")
			default:
				return nil
			}
		}),
	}

	if _, err := NewEngine(4).ExecuteTasks(context.Background(), tasks, nil); err != nil {
		t.Fatal(err)
	}
	if bStartedBeforeA.Load() {
		t.Fatal("dependent started before its dependency finished")
	}
}

func TestDAGBoundedWorkers(t *testing.T) {
	const limit = 3
	var cur, peak atomic.Int32

	tasks := make([]Task, 20)
	for i := range tasks {
		tasks[i] = task(i+1, nil, func(ctx context.Context) error {
			n := cur.Add(1)
			for {
				p := peak.Load()
				if n <= p || peak.CompareAndSwap(p, n) {
					break
				}
			}
			time.Sleep(5 * time.Millisecond)
			cur.Add(-1)
			return nil
		})
	}

	if _, err := NewEngine(limit).ExecuteTasks(context.Background(), tasks, nil); err != nil {
		t.Fatal(err)
	}
	if peak.Load() > limit {
		t.Fatalf("peak concurrency %d exceeds limit %d", peak.Load(), limit)
	}
	if peak.Load() < 2 {
		t.Fatalf("no parallelism observed (peak %d)", peak.Load())
	}
}

func TestDAGLocksSerializeWithoutBlockingOthers(t *testing.T) {
	var inLock atomic.Int32
	var overlap atomic.Bool

	locked := func(id int) Task {
		tk := task(id, nil, func(ctx context.Context) error {
			if inLock.Add(1) > 1 {
				overlap.Store(true)
			}
			time.Sleep(10 * time.Millisecond)
			inLock.Add(-1)
			return nil
		})
		tk.Locks = []string{"pkg:apt"}
		return tk
	}

	tasks := []Task{locked(1), locked(2), locked(3), task(4, nil, nil)}
	if _, err := NewEngine(4).ExecuteTasks(context.Background(), tasks, nil); err != nil {
		t.Fatal(err)
	}
	if overlap.Load() {
		t.Fatal("tasks sharing a lock ran concurrently")
	}
}

func TestDAGRejectsInvalidGraphs(t *testing.T) {
	noop := func(context.Context) error { return nil }
	cases := []struct {
		name  string
		tasks []Task
		want  error
	}{
		{"cycle", []Task{task(1, []int{2}, noop), task(2, []int{1}, noop)}, ErrDependencyCycle},
		{"self", []Task{task(1, []int{1}, noop)}, ErrDependencyCycle},
		{"unknown", []Task{task(1, []int{9}, noop)}, ErrUnknownDependency},
		{"duplicate", []Task{task(1, nil, noop), task(1, nil, noop)}, ErrDuplicateStepID},
		{ // explicit forward dependency contradicts the barrier order
			"barrier-contradiction",
			[]Task{
				task(1, []int{2}, noop),
				{Step: domain.PlanStep{ID: 2, IsExclusive: true}, Execute: noop},
			},
			ErrDependencyCycle,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var ran atomic.Bool
			for i := range tc.tasks {
				orig := tc.tasks[i].Execute
				tc.tasks[i].Execute = func(ctx context.Context) error { ran.Store(true); return orig(ctx) }
			}
			_, err := NewEngine(2).ExecuteTasks(context.Background(), tc.tasks, nil)
			if !errors.Is(err, tc.want) {
				t.Fatalf("want %v, got %v", tc.want, err)
			}
			if ran.Load() {
				t.Fatal("invalid graph must fail before running anything")
			}
		})
	}
}

func TestDAGFailureCancelsSiblingsAndWaitsForThem(t *testing.T) {
	var siblingFinished atomic.Bool

	tasks := []Task{
		task(1, nil, func(ctx context.Context) error { return errors.New("boom") }),
		task(2, nil, func(ctx context.Context) error {
			<-ctx.Done() // must be cancelled by the failure
			siblingFinished.Store(true)
			return ctx.Err()
		}),
		task(3, []int{1}, func(ctx context.Context) error { return errors.New("dependent of failed step must never run") }),
	}

	results, err := NewEngine(4, WithRollbackOnError(false)).ExecuteTasks(context.Background(), tasks, nil)
	if err == nil {
		t.Fatal("expected failure")
	}
	if !siblingFinished.Load() {
		t.Fatal("ExecuteTasks returned before the cancelled sibling finished (goroutine leak)")
	}
	if results[2].Status != domain.StepPending {
		t.Fatalf("dependent of failed step must stay pending, got %v", results[2].Status)
	}
}

func TestDAGPanicBecomesFailureAndRollsBack(t *testing.T) {
	var compensated atomic.Bool

	first := task(1, nil, func(ctx context.Context) error { return nil })
	first.Compensate = func(ctx context.Context) error { compensated.Store(true); return nil }
	second := task(2, []int{1}, func(ctx context.Context) error { panic("kaboom") })

	_, err := NewEngine(2).ExecuteTasks(context.Background(), []Task{first, second}, nil)
	if err == nil {
		t.Fatal("expected error from panicking task")
	}
	if !compensated.Load() {
		t.Fatal("completed task was not compensated after sibling panic")
	}
}

func TestDAGCompensationFailureIsReported(t *testing.T) {
	first := task(1, nil, nil)
	first.Compensate = func(ctx context.Context) error { return errors.New("cannot undo") }
	second := task(2, []int{1}, func(ctx context.Context) error { return errors.New("step failed") })

	_, err := NewEngine(2).ExecuteTasks(context.Background(), []Task{first, second}, nil)
	if err == nil {
		t.Fatal("expected error")
	}
	for _, want := range []string{"step failed", "cannot undo"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q must mention %q", err, want)
		}
	}
}

func TestDAGAlreadyCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var ran atomic.Bool
	_, err := NewEngine(2).ExecuteTasks(ctx, []Task{
		task(1, nil, func(context.Context) error { ran.Store(true); return nil }),
	}, nil)
	if !errors.Is(err, ErrExecutionInterrupted) {
		t.Fatalf("want ErrExecutionInterrupted, got %v", err)
	}
	if ran.Load() {
		t.Fatal("task ran under a cancelled context")
	}
}

func TestProgressCallbackIsSerialized(t *testing.T) {
	var inCallback atomic.Int32
	var concurrent atomic.Bool

	tasks := make([]Task, 30)
	for i := range tasks {
		tasks[i] = task(i+1, nil, nil)
	}
	_, err := NewEngine(8).ExecuteTasks(context.Background(), tasks, func(Result) {
		if inCallback.Add(1) > 1 {
			concurrent.Store(true)
		}
		time.Sleep(100 * time.Microsecond)
		inCallback.Add(-1)
	})
	if err != nil {
		t.Fatal(err)
	}
	if concurrent.Load() {
		t.Fatal("onProgress invoked concurrently")
	}
}

func TestCompensationOrderDependentsFirst(t *testing.T) {
	tasks := []Task{
		task(1, nil, nil),
		task(2, []int{1}, nil),
		task(3, []int{2}, nil),
		task(4, nil, nil),
	}
	g, err := buildGraph(tasks)
	if err != nil {
		t.Fatal(err)
	}
	// Completion order deliberately differs from topological order.
	order := g.compensationOrder([]int{0, 3, 1, 2})

	at := map[int]int{}
	for p, idx := range order {
		at[idx] = p
	}
	if !(at[2] < at[1] && at[1] < at[0]) {
		t.Fatalf("dependents must be compensated before dependencies, got %v", order)
	}
	if len(order) != 4 {
		t.Fatalf("lost tasks in compensation order: %v", order)
	}
}
