package cli

import (
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"
)

func newJobsCmd(appCtx *AppContext) *cobra.Command {
	jobsCmd := &cobra.Command{
		Use:     "jobs",
		Short:   "Inspect and manage durable background execution jobs and events",
		GroupID: "meta",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runJobsList(cmd, appCtx, 20)
		},
	}

	jobsCmd.AddCommand(newJobsListCmd(appCtx))
	jobsCmd.AddCommand(newJobsShowCmd(appCtx))
	jobsCmd.AddCommand(newJobsEventsCmd(appCtx))
	jobsCmd.AddCommand(newJobsCancelCmd(appCtx))

	return jobsCmd
}

func newJobsListCmd(appCtx *AppContext) *cobra.Command {
	var limit int
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List recent execution jobs",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runJobsList(cmd, appCtx, limit)
		},
	}
	cmd.Flags().IntVar(&limit, "limit", 20, "Maximum number of jobs to list")
	return cmd
}

func runJobsList(cmd *cobra.Command, appCtx *AppContext, limit int) error {
	if appCtx.ControlPlane == nil {
		return fmt.Errorf("control plane service is unavailable")
	}

	jobs, err := appCtx.ControlPlane.ListJobs(cmd.Context(), limit)
	if err != nil {
		return fmt.Errorf("listing jobs: %w", err)
	}

	return emit(cmd, appCtx, jobs, func(w io.Writer) {
		if len(jobs) == 0 {
			fmt.Fprintln(w, "No execution jobs recorded yet.")
			return
		}

		fmt.Fprintln(w, "Recent Execution Jobs")
		fmt.Fprintln(w, "=====================")
		for _, j := range jobs {
			finStr := "-"
			if j.FinishedAt != nil {
				finStr = j.FinishedAt.Format("15:04:05")
			}
			fmt.Fprintf(w, "• %-16s [%-10s] Queued: %s  Finished: %s\n",
				j.JobID,
				j.State,
				j.QueuedAt.Format("2006-01-02 15:04:05"),
				finStr,
			)
		}
	})
}

func newJobsShowCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "show <job-id>",
		Short: "Show details for a specific execution job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if appCtx.ControlPlane == nil {
				return fmt.Errorf("control plane service is unavailable")
			}

			jobID := args[0]
			job, ok, err := appCtx.ControlPlane.GetJob(cmd.Context(), jobID)
			if err != nil {
				return fmt.Errorf("retrieving job: %w", err)
			}
			if !ok {
				return fmt.Errorf("job %q not found", jobID)
			}

			return emit(cmd, appCtx, job, func(w io.Writer) {
				fmt.Fprintf(w, "Job: %s\n", job.JobID)
				fmt.Fprintf(w, "Plan ID: %s\n", job.PlanID)
				fmt.Fprintf(w, "State: %s\n", job.State)
				fmt.Fprintf(w, "Queued At: %s\n", job.QueuedAt.Format("2006-01-02 15:04:05 UTC"))
				if job.StartedAt != nil {
					fmt.Fprintf(w, "Started At: %s\n", job.StartedAt.Format("2006-01-02 15:04:05 UTC"))
				}
				if job.FinishedAt != nil {
					fmt.Fprintf(w, "Finished At: %s\n", job.FinishedAt.Format("2006-01-02 15:04:05 UTC"))
					fmt.Fprintf(w, "Exit Code: %d\n", job.ExitErrorCode)
				}
			})
		},
	}
}

func newJobsEventsCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "events <job-id>",
		Short: "Display chronological audit events for a job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if appCtx.ControlPlane == nil {
				return fmt.Errorf("control plane service is unavailable")
			}

			jobID := args[0]
			events, err := appCtx.ControlPlane.GetJobEvents(cmd.Context(), jobID)
			if err != nil {
				return fmt.Errorf("retrieving events: %w", err)
			}

			return emit(cmd, appCtx, events, func(w io.Writer) {
				if len(events) == 0 {
					fmt.Fprintf(w, "No events recorded for job %s.\n", jobID)
					return
				}

				fmt.Fprintf(w, "Events for Job %s\n", jobID)
				fmt.Fprintln(w, "====================")
				for _, ev := range events {
					fmt.Fprintf(w, "[Seq %d] %s  %-15s %s\n",
						ev.Seq,
						ev.Timestamp.Format("15:04:05"),
						ev.EventType,
						ev.PayloadJSON,
					)
				}
			})
		},
	}
}

func newJobsCancelCmd(appCtx *AppContext) *cobra.Command {
	return &cobra.Command{
		Use:   "cancel <job-id>",
		Short: "Cancel an active or running job",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if appCtx.ControlPlane == nil {
				return fmt.Errorf("control plane service is unavailable")
			}

			jobID := args[0]
			if err := appCtx.ControlPlane.Cancel(cmd.Context(), jobID); err != nil {
				return fmt.Errorf("cancelling job %s: %w", jobID, err)
			}

			return emit(cmd, appCtx, map[string]string{
				"job_id": jobID,
				"status": "cancelled",
			}, func(w io.Writer) {
				fmt.Fprintf(w, "Job %s cancelled successfully.\n", jobID)
			})
		},
	}
}

// Suppress unused strconv warning if any
var _ = strconv.Itoa
