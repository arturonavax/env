package cli

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/arturonavax/env/internal/capability"
	"github.com/arturonavax/env/internal/domain"
)

func newLsCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:                "ls [flags] [path...]",
		Short:              "List directory contents via capability abstraction (eza with ls fallback)",
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := capability.NewRegistry()
			if appCtx.ControlPlane != nil && appCtx.ControlPlane.Capabilities() != nil {
				reg = appCtx.ControlPlane.Capabilities()
			}

			code, err := reg.Execute(cmd.Context(), "files.list", args, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr())
			if err != nil {
				return domain.NewExitError(code, err)
			}
			if code != 0 {
				return domain.NewExitError(code, nil)
			}
			return nil
		},
	}
}

func newWhatCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "what <capability-or-alias>",
		Short: "Explain which provider and binary resolves an abstract capability",
		Example: `  anx what ls
  anx what files.list
  anx what grep`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			reg := capability.NewRegistry()
			if appCtx.ControlPlane != nil && appCtx.ControlPlane.Capabilities() != nil {
				reg = appCtx.ControlPlane.Capabilities()
			}

			res, err := reg.Resolve(args[0])
			if err != nil {
				return domain.NewExitError(domain.ExitUsageError, err)
			}

			st := styles(appCtx)
			return emit(cmd, appCtx, res, func(w io.Writer) {
				fmt.Fprintln(w, st.Header.Render(fmt.Sprintf("Capability Resolution: %s (alias: %s)", res.Definition.ID, res.Definition.Alias)))
				fmt.Fprintf(w, "  • Description: %s\n", res.Definition.Description)
				fmt.Fprintf(w, "  • Primary:     %s\n", res.Definition.Primary)
				fmt.Fprintf(w, "  • Fallback:    %s\n", res.Definition.Fallback)
				if res.IsFallback {
					fmt.Fprintf(w, "  • Status:      %s\n", st.Warning.Render("Active Fallback: "+res.Selected))
				} else {
					fmt.Fprintf(w, "  • Status:      %s\n", st.Success.Render("Active Primary: "+res.Selected))
				}
				fmt.Fprintf(w, "  • Resolution:  %s\n", res.Explanation)
			})
		},
	}
}
