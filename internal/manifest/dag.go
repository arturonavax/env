package manifest

import (
	"fmt"
	"strings"
)

type visitState uint8

const (
	visitUnseen visitState = iota
	visitActive
	visitDone
)

// ResolveInstallOrder resolves transitive depends_on relationships and returns PackageSpecs in topological install order.
func (c *Catalog) ResolveInstallOrder(ids []string) ([]PackageSpec, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	states := make(map[string]visitState, len(ids))
	stack := make([]string, 0, len(ids))
	ordered := make([]PackageSpec, 0, len(ids))

	var visit func(id string) error
	visit = func(id string) error {
		trimmed := strings.TrimSpace(id)
		if trimmed == "" {
			return nil
		}

		switch states[trimmed] {
		case visitDone:
			return nil
		case visitActive:
			cyclePath := append(append([]string{}, stack...), trimmed)
			return fmt.Errorf("dependency cycle detected: %s", strings.Join(cyclePath, " -> "))
		}

		spec, ok := c.packages[trimmed]
		if !ok {
			return fmt.Errorf("unknown package dependency %q", trimmed)
		}

		states[trimmed] = visitActive
		stack = append(stack, trimmed)

		for _, dep := range spec.DependsOn {
			if err := visit(dep); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		states[trimmed] = visitDone
		ordered = append(ordered, spec)
		return nil
	}

	for _, id := range ids {
		if err := visit(id); err != nil {
			return nil, err
		}
	}

	return ordered, nil
}
