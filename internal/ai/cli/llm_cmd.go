package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/arturonavax/env/internal/ai/contextbuilder"
	"github.com/arturonavax/env/internal/ai/harness"
	"github.com/spf13/cobra"
)

func newLLMCmd(rt *runtimeEnv) *cobra.Command {
	var (
		backend         string
		model           string
		effort          string
		files           []string
		dirs            []string
		mode            string
		outputFormat    string
		outputFile      string
		systemPrompt    string
		timeoutSec      int
		noCleanMarkdown bool
		listBackends    bool
	)

	cmd := &cobra.Command{
		Use:     "llm [prompt...]",
		Short:   "Execute non-interactive prompt queries through any AI CLI harness",
		GroupID: "workflows",
		Args:    cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if listBackends {
				return renderBackends(cmd, rt, false)
			}

			var stdinData string
			if !rt.isStdinTTY(cmd) {
				b, err := io.ReadAll(cmd.InOrStdin())
				if err == nil {
					stdinData = strings.TrimSpace(string(b))
				}
			}

			cliPrompt := strings.TrimSpace(strings.Join(args, " "))
			ctxAttachment, err := contextbuilder.Build(contextbuilder.Options{
				Files: files,
				Dirs:  dirs,
			})
			if err != nil {
				return fmt.Errorf("building context attachments: %w", err)
			}

			parts := make([]string, 0, 3)
			if ctxAttachment != "" {
				parts = append(parts, ctxAttachment)
			}
			if stdinData != "" {
				if ctxAttachment != "" || cliPrompt != "" {
					parts = append(parts, "=== Standard Input ===\n"+stdinData)
				} else {
					parts = append(parts, stdinData)
				}
			}
			if cliPrompt != "" {
				if ctxAttachment != "" || stdinData != "" {
					parts = append(parts, "=== Instruction ===\n"+cliPrompt)
				} else {
					parts = append(parts, cliPrompt)
				}
			}

			finalPrompt := strings.TrimSpace(strings.Join(parts, "\n\n"))
			if finalPrompt == "" {
				return errors.New("no prompt, stdin data, or file/dir attachments provided")
			}

			effFormat := strings.ToLower(strings.TrimSpace(outputFormat))
			if effFormat == "" {
				effFormat = rt.cfg.LLM.DefaultOutputFormat
			}
			cleanMD := !noCleanMarkdown
			var timeout time.Duration
			if timeoutSec > 0 {
				timeout = time.Duration(timeoutSec) * time.Second
			}

			resp, usedBackend, qErr := rt.registry.Query(cmd.Context(), backend, harness.Request{
				Prompt:        finalPrompt,
				SystemPrompt:  systemPrompt,
				Model:         model,
				Effort:        effort,
				Mode:          mode,
				OutputFormat:  effFormat,
				Timeout:       timeout,
				CleanMarkdown: &cleanMD,
			})
			if qErr != nil {
				if effFormat == "json" {
					errEnv := map[string]string{
						"backend": usedBackend,
						"status":  "error",
						"error":   qErr.Error(),
					}
					if b, mErr := json.MarshalIndent(errEnv, "", "  "); mErr == nil {
						fmt.Fprintln(cmd.OutOrStdout(), string(b))
					}
				}
				return qErr
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, []byte(resp+"\n"), 0o644); err != nil {
					return fmt.Errorf("writing output file %q: %w", outputFile, err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Output written to %s\n", outputFile)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), resp)
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&backend, "backend", "b", "", "AI CLI backend override (agy, claude, codex, opencode, pi, llm, ollama)")
	f.StringVarP(&model, "model", "m", "", "Model identifier override")
	f.StringVarP(&effort, "effort", "e", "", "Reasoning effort level (low|medium|high|xhigh|max)")
	f.StringArrayVarP(&files, "file", "f", nil, "Attach file content to context (repeatable)")
	f.StringArrayVarP(&dirs, "dir", "d", nil, "Attach directory tree and file contents to context (repeatable)")
	f.StringVar(&mode, "mode", "", "Execution mode (chat|ask|action|simple-agent|agent)")
	f.StringVarP(&outputFormat, "output-format", "o", "", "Output format (text|json|raw)")
	f.StringVarP(&outputFile, "output-file", "O", "", "Write LLM response directly to file")
	f.StringVarP(&systemPrompt, "system", "s", "", "System prompt or context preamble")
	f.IntVarP(&timeoutSec, "timeout", "t", 0, "Command timeout in seconds")
	f.BoolVar(&noCleanMarkdown, "no-clean-markdown", false, "Preserve outer markdown code fences in response")
	f.BoolVar(&listBackends, "list-backends", false, "List available AI CLI harnesses")

	return cmd
}
