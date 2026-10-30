package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/arturonavax/env/internal/ai/config"
	"github.com/arturonavax/env/internal/ai/wizard"
	"github.com/spf13/cobra"
)

func newConfigCmd(rt *runtimeEnv) *cobra.Command {
	var interactive bool
	var forceInit bool

	cmd := &cobra.Command{
		Use:       "config [show|path|init|wizard]",
		Short:     "Show, locate, initialize, or interactively configure aitools",
		GroupID:   "system",
		ValidArgs: []string{"show", "path", "init", "wizard", "interactive"},
		Args:      cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "show"
			if len(args) == 1 {
				action = strings.ToLower(strings.TrimSpace(args[0]))
			}
			if interactive || action == "wizard" || action == "interactive" {
				return wizard.Run(cmd.Context(), cmd.OutOrStdout(), rt.cfg, rt.configPath, rt.registry, rt.isInteractive(cmd))
			}

			out := cmd.OutOrStdout()
			switch action {
			case "path":
				fmt.Fprintln(out, rt.configPath)
				return nil

			case "init":
				if err := config.InitConfig(rt.configPath, forceInit); err != nil {
					return err
				}
				fmt.Fprintf(out, "Configuration initialized at %s\n", rt.configPath)
				return nil

			case "show":
				raw, err := os.ReadFile(rt.configPath)
				if err == nil {
					fmt.Fprintln(out, strings.TrimSpace(string(raw)))
					return nil
				}
				if !errors.Is(err, os.ErrNotExist) {
					return err
				}
				encoded, mErr := json.MarshalIndent(rt.cfg, "", "  ")
				if mErr != nil {
					return mErr
				}
				fmt.Fprintln(out, string(encoded))
				return nil

			default:
				return fmt.Errorf("unknown config action %q (valid: show, path, init, wizard)", action)
			}
		},
	}

	cmd.Flags().BoolVar(&interactive, "interactive", false, "Launch interactive configuration wizard")
	cmd.Flags().BoolVarP(&forceInit, "force", "f", false, "Overwrite existing configuration file on 'config init'")
	return cmd
}

func newSetupCmd(rt *runtimeEnv) *cobra.Command {
	return &cobra.Command{
		Use:     "setup",
		Short:   "Launch the interactive Bubble Tea configuration wizard",
		GroupID: "system",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return wizard.Run(cmd.Context(), cmd.OutOrStdout(), rt.cfg, rt.configPath, rt.registry, rt.isInteractive(cmd))
		},
	}
}
