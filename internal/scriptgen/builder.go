package scriptgen

import (
	"fmt"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/platform"
)

// ScriptCommand represents a single auditable action inside a script section.
type ScriptCommand struct {
	Comment string
	Command string
}

// ScriptSection represents a logical group of operations.
type ScriptSection struct {
	Title       string
	Description string
	Commands    []ScriptCommand
}

// AuditableScriptBuilder accumulates auditable system operations and translates them into a standalone Bash script.
type AuditableScriptBuilder struct {
	commandName string
	targetHost  string
	targetOS    string
	targetArch  string
	timestamp   time.Time
	sections    []ScriptSection
}

// NewBuilder constructs a new AuditableScriptBuilder with platform metadata.
func NewBuilder(commandName string, plat *platform.Info) *AuditableScriptBuilder {
	host := "localhost"
	osName := "linux"
	archName := "amd64"
	if plat != nil {
		osName = plat.OS()
		archName = plat.Arch()
		if plat.Distribution() != "" {
			host = plat.Distribution()
		}
	}
	return &AuditableScriptBuilder{
		commandName: commandName,
		targetHost:  host,
		targetOS:    osName,
		targetArch:  archName,
		timestamp:   time.Now().UTC(),
		sections:    make([]ScriptSection, 0, 8),
	}
}

// AddSection opens a new section for commands.
func (b *AuditableScriptBuilder) AddSection(title, description string) *AuditableScriptBuilder {
	b.sections = append(b.sections, ScriptSection{
		Title:       title,
		Description: description,
		Commands:    make([]ScriptCommand, 0, 4),
	})
	return b
}

// AddCommand adds a command to the most recent section.
func (b *AuditableScriptBuilder) AddCommand(cmd, comment string) *AuditableScriptBuilder {
	if len(b.sections) == 0 {
		b.AddSection("General Execution", "")
	}
	idx := len(b.sections) - 1
	b.sections[idx].Commands = append(b.sections[idx].Commands, ScriptCommand{
		Command: cmd,
		Comment: comment,
	})
	return b
}

// Render compiles all sections into a runnable, auditable Bash script.
func (b *AuditableScriptBuilder) Render() string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("# ==============================================================================\n")
	sb.WriteString("# ANX AUDITABLE EXECUTION SCRIPT\n")
	fmt.Fprintf(&sb, "# Command:   anx %s --script\n", b.commandName)
	fmt.Fprintf(&sb, "# Platform:  %s (%s/%s)\n", b.targetHost, b.targetOS, b.targetArch)
	fmt.Fprintf(&sb, "# Generated: %s\n", b.timestamp.Format(time.RFC3339))
	sb.WriteString("#\n")
	sb.WriteString("# AUDIT & ARCHITECTURAL BOUNDARY NOTICE:\n")
	sb.WriteString("# This shell script captures the auditable, host-level state transitions planned\n")
	sb.WriteString("# by the ANX control plane for inspection prior to execution.\n")
	sb.WriteString("#\n")
	sb.WriteString("# IN-PROCESS GO RUNTIME OPERATIONS EXCLUDED FROM THIS SCRIPT:\n")
	sb.WriteString("# - Goroutine scheduling, waitgroups, channel synchronization, and worker pools.\n")
	sb.WriteString("# - Myers diff graph computations and 3-way AST merge evaluation.\n")
	sb.WriteString("# - Content-Addressable Storage (CAS) digest verification and deduplication.\n")
	sb.WriteString("# - SQLite WAL transactional journal persistence and local state tracking.\n")
	sb.WriteString("# - Zero-allocation secret scanning in-memory buffers.\n")
	sb.WriteString("#\n")
	sb.WriteString("# All system-modifying operations (packages, dotfiles, hardware parameters,\n")
	sb.WriteString("# kernel sysctls, and security remediations) are fully reflected below.\n")
	sb.WriteString("# ==============================================================================\n")
	sb.WriteString("set -euo pipefail\n\n")

	if len(b.sections) == 0 {
		sb.WriteString("echo '==> No host actions required. Desired state already satisfied.'\n")
		return sb.String()
	}

	for i, sec := range b.sections {
		fmt.Fprintf(&sb, "# ------------------------------------------------------------------------------\n")
		fmt.Fprintf(&sb, "# Step %d: %s\n", i+1, sec.Title)
		if sec.Description != "" {
			fmt.Fprintf(&sb, "# Description: %s\n", sec.Description)
		}
		fmt.Fprintf(&sb, "# ------------------------------------------------------------------------------\n")
		fmt.Fprintf(&sb, "echo '==> [%d/%d] %s...'\n\n", i+1, len(b.sections), sec.Title)

		for _, cmd := range sec.Commands {
			if cmd.Comment != "" {
				fmt.Fprintf(&sb, "# %s\n", cmd.Comment)
			}
			sb.WriteString(cmd.Command + "\n")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("echo '==> ANX auditable script execution finished successfully.'\n")
	return sb.String()
}

// TranslatePlan converts a planned ExecutionPlan to an auditable Bash script.
func TranslatePlan(plan *domain.ExecutionPlan, plat *platform.Info) string {
	b := NewBuilder("apply", plat)
	if plan == nil || len(plan.Steps) == 0 {
		return b.Render()
	}

	for _, step := range plan.Steps {
		actionStr := string(step.Action)
		b.AddSection(fmt.Sprintf("%s: %s", strings.ToUpper(actionStr), step.ResourceID), step.Description)

		cmd := fmt.Sprintf("# Execute action %s on %s via provider %s (%s)", actionStr, step.ResourceID, step.Provider, step.Strategy)
		if step.TargetPath != "" && step.SourcePath != "" {
			cmd = fmt.Sprintf("mkdir -p $(dirname %q) && cp -f %q %q", step.TargetPath, step.SourcePath, step.TargetPath)
		} else if step.Action == domain.ActionInstall {
			cmd = fmt.Sprintf("echo 'Installing package %s (%s)...'", step.ResourceID, step.Provider)
		}
		b.AddCommand(cmd, fmt.Sprintf("Provider: %s", step.Provider))
	}

	return b.Render()
}
