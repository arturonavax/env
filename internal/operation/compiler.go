package operation

import (
	"fmt"
	"strings"

	"github.com/arturonavax/env/internal/domain"
)

// CompileIR transforms a domain.ExecutionPlan into an immutable PlanIR.
func CompileIR(plan *domain.ExecutionPlan, goos, arch string) (*PlanIR, error) {
	if plan == nil {
		return nil, fmt.Errorf("plan cannot be nil")
	}

	ir := &PlanIR{
		ID:            plan.ID,
		SchemaVersion: 1,
		Operations:    make([]Operation, 0, len(plan.Steps)),
	}

	stepIDToOpID := make(map[int]string, len(plan.Steps))
	for _, step := range plan.Steps {
		opID := fmt.Sprintf("step-%d-%s", step.ID, step.ResourceID)
		if step.ResourceID == "" {
			opID = fmt.Sprintf("step-%d", step.ID)
		}
		stepIDToOpID[step.ID] = opID
	}

	for _, step := range plan.Steps {
		op := Operation{
			OpID:        stepIDToOpID[step.ID],
			Description: step.Description,
			Resources:   []string{step.ResourceID},
			TargetOS:    goos,
			TargetArch:  arch,
			ProviderID:  step.Provider,
			SourcePath:  step.SourcePath,
			TargetPath:  step.TargetPath,
			Ownership:   "user",
			Risk:        RiskLow,
		}

		if step.IsDestructive {
			op.Risk = RiskHigh
		} else if step.IsPrivileged {
			op.Risk = RiskMedium
		}

		for _, dep := range step.DependsOn {
			if targetOpID, ok := stepIDToOpID[dep]; ok {
				op.DependsOn = append(op.DependsOn, targetOpID)
			} else {
				return nil, fmt.Errorf("step %d depends on unknown step ID %d", step.ID, dep)
			}
		}

		switch step.Action {
		case domain.ActionInstall:
			op.Kind = OpPackageInstall
			op.SideEffect = SideEffectSystemPackage
			op.Argv = resolvePackageInstallArgv(step.Provider, step.ResourceID, step.IsPrivileged)
			if len(op.Argv) == 0 {
				op.Kind = OpAnxRuntimeDelegation
				op.Argv = []string{"anx", "apply", "--requested", step.ResourceID}
			}

		case domain.ActionUninstall:
			op.Kind = OpPackageRemove
			op.SideEffect = SideEffectSystemPackage
			op.Argv = resolvePackageRemoveArgv(step.Provider, step.ResourceID, step.IsPrivileged)

		case domain.ActionSync:
			op.SideEffect = SideEffectFilesystem
			if step.Strategy == "symlink" && step.SourcePath != "" && step.TargetPath != "" {
				op.Kind = OpCreateSymlinkOwned
			} else if step.SourcePath != "" && step.TargetPath != "" {
				op.Kind = OpCopyAtomic
			} else {
				op.Kind = OpAnxRuntimeDelegation
				op.Argv = []string{"anx", "sync"}
			}

		default:
			op.Kind = OpAnxRuntimeDelegation
			op.SideEffect = SideEffectProcess
			op.Argv = []string{"anx", string(step.Action), step.ResourceID}
		}

		ir.Operations = append(ir.Operations, op)
	}

	return ir, nil
}

func resolvePackageInstallArgv(provider, pkg string, privileged bool) []string {
	prefix := []string{}
	if privileged {
		prefix = []string{"sudo"}
	}

	switch strings.ToLower(provider) {
	case "apt":
		return append(prefix, "apt-get", "install", "-y", pkg)
	case "dnf":
		return append(prefix, "dnf", "install", "-y", pkg)
	case "pacman":
		return append(prefix, "pacman", "-S", "--noconfirm", pkg)
	case "apk":
		return append(prefix, "apk", "add", pkg)
	case "brew":
		return []string{"brew", "install", pkg}
	default:
		return nil
	}
}

func resolvePackageRemoveArgv(provider, pkg string, privileged bool) []string {
	prefix := []string{}
	if privileged {
		prefix = []string{"sudo"}
	}

	switch strings.ToLower(provider) {
	case "apt":
		return append(prefix, "apt-get", "remove", "-y", pkg)
	case "dnf":
		return append(prefix, "dnf", "remove", "-y", pkg)
	case "pacman":
		return append(prefix, "pacman", "-R", "--noconfirm", pkg)
	case "apk":
		return append(prefix, "apk", "del", pkg)
	case "brew":
		return []string{"brew", "uninstall", pkg}
	default:
		return nil
	}
}
