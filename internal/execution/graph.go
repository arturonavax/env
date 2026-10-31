package execution

import (
	"errors"
	"fmt"
	"slices"
)

// Sentinel errors returned (wrapped) when a task set is not a valid DAG.
var (
	// ErrDuplicateStepID indicates two tasks share the same Step.ID.
	ErrDuplicateStepID = errors.New("duplicate step id in execution graph")
	// ErrUnknownDependency indicates a DependsOn entry references a step that is not in the task set.
	ErrUnknownDependency = errors.New("step depends on unknown step id")
	// ErrDependencyCycle indicates the dependency (plus exclusive-barrier) edges contain a cycle.
	ErrDependencyCycle = errors.New("dependency cycle in execution graph")
)

// graph is the immutable scheduling structure derived from a task slice.
// Nodes are task indices, so every lookup on the hot path is a slice index, not a map hit.
type graph struct {
	succ  [][]int    // succ[i]: tasks that become unblocked (partially) when i succeeds
	indeg []int      // indeg[i]: number of unmet predecessor edges (duplicates counted consistently)
	locks [][]string // locks[i]: effective lock domains of task i
}

// buildGraph validates tasks and derives the dependency graph.
//
// Edges come from two sources:
//  1. Explicit Step.DependsOn (by step ID).
//  2. Implicit exclusive barriers: an exclusive task runs strictly alone, after every task that
//     precedes it in slice order and before every task that follows it. This is encoded in O(n)
//     edges (barrier -> following tasks, preceding tasks -> barrier) instead of O(n^2).
func buildGraph(tasks []Task) (*graph, error) {
	n := len(tasks)
	g := &graph{
		succ:  make([][]int, n),
		indeg: make([]int, n),
		locks: make([][]string, n),
	}

	idToIdx := make(map[int]int, n)
	for i := range tasks {
		id := tasks[i].Step.ID
		if prev, dup := idToIdx[id]; dup {
			return nil, fmt.Errorf("%w: id %d at positions %d and %d", ErrDuplicateStepID, id, prev, i)
		}
		idToIdx[id] = i

		g.locks[i] = tasks[i].Locks
		if len(g.locks[i]) == 0 {
			g.locks[i] = tasks[i].Step.Locks
		}
	}

	addEdge := func(from, to int) {
		g.succ[from] = append(g.succ[from], to)
		g.indeg[to]++
	}

	for i := range tasks {
		for _, depID := range tasks[i].Step.DependsOn {
			j, ok := idToIdx[depID]
			if !ok {
				return nil, fmt.Errorf("%w: step %d requires %d", ErrUnknownDependency, tasks[i].Step.ID, depID)
			}
			if j == i {
				return nil, fmt.Errorf("%w: step %d depends on itself", ErrDependencyCycle, tasks[i].Step.ID)
			}
			addEdge(j, i)
		}
	}

	lastBarrier := -1
	sinceBarrier := make([]int, 0, n)
	for i := range tasks {
		if isExclusive(&tasks[i]) {
			for _, j := range sinceBarrier {
				addEdge(j, i)
			}
			if len(sinceBarrier) == 0 && lastBarrier >= 0 {
				addEdge(lastBarrier, i)
			}
			lastBarrier = i
			sinceBarrier = sinceBarrier[:0]
			continue
		}
		if lastBarrier >= 0 {
			addEdge(lastBarrier, i)
		}
		sinceBarrier = append(sinceBarrier, i)
	}

	if err := g.checkAcyclic(tasks); err != nil {
		return nil, err
	}
	return g, nil
}

func isExclusive(t *Task) bool {
	return t.Exclusive || t.Step.IsExclusive
}

// checkAcyclic runs Kahn's algorithm on a copy of the in-degrees; leftover nodes form a cycle.
func (g *graph) checkAcyclic(tasks []Task) error {
	indeg := slices.Clone(g.indeg)
	queue := make([]int, 0, len(indeg))
	for i, d := range indeg {
		if d == 0 {
			queue = append(queue, i)
		}
	}

	visited := 0
	for head := 0; head < len(queue); head++ {
		visited++
		for _, s := range g.succ[queue[head]] {
			indeg[s]--
			if indeg[s] == 0 {
				queue = append(queue, s)
			}
		}
	}
	if visited == len(indeg) {
		return nil
	}

	for i, d := range indeg {
		if d > 0 {
			return fmt.Errorf("%w: involves step %d (%s)", ErrDependencyCycle, tasks[i].Step.ID, tasks[i].Step.ResourceID)
		}
	}
	return ErrDependencyCycle
}

// compensationOrder returns completed task indices ordered so a task is compensated only after
// every completed task that depends on it. Among eligible tasks the most recently completed goes
// first (LIFO), which makes the order deterministic and matches operator intuition.
// completed holds task indices in completion order.
func (g *graph) compensationOrder(completed []int) []int {
	if len(completed) <= 1 {
		return completed
	}

	pos := make(map[int]int, len(completed)) // task index -> completion position
	for p, idx := range completed {
		pos[idx] = p
	}

	// pending[i]: completed successors of i that have not been compensated yet.
	pending := make(map[int]int, len(completed))
	for _, idx := range completed {
		for _, s := range g.succ[idx] {
			if _, ok := pos[s]; ok {
				pending[idx]++
			}
		}
	}

	// eligible holds completion positions; the largest is the most recently completed.
	eligible := make([]int, 0, len(completed))
	for _, idx := range completed {
		if pending[idx] == 0 {
			eligible = append(eligible, pos[idx])
		}
	}
	slices.Sort(eligible)

	// preds lets us release predecessors once a task is compensated.
	preds := make(map[int][]int, len(completed))
	for _, idx := range completed {
		for _, s := range g.succ[idx] {
			if _, ok := pos[s]; ok {
				preds[s] = append(preds[s], idx)
			}
		}
	}

	order := make([]int, 0, len(completed))
	for len(eligible) > 0 {
		last := len(eligible) - 1
		idx := completed[eligible[last]]
		eligible = eligible[:last]
		order = append(order, idx)

		for _, p := range preds[idx] {
			pending[p]--
			if pending[p] == 0 {
				at, _ := slices.BinarySearch(eligible, pos[p])
				eligible = slices.Insert(eligible, at, pos[p])
			}
		}
	}
	return order
}
