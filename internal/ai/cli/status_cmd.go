package cli

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"
)

type diagnosticEntry struct {
	Binary      string `json:"binary"`
	Description string `json:"description"`
	Available   bool   `json:"available"`
	Path        string `json:"path,omitempty"`
}

func newStatusCmd(rt *runtimeEnv) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Check availability and paths of AI CLI harnesses and companion tools",
		GroupID: "system",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			lookPath := rt.opts.lookPath
			if lookPath == nil {
				lookPath = exec.LookPath
			}

			probes := []struct {
				bin  string
				desc string
			}{
				{bin: "agy", desc: "Google Antigravity CLI (Gemini, Claude, GPT)"},
				{bin: "claude", desc: "Anthropic Claude Code CLI"},
				{bin: "codex", desc: "OpenAI Codex CLI"},
				{bin: "opencode", desc: "OpenCode CLI"},
				{bin: "pi", desc: "Inflection Pi CLI"},
				{bin: "llm", desc: "Datasette LLM CLI"},
				{bin: "ollama", desc: "Ollama Local LLM Runner"},
				{bin: "rtk", desc: "Rust Token Killer proxy"},
				{bin: "git", desc: "Git version control"},
				{bin: "notify-send", desc: "Desktop notifications"},
				{bin: "rtcwake", desc: "Hardware RTC wakeup alarm"},
			}

			entries := make([]diagnosticEntry, 0, len(probes))
			for _, p := range probes {
				resolved, err := lookPath(p.bin)
				entries = append(entries, diagnosticEntry{
					Binary:      p.bin,
					Description: p.desc,
					Available:   err == nil,
					Path:        resolved,
				})
			}

			out := cmd.OutOrStdout()
			if asJSON {
				b, err := json.MarshalIndent(entries, "", "  ")
				if err != nil {
					return err
				}
				fmt.Fprintln(out, string(b))
				return nil
			}

			fmt.Fprintln(out, "ANX AI · Ecosystem & CLI Status Diagnostics")
			fmt.Fprintln(out, strings.Repeat("─", 78))
			for _, e := range entries {
				if e.Available {
					fmt.Fprintf(out, "  [ok] %-14s %-42s (%s)\n", e.Binary, e.Description, e.Path)
				} else {
					fmt.Fprintf(out, "  [--] %-14s %-42s (not found in PATH)\n", e.Binary, e.Description)
				}
			}
			fmt.Fprintln(out, strings.Repeat("─", 78))
			return nil
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output status diagnostics as JSON")
	return cmd
}

func newBackendsCmd(rt *runtimeEnv) *cobra.Command {
	var asJSON bool

	cmd := &cobra.Command{
		Use:     "backends",
		Short:   "List all registered AI CLI harnesses and I/O execution strategies",
		GroupID: "system",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return renderBackends(cmd, rt, asJSON)
		},
	}

	cmd.Flags().BoolVar(&asJSON, "json", false, "Output backends table as JSON")
	return cmd
}

func renderBackends(cmd *cobra.Command, rt *runtimeEnv, asJSON bool) error {
	statuses := rt.registry.ListStatuses()
	out := cmd.OutOrStdout()

	if asJSON {
		b, err := json.MarshalIndent(statuses, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(out, string(b))
		return nil
	}

	fmt.Fprintln(out, "ANX AI · Available AI Harness Backends & Execution Strategies")
	fmt.Fprintln(out, strings.Repeat("─", 86))
	fmt.Fprintf(out, "  %-16s %-14s %-11s %-11s %s\n", "BACKEND", "STATUS", "INPUT", "OUTPUT", "DESCRIPTION")
	fmt.Fprintln(out, strings.Repeat("─", 86))
	for _, s := range statuses {
		name := s.Name
		if s.IsDefault {
			name += " *"
		}
		badge := "not found"
		if s.Available {
			badge = "installed"
		}
		fmt.Fprintf(out, "  %-16s %-14s %-11s %-11s %s\n", name, badge, s.InputMode, s.OutputMode, s.Description)
	}
	fmt.Fprintln(out, strings.Repeat("─", 86))
	fmt.Fprintln(out, "(* = active default backend)")
	return nil
}
