package execution

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

func TestEngineDrainsParallelBeforeExclusiveAndNeverOverlaps(t *testing.T) {
	eng := NewEngine(4)
	var activeParallel atomic.Int32
	var overlapDetected atomic.Bool

	tasks := []Task{
		{
			Step: domain.PlanStep{ID: 1, ResourceID: "p1"},
			Execute: func(ctx context.Context) error {
				activeParallel.Add(1)
				time.Sleep(40 * time.Millisecond)
				activeParallel.Add(-1)
				return nil
			},
		},
		{
			Step: domain.PlanStep{ID: 2, ResourceID: "p2"},
			Execute: func(ctx context.Context) error {
				activeParallel.Add(1)
				time.Sleep(50 * time.Millisecond)
				activeParallel.Add(-1)
				return nil
			},
		},
		{
			Step:      domain.PlanStep{ID: 3, ResourceID: "apt-exclusive"},
			Exclusive: true,
			Execute: func(ctx context.Context) error {
				if activeParallel.Load() > 0 {
					overlapDetected.Store(true)
				}
				return nil
			},
		},
	}

	results, err := eng.ExecuteTasks(context.Background(), tasks, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if overlapDetected.Load() {
		t.Fatal("exclusive task executed while parallel tasks were still active")
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}
}

func TestEngineLIFOCompensationOnFailure(t *testing.T) {
	eng := NewEngine(1, WithRollbackOnError(true))
	var mu sync.Mutex
	compensatedOrder := []int{}

	tasks := []Task{
		{
			Step:      domain.PlanStep{ID: 1, ResourceID: "step-1"},
			Exclusive: true,
			Execute:   func(ctx context.Context) error { return nil },
			Compensate: func(ctx context.Context) error {
				mu.Lock()
				compensatedOrder = append(compensatedOrder, 1)
				mu.Unlock()
				return nil
			},
		},
		{
			Step:      domain.PlanStep{ID: 2, ResourceID: "step-2"},
			Exclusive: true,
			Execute:   func(ctx context.Context) error { return nil },
			Compensate: func(ctx context.Context) error {
				mu.Lock()
				compensatedOrder = append(compensatedOrder, 2)
				mu.Unlock()
				return nil
			},
		},
		{
			Step:      domain.PlanStep{ID: 3, ResourceID: "step-3-fail"},
			Exclusive: true,
			Execute: func(ctx context.Context) error {
				return errors.New("boom")
			},
		},
	}

	results, err := eng.ExecuteTasks(context.Background(), tasks, nil)
	if err == nil {
		t.Fatal("expected execution error, got nil")
	}
	if len(compensatedOrder) != 2 || compensatedOrder[0] != 2 || compensatedOrder[1] != 1 {
		t.Fatalf("expected LIFO compensation order [2, 1], got %v", compensatedOrder)
	}
	if results[0].Status != domain.StepCompensated || results[1].Status != domain.StepCompensated {
		t.Fatalf("expected steps 1 and 2 marked StepCompensated, got %v, %v", results[0].Status, results[1].Status)
	}
}

func TestEngineInverseTopologicalRollbackOrder(t *testing.T) {
	eng := NewEngine(4, WithRollbackOnError(true))
	var mu sync.Mutex
	var order []int

	// Task 1 (base library)
	// Task 2 (depends on Task 1)
	// Task 3 (independent)
	// Task 4 (fails)
	tasks := []Task{
		{
			Step:    domain.PlanStep{ID: 1, ResourceID: "lib-base"},
			Execute: func(ctx context.Context) error { return nil },
			Compensate: func(ctx context.Context) error {
				mu.Lock()
				order = append(order, 1)
				mu.Unlock()
				return nil
			},
		},
		{
			Step:    domain.PlanStep{ID: 2, ResourceID: "app-dependent", DependsOn: []int{1}},
			Execute: func(ctx context.Context) error { return nil },
			Compensate: func(ctx context.Context) error {
				mu.Lock()
				order = append(order, 2)
				mu.Unlock()
				return nil
			},
		},
		{
			Step:    domain.PlanStep{ID: 3, ResourceID: "unrelated-tool"},
			Execute: func(ctx context.Context) error { return nil },
			Compensate: func(ctx context.Context) error {
				mu.Lock()
				order = append(order, 3)
				mu.Unlock()
				return nil
			},
		},
		{
			Step:      domain.PlanStep{ID: 4, ResourceID: "fail-step"},
			Exclusive: true,
			Execute: func(ctx context.Context) error {
				return errors.New("simulated failure")
			},
		},
	}

	_, err := eng.ExecuteTasks(context.Background(), tasks, nil)
	if err == nil {
		t.Fatal("expected failure")
	}

	// In inverse topological rollback, Task 2 (dependent) MUST be compensated before Task 1 (base)!
	pos1, pos2 := -1, -1
	for i, id := range order {
		if id == 1 {
			pos1 = i
		}
		if id == 2 {
			pos2 = i
		}
	}
	if pos1 == -1 || pos2 == -1 {
		t.Fatalf("expected both tasks 1 and 2 in rollback, got %v", order)
	}
	if pos2 >= pos1 {
		t.Fatalf("inverse topological violation: task 2 (dependent) must be compensated BEFORE task 1 (base), got order: %v", order)
	}
}

func TestEngineFineGrainedLockSerialization(t *testing.T) {
	eng := NewEngine(4)
	var activeLock atomic.Int32
	var lockCollision atomic.Bool

	tasks := []Task{
		{
			Step: domain.PlanStep{ID: 1, ResourceID: "step-apt-1", Locks: []string{"pkg:apt"}},
			Execute: func(ctx context.Context) error {
				val := activeLock.Add(1)
				if val > 1 {
					lockCollision.Store(true)
				}
				time.Sleep(30 * time.Millisecond)
				activeLock.Add(-1)
				return nil
			},
		},
		{
			Step: domain.PlanStep{ID: 2, ResourceID: "step-apt-2", Locks: []string{"pkg:apt"}},
			Execute: func(ctx context.Context) error {
				val := activeLock.Add(1)
				if val > 1 {
					lockCollision.Store(true)
				}
				time.Sleep(30 * time.Millisecond)
				activeLock.Add(-1)
				return nil
			},
		},
	}

	_, err := eng.ExecuteTasks(context.Background(), tasks, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if lockCollision.Load() {
		t.Fatal("fine-grained lock conflict: steps sharing 'pkg:apt' executed concurrently")
	}
}
