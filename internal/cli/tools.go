package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/charmbracelet/x/term"
	"github.com/spf13/cobra"

	aicli "github.com/arturonavax/env/internal/ai/cli"
	"github.com/arturonavax/env/internal/domain"
	"github.com/arturonavax/env/internal/theme"
)

func toolsCommands(appCtx *AppContext) []*cobra.Command {
	toolsCmd := newToolsCmd(appCtx)
	// Top-level aliases for backward compatibility and fast access:
	aiAlias := newAICmd(appCtx)
	aiAlias.GroupID = "tools"

	llmAlias := newToolsLLMShortcut(appCtx)
	llmAlias.Use = "llm [flags] [prompt]"
	llmAlias.GroupID = "tools"

	return []*cobra.Command{toolsCmd, aiAlias, llmAlias}
}

func toolCompletion(appCtx *AppContext) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 || appCtx.ToolsSvc == nil {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		var ids []string
		for _, m := range appCtx.ToolsSvc.Registry().List() {
			ids = append(ids, m.ID+"\t"+m.Name)
		}
		return ids, cobra.ShellCompDirectiveNoFileComp
	}
}

func newToolsCmd(appCtx *AppContext) *cobra.Command {
	toolsCmd := &cobra.Command{
		Use:   "tools",
		Short: "Manage, execute, and update decoupled developer tools and AI suite",
		Long: `The tools subsystem orchestrates personal and decoupled developer tools (Go, Rust, Python, Bash, Node.js).
Tools can live as standalone external projects in ~/.local/bin or PATH without requiring anx to be recompiled,
while built-in engines (such as aitools) provide zero-dependency standalone fallbacks.`,
		GroupID: "tools",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return runToolsList(cmd, appCtx)
			}
			return cmd.Help()
		},
	}

	// 1. Tool Lifecycle & Inspection Subcommands
	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all registered tools, runtime languages, and active execution targets",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runToolsList(cmd, appCtx)
		},
	}

	infoCmd := &cobra.Command{
		Use:               "info <tool>",
		Short:             "Show detailed configuration, path, and update strategy for a tool",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: toolCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			return runToolsInfo(cmd, appCtx, args[0])
		},
	}

	runCmd := &cobra.Command{
		Use:                "run <tool> [args...]",
		Short:              "Run any decoupled tool forwarding arguments, environment, and I/O streams",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		ValidArgsFunction:  toolCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			toolID := args[0]
			toolArgs := args[1:]
			return appCtx.ToolsSvc.Run(cmd.Context(), toolID, toolArgs, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}

	updateCmd := &cobra.Command{
		Use:               "update [tool]",
		Short:             "Check and apply updates for tools via their native update strategy",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: toolCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return runToolsUpdate(cmd, appCtx, target)
		},
	}

	upgradeCmd := &cobra.Command{
		Use:               "upgrade [tool]",
		Short:             "Alias for 'anx tools update'",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: toolCompletion(appCtx),
		RunE: func(cmd *cobra.Command, args []string) error {
			target := ""
			if len(args) == 1 {
				target = args[0]
			}
			return runToolsUpdate(cmd, appCtx, target)
		},
	}

	toolsCmd.AddCommand(listCmd, infoCmd, runCmd, updateCmd, upgradeCmd)

	// 2. Full AI Productivity Suite Subcommand (`anx tools ai ...`)
	aiSubCmd := newAICmd(appCtx)
	toolsCmd.AddCommand(aiSubCmd)

	// 3. First-Class Tool Shortcuts under `anx tools`:
	toolsCmd.AddCommand(newToolsLLMShortcut(appCtx))
	toolsCmd.AddCommand(newToolsCommitShortcut(appCtx))
	toolsCmd.AddCommand(newToolsPRShortcut(appCtx))
	toolsCmd.AddCommand(newToolsQuotaShortcut(appCtx))

	// 4. Dynamically register all other configured tools as direct subcommands
	if appCtx.ToolsSvc != nil {
		for _, m := range appCtx.ToolsSvc.Registry().List() {
			if m.ID == "ai" || m.ID == "llm" || m.ID == "commit" || m.ID == "pr" || m.ID == "quota" || m.ID == "list" || m.ID == "info" || m.ID == "run" || m.ID == "update" || m.ID == "upgrade" {
				continue
			}
			toolID := m.ID
			desc := m.Description
			if desc == "" {
				desc = fmt.Sprintf("Execute %s tool", m.Name)
			}
			toolsCmd.AddCommand(&cobra.Command{
				Use:                toolID + " [args...]",
				Short:              desc,
				DisableFlagParsing: true,
				RunE: func(cmd *cobra.Command, args []string) error {
					return appCtx.ToolsSvc.Run(cmd.Context(), toolID, args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
				},
			})
		}
	}

	return toolsCmd
}

func newAICmd(appCtx *AppContext) *cobra.Command {
	return aicli.NewCommand(
		aicli.WithUse("ai"),
		aicli.WithOptimizeFunc(aiOptimizeFunc(appCtx)),
		aicli.WithPlatform(appCtx.Platform),
	)
}

func newToolsLLMShortcut(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "llm [flags] [prompt]",
		Short:              "Query AI models non-interactively (shortcut for 'anx tools ai llm')",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appCtx.ToolsSvc.Run(cmd.Context(), "ai", append([]string{"llm"}, args...), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newToolsCommitShortcut(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "commit [flags]",
		Short:              "Generate conventional commit message (shortcut for 'anx tools ai commit')",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appCtx.ToolsSvc.Run(cmd.Context(), "ai", append([]string{"commit"}, args...), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newToolsPRShortcut(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "pr [flags]",
		Short:              "Generate Pull Request description (shortcut for 'anx tools ai pr')",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appCtx.ToolsSvc.Run(cmd.Context(), "ai", append([]string{"pr"}, args...), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func newToolsQuotaShortcut(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "quota [flags]",
		Short:              "Inspect rolling AI quota and usage (shortcut for 'anx tools ai quota')",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return appCtx.ToolsSvc.Run(cmd.Context(), "ai", append([]string{"quota"}, args...), cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
		},
	}
}

func terminalWidth(cmd *cobra.Command) int {
	if f, ok := cmd.OutOrStdout().(*os.File); ok {
		if w, _, err := term.GetSize(f.Fd()); err == nil && w > 0 {
			return w
		}
	}
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		return w
	}
	return 80
}

func isExternalApart(s domain.ToolStatus) bool {
	id := strings.ToLower(s.ID)
	if id == "ai" || id == "kos" {
		return true
	}
	if s.UpdateStrategy == "package" {
		return false
	}
	return true
}

func renderAdaptiveToolRow(w io.Writer, st *theme.Styles, s domain.ToolStatus, targetWidth int) {
	lang := s.Language
	if lang == "" {
		lang = "any"
	}
	target := s.ActiveTarget
	if len(target) > targetWidth {
		if targetWidth > 6 {
			target = target[:targetWidth-3] + "..."
		} else {
			target = target[:targetWidth]
		}
	}
	statusStr := st.Success.Render("[Available]")
	if s.Version != "" && s.Version != "available" {
		statusStr = st.Success.Render(s.Version)
	}

	paddedID := fmt.Sprintf("%-14s", s.ID)
	renderedID := st.Highlight.Render(paddedID)

	fmt.Fprintf(w, "  %s %-12s %-8s %-*s %s\n",
		renderedID,
		s.Category,
		lang,
		targetWidth,
		target,
		statusStr,
	)
}

func runToolsList(cmd *cobra.Command, appCtx *AppContext) error {
	statuses := appCtx.ToolsSvc.ListFast(cmd.Context())

	var externalTools []domain.ToolStatus
	for _, s := range statuses {
		if !s.Available {
			continue
		}
		if isExternalApart(s) {
			externalTools = append(externalTools, s)
		}
	}

	termW := terminalWidth(cmd)
	if termW < 40 {
		termW = 40
	}

	// Calculate adaptive target width:
	// Fixed columns: margin(2) + ID(14) + space(1) + Category(12) + space(1) + Lang(8) + space(1) + Status(12) = 51
	targetWidth := termW - 53
	if targetWidth < 18 {
		targetWidth = 18
	} else if targetWidth > 55 {
		targetWidth = 55
	}

	divLen := min(termW-2, 14+12+8+targetWidth+14+6)
	divider := strings.Repeat("─", max(35, divLen))

	return emit(cmd, appCtx, externalTools, func(w io.Writer) {
		st := styles(appCtx)
		fmt.Fprintln(w, st.Header.Render("External Developer Tools & Suites (Herramientas Externas)"))
		if len(externalTools) == 0 {
			fmt.Fprintln(w, st.Muted.Render("  No external extensions or tools currently registered/available."))
			fmt.Fprintln(w, st.Muted.Render("  Drop custom manifests in ~/.config/anx/tools.d/ to register personal tools."))
			return
		}

		fmt.Fprintf(w, "  %-14s %-12s %-8s %-*s %s\n", "TOOL ID", "CATEGORY", "LANG", targetWidth, "ACTIVE TARGET", "STATUS")
		fmt.Fprintf(w, "  %s\n", st.Muted.Render(divider))
		for _, s := range externalTools {
			renderAdaptiveToolRow(w, st, s, targetWidth)
		}
	})
}

func runToolsInfo(cmd *cobra.Command, appCtx *AppContext, toolID string) error {
	st, err := appCtx.ToolsSvc.Get(cmd.Context(), toolID)
	if err != nil {
		return err
	}
	return emit(cmd, appCtx, st, func(w io.Writer) {
		themeStyles := styles(appCtx)
		fmt.Fprintln(w, themeStyles.Header.Render(fmt.Sprintf("Tool Info: %s (%s)", st.Name, st.ID)))
		fmt.Fprintf(w, "  • Category:        %s\n", st.Category)
		fmt.Fprintf(w, "  • Description:     %s\n", st.Description)
		fmt.Fprintf(w, "  • Language:        %s\n", st.Language)
		fmt.Fprintf(w, "  • Runtime Kind:    %s\n", st.Kind)
		fmt.Fprintf(w, "  • Executable Name: %s\n", st.Executable)
		fmt.Fprintf(w, "  • Active Target:   %s\n", st.ActiveTarget)
		fmt.Fprintf(w, "  • Installed Path:  %s\n", st.Path)
		fmt.Fprintf(w, "  • Version:         %s\n", st.Version)
		fmt.Fprintf(w, "  • Has Builtin:     %v\n", st.HasBuiltin)
		fmt.Fprintf(w, "  • Update Strategy: %s\n", st.UpdateStrategy)
		if st.RepoURL != "" {
			fmt.Fprintf(w, "  • Repository URL:  %s\n", st.RepoURL)
		}
	})
}

func runToolsUpdate(cmd *cobra.Command, appCtx *AppContext, target string) error {
	ctx := cmd.Context()
	themeStyles := styles(appCtx)

	var targets []string
	if target != "" {
		targets = []string{target}
	} else {
		for _, m := range appCtx.ToolsSvc.Registry().List() {
			targets = append(targets, m.ID)
		}
	}

	results := make([]domain.ToolUpdateResult, 0, len(targets))
	for _, id := range targets {
		res, err := appCtx.ToolsSvc.Update(ctx, id)
		if err != nil {
			res.Status = "error"
			res.Message = err.Error()
		}
		results = append(results, res)
	}

	return emit(cmd, appCtx, results, func(w io.Writer) {
		fmt.Fprintln(w, themeStyles.Header.Render("Tools Update Results"))
		for _, r := range results {
			tag := themeStyles.Success.Render("[" + strings.ToUpper(r.Status) + "]")
			if r.Status == "error" || r.Status == "failed" {
				tag = themeStyles.Error.Render("[" + strings.ToUpper(r.Status) + "]")
			} else if r.Status == "skipped" || r.Status == "up-to-date" {
				tag = themeStyles.Muted.Render("[" + strings.ToUpper(r.Status) + "]")
			}
			fmt.Fprintf(w, "  %-14s %s %s\n", r.ToolID, tag, r.Message)
		}
	})
}
