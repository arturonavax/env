package tools

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
)

// PackageUpdaterFunc signature for upgrading tools managed as anx declarative packages.
type PackageUpdaterFunc func(ctx context.Context, packageID string) error

// Updater orchestrates lifecycle updates for decoupled tools.
type Updater struct {
	registry       *Registry
	packageUpdater PackageUpdaterFunc
}

// NewUpdater instantiates a tool Updater.
func NewUpdater(r *Registry, pkgUp PackageUpdaterFunc) *Updater {
	return &Updater{
		registry:       r,
		packageUpdater: pkgUp,
	}
}

// Update attempts to update the specified tool based on its configured strategy.
func (u *Updater) Update(ctx context.Context, toolID string) (domain.ToolUpdateResult, error) {
	m, ok := u.registry.Get(toolID)
	if !ok {
		return domain.ToolUpdateResult{ToolID: toolID, Status: "error", Message: "unknown tool"}, fmt.Errorf("unknown tool %q", toolID)
	}

	st, err := u.registry.CheckStatus(ctx, toolID)
	if err != nil {
		return domain.ToolUpdateResult{ToolID: toolID, Status: "error", Message: err.Error()}, err
	}

	res := domain.ToolUpdateResult{
		ToolID: toolID,
	}

	if !st.Available && !m.FallbackBuiltin {
		res.Status = "skipped"
		res.Message = "tool is not installed on this system"
		return res, nil
	}

	switch m.UpdateStrategy {
	case "self_update":
		if len(m.UpdateCmd) == 0 {
			res.Status = "skipped"
			res.Message = "tool declares self_update but has no update_cmd"
			return res, nil
		}
		out, err := u.runCmd(ctx, m.UpdateCmd)
		if err != nil {
			res.Status = "failed"
			res.Message = fmt.Sprintf("self_update failed: %v (%s)", err, out)
			return res, err
		}
		res.Status = "updated"
		res.Updated = true
		res.Message = fmt.Sprintf("updated via %s: %s", m.UpdateCmd[0], out)
		return res, nil

	case "package":
		if u.packageUpdater != nil {
			pkgID := toolID
			if m.PackageID != "" {
				pkgID = m.PackageID
			}
			if err := u.packageUpdater(ctx, pkgID); err != nil {
				res.Status = "failed"
				res.Message = fmt.Sprintf("package upgrade failed: %v", err)
				return res, err
			}
			res.Status = "updated"
			res.Updated = true
			res.Message = "updated via anx package manager"
			return res, nil
		}
		res.Status = "skipped"
		res.Message = "package updater not available in this context"
		return res, nil

	case "anx_self":
		if st.IsExternal {
			res.Status = "external"
			res.Message = fmt.Sprintf("external tool at %s is managed independently", st.ExternalPath)
			return res, nil
		}
		res.Status = "up-to-date"
		res.Message = "tool is built into anx core; update via 'anx self update'"
		return res, nil

	case "command":
		if len(m.UpdateCmd) == 0 {
			res.Status = "skipped"
			res.Message = "no update_cmd specified"
			return res, nil
		}
		out, err := u.runCmd(ctx, m.UpdateCmd)
		if err != nil {
			res.Status = "failed"
			res.Message = fmt.Sprintf("custom update command failed: %v", err)
			return res, err
		}
		res.Status = "updated"
		res.Updated = true
		res.Message = out
		return res, nil

	default:
		res.Status = "unmanaged"
		res.Message = fmt.Sprintf("no update strategy defined for %s", toolID)
		return res, nil
	}
}

func (u *Updater) runCmd(ctx context.Context, cmdArgs []string) (string, error) {
	if len(cmdArgs) == 0 {
		return "", nil
	}
	uCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(uCtx, cmdArgs[0], cmdArgs[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := strings.TrimSpace(buf.String())
	if first := strings.Split(output, "\n"); len(first) > 0 {
		output = strings.TrimSpace(first[0])
	}
	return output, err
}
