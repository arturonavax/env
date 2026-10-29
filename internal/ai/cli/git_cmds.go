package cli

import (
	"fmt"
	"os"

	"github.com/arturonavax/env/internal/ai/gitflow"
	"github.com/spf13/cobra"
)

func (rt *runtimeEnv) workDir() string {
	if rt.opts.workDir != "" {
		return rt.opts.workDir
	}
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

func newCommitCmd(rt *runtimeEnv) *cobra.Command {
	var (
		opts         gitflow.CommitOptions
		listBackends bool
		legacyJiraID string
	)

	cmd := &cobra.Command{
		Use:     "commit",
		Aliases: []string{"cm"},
		Short:   "Generate Conventional Git Commit messages from the diff using AI",
		GroupID: "workflows",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listBackends {
				return renderBackends(cmd, rt, false)
			}
			if opts.TaskID == "" {
				opts.TaskID = legacyJiraID
			}
			opts.WorkDir = rt.workDir()
			opts.Interactive = rt.isInteractive(cmd)
			opts.In = cmd.InOrStdin()
			opts.Out = cmd.OutOrStdout()
			opts.ErrOut = cmd.ErrOrStderr()

			_, err := rt.gitSvc.RunCommit(cmd.Context(), opts)
			return err
		},
	}

	f := cmd.Flags()
	f.BoolVarP(&opts.StageAll, "all", "a", false, "Stage all tracked modified files before diffing")
	f.StringVarP(&opts.Backend, "backend", "b", "", "AI CLI backend (agy, claude, codex, llm, opencode, pi, ollama, or custom)")
	f.StringVarP(&opts.Model, "model", "m", "", "Model identifier override")
	f.StringVarP(&opts.Template, "template", "t", "", "Template name (conventional, task, task-scope, detailed, repo) or custom string/path")
	f.StringVarP(&opts.TaskID, "task", "k", "", "Task / issue ticket ID override (e.g. PROJ-1234, #42)")
	f.StringVar(&opts.TaskID, "ticket", "", "Alias of --task")
	f.StringVar(&opts.TaskID, "issue", "", "Alias of --task")
	f.StringVarP(&legacyJiraID, "jira", "j", "", "Deprecated alias of --task")
	_ = f.MarkHidden("jira")
	f.StringVarP(&opts.Scope, "scope", "s", "", "Optional commit scope override")
	f.BoolVar(&opts.ListTemplates, "list-templates", false, "List all available commit templates")
	f.BoolVar(&listBackends, "list-backends", false, "List all available AI CLI harnesses")
	f.BoolVarP(&opts.DryRun, "dry-run", "d", false, "Preview message without committing")
	return cmd
}

func newPRCmd(rt *runtimeEnv) *cobra.Command {
	var (
		opts         gitflow.PROptions
		listBackends bool
	)

	cmd := &cobra.Command{
		Use:     "pr",
		Short:   "Generate Pull Request Markdown descriptions from branch diff and commits using AI",
		GroupID: "workflows",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if listBackends {
				return renderBackends(cmd, rt, false)
			}
			opts.WorkDir = rt.workDir()
			opts.Interactive = rt.isInteractive(cmd) && !opts.PrintRaw
			opts.In = cmd.InOrStdin()
			opts.Out = cmd.OutOrStdout()
			opts.ErrOut = cmd.ErrOrStderr()

			_, err := rt.gitSvc.RunPR(cmd.Context(), opts)
			return err
		},
	}

	f := cmd.Flags()
	f.StringVarP(&opts.Base, "base", "B", "", "Target base branch (main, master, develop)")
	f.StringVarP(&opts.Head, "head", "H", "", "Head branch or ref (defaults to HEAD)")
	f.StringVarP(&opts.Backend, "backend", "b", "", "AI CLI backend")
	f.StringVarP(&opts.Model, "model", "m", "", "Model identifier override")
	f.StringVarP(&opts.Template, "template", "t", "", "PR template name (standard, task, minimal, detailed, repo) or file/string")
	f.StringVarP(&opts.TaskID, "task", "k", "", "Task / issue ticket ID override")
	f.StringVar(&opts.TaskID, "ticket", "", "Alias of --task")
	f.StringVar(&opts.TaskID, "issue", "", "Alias of --task")
	f.StringVarP(&opts.OutputFile, "output", "o", "", "Write generated Markdown to file")
	f.BoolVarP(&opts.CopyClipboard, "copy", "c", false, "Copy generated description to clipboard")
	f.BoolVarP(&opts.PrintRaw, "print", "p", false, "Print raw Markdown to stdout without interactive UI")
	f.BoolVarP(&opts.DryRun, "dry-run", "d", false, "Preview description without prompting for actions")
	f.BoolVar(&opts.ListTemplates, "list-templates", false, "List available PR templates")
	f.BoolVar(&listBackends, "list-backends", false, "List all available AI CLI harnesses")
	return cmd
}

func newRunCmd(rt *runtimeEnv) *cobra.Command {
	return &cobra.Command{
		Use:                   "run <command-name> [args...]",
		Short:                 "Execute a declarative custom AI command/pipeline from config.json",
		GroupID:               "workflows",
		Args:                  cobra.MinimumNArgs(1),
		DisableFlagsInUseLine: true,
		ValidArgsFunction: func(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
			if len(args) > 0 || rt.init() != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			names := []string{}
			for _, c := range rt.pipeRunner.ListCustomCommands() {
				names = append(names, c.Name)
			}
			return names, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := rt.pipeRunner.Execute(cmd.Context(), pipelineOpts(rt, cmd, args))
			return err
		},
	}
}

func newCommandsCmd(rt *runtimeEnv) *cobra.Command {
	return &cobra.Command{
		Use:     "commands",
		Short:   "List all built-in and declarative custom AI commands",
		GroupID: "workflows",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "  %-16s %-9s %-12s %-10s %s\n", "COMMAND", "KIND", "INPUT", "BACKEND", "DESCRIPTION")
			for _, c := range rt.pipeRunner.ListCustomCommands() {
				fmt.Fprintf(out, "  %-16s %-9s %-12s %-10s %s\n", c.Name, c.Kind, c.InputType, c.Backend, c.Description)
			}
			return nil
		},
	}
}
