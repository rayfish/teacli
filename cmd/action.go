// Package cmd contains CI/CD action commands
package cmd

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"text/tabwriter"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var actionCmd = &cobra.Command{
	Use:     "action",
	Aliases: []string{"actions"},
	Short:   "Manage CI/CD actions",
	Long:    `List, trigger, and manage workflow runs and jobs`,
}

var actionListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List workflow runs",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runActionList,
}

var actionGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <run-id>",
	Short: "Get workflow run details",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionGet,
}

var actionRunCmd = &cobra.Command{
	Use:   "run [<owner>/<repo>] --workflow <file>",
	Short: "Trigger a workflow dispatch",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runActionRun,
}

var actionDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <run-id>",
	Short: "Delete a workflow run",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionDelete,
}

var actionJobsCmd = &cobra.Command{
	Use:   "jobs [<owner>/<repo>] <run-id>",
	Short: "List jobs for a workflow run",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionJobs,
}

var actionJobCmd = &cobra.Command{
	Use:   "job [<owner>/<repo>] <job-id>",
	Short: "Get job details",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionJob,
}

var actionLogsCmd = &cobra.Command{
	Use:   "logs [<owner>/<repo>] <job-id>",
	Short: "Get job logs",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionLogs,
}

func init() {
	RootCmd.AddCommand(actionCmd)
	actionCmd.AddCommand(actionListCmd, actionGetCmd, actionRunCmd, actionDeleteCmd, actionJobsCmd, actionJobCmd, actionLogsCmd)

	actionListCmd.Flags().Int("page", 0, "Page number")
	actionListCmd.Flags().Int("per-page", 0, "Results per page")
	actionListCmd.Flags().String("status", "", "Filter by status (success, failure, cancelled, etc.)")
	actionListCmd.Flags().String("event", "", "Filter by event type (push, pull_request, etc.)")
	actionListCmd.Flags().String("branch", "", "Filter by branch")

	actionRunCmd.Flags().String("workflow", "", "Workflow file name or ID (required)")
	actionRunCmd.Flags().String("ref", "", "Branch or tag to run on (defaults to the repo's default branch)")
	actionRunCmd.Flags().String("inputs", "", "Workflow inputs as JSON")

	actionDeleteCmd.Flags().BoolP("confirm", "c", false, "No-op, kept for compatibility: delete never prompts")

	actionJobsCmd.Flags().Int("page", 0, "Page number")
	actionJobsCmd.Flags().Int("per-page", 0, "Results per page")
}

func runActionList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	perPage, _ := cmd.Flags().GetInt("per-page")
	status, _ := cmd.Flags().GetString("status")
	event, _ := cmd.Flags().GetString("event")
	branch, _ := cmd.Flags().GetString("branch")

	opts := gitea.ListRepoActionRunsOptions{
		ListOptions: gitea.ListOptions{
			Page:     page,
			PageSize: perPage,
		},
	}
	if status != "" {
		opts.Status = status
	}
	if event != "" {
		opts.Event = event
	}
	if branch != "" {
		opts.Branch = branch
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would list workflow runs for %s/%s\n", owner, repo)
		printer.Printf("  Page: %d, Per-page: %d\n", page, perPage)
		if opts.Status != "" {
			printer.Printf("  Status filter: %s\n", opts.Status)
		}
		if opts.Event != "" {
			printer.Printf("  Event filter: %s\n", opts.Event)
		}
		if opts.Branch != "" {
			printer.Printf("  Branch filter: %s\n", opts.Branch)
		}
		return nil
	}

	workflowRuns, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.ActionWorkflowRun, *gitea.Response, error) {
		pageOpts := opts
		pageOpts.ListOptions = lo
		res, resp, err := client.ListRepoActionRuns(owner, repo, pageOpts)
		if err != nil {
			return nil, resp, err
		}
		return res.WorkflowRuns, resp, nil
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(workflowRuns, func() error {
		if printer.Format() == "csv" {
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"id", "status", "event", "branch", "commit_sha", "started"})
			for _, run := range workflowRuns {
				started := run.StartedAt.Format(time.RFC3339)
				w.Write([]string{
					strconv.FormatInt(run.ID, 10),
					run.Status,
					run.Event,
					run.HeadBranch,
					run.HeadSha,
					started,
				})
			}
			w.Flush()
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "RUN ID\tSTATUS\tEVENT\tBRANCH\tCOMMIT SHA\tSTARTED")
		for _, run := range workflowRuns {
			commit := run.HeadSha
			if len(commit) > 40 {
				commit = commit[:40] + "..."
			}
			started := run.StartedAt.Format("2006-01-02 15:04:05")
			fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
				run.ID, run.Status, run.Event, run.HeadBranch, commit, started)
		}
		w.Flush()
		return nil
	})
}

// resolveActionRunID maps a user-supplied run identifier to the database ID the
// Gitea API requires. It accepts either the database ID directly or the run
// number shown in the Gitea web UI (the short, human-facing number). It tries
// the database ID first; on a 404 it walks the run list (newest-first) looking
// for a matching run_number. The input string is echoed in any not-found error
// so the caller's message reflects what the user typed.
func resolveActionRunID(client *gitea.Client, owner, repo, input string) (int64, error) {
	id, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return 0, errors.NewValidationError("invalid run ID",
			map[string]interface{}{"value": input})
	}

	// Fast path: input is the database ID the API expects.
	if _, resp, err := client.GetRepoActionRun(owner, repo, id); err == nil {
		return id, nil
	} else if resp == nil || resp.Response == nil || resp.StatusCode != http.StatusNotFound {
		return 0, errors.FromGitea(resp, err)
	}

	// Fallback: treat input as the run_number shown in the UI and resolve it by
	// listing. Runs come back newest-first, so run_number descends; stop once we
	// pass the target.
	const pageSize = 50
	page := 1
	for {
		runs, resp, err := client.ListRepoActionRuns(owner, repo, gitea.ListRepoActionRunsOptions{
			ListOptions: gitea.ListOptions{Page: page, PageSize: pageSize},
		})
		if err != nil {
			return 0, errors.FromGitea(resp, err)
		}
		list := runs.WorkflowRuns
		if len(list) == 0 {
			break
		}
		for _, r := range list {
			if r.RunNumber == id {
				return r.ID, nil
			}
			if r.RunNumber < id {
				return 0, errors.NewNotFoundError("workflow run", input)
			}
		}
		if len(list) < pageSize {
			break
		}
		page++
	}
	return 0, errors.NewNotFoundError("workflow run", input)
}

func runActionGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would get workflow run %s for %s/%s\n", args[1], owner, repo)
		return nil
	}

	runID, err := resolveActionRunID(client, owner, repo, args[1])
	if err != nil {
		return err
	}

	run, resp, err := client.GetRepoActionRun(owner, repo, runID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "workflow run", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(run, func() error {
		return printActionRunText(printer, run)
	})
}

func runActionRun(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	workflow, _ := cmd.Flags().GetString("workflow")
	ref, _ := cmd.Flags().GetString("ref")
	inputsJSON, _ := cmd.Flags().GetString("inputs")

	if workflow == "" {
		return errors.NewValidationError("--workflow is required",
			map[string]interface{}{"missing_flag": "--workflow"})
	}

	var inputs map[string]string
	if inputsJSON != "" {
		if err := json.Unmarshal([]byte(inputsJSON), &inputs); err != nil {
			return errors.NewValidationError("invalid inputs JSON",
				map[string]interface{}{"error": err.Error()})
		}
	}

	// The API requires a ref, so fall back to the repo's default branch.
	if ref == "" {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		repository, resp, err := client.GetRepo(owner, repo)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
		ref = repository.DefaultBranch
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would dispatch workflow '%s' for %s/%s\n", workflow, owner, repo)
		printer.Printf("  Ref: %s\n", ref)
		if len(inputs) > 0 {
			printer.Printf("  Inputs: %s\n", inputsJSON)
		}
		return nil
	}

	// Not in the SDK as of v0.23.2, so this goes straight to the REST API.
	body := map[string]any{"ref": ref}
	if len(inputs) > 0 {
		body["inputs"] = inputs
	}

	path := fmt.Sprintf("repos/%s/%s/actions/workflows/%s/dispatches",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflow))

	if _, err := apiRequest(cmd, http.MethodPost, path, body); err != nil {
		return err
	}

	return printer.Emit(map[string]any{
		"status":   "dispatched",
		"workflow": workflow,
		"ref":      ref,
		"inputs":   inputs,
	}, func() error {
		printer.Printf("Dispatched workflow '%s' on %s\n", workflow, ref)
		return nil
	})
}

func runActionDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete workflow run %s for %s/%s\n", args[1], owner, repo)
		return nil
	}

	runID, err := resolveActionRunID(client, owner, repo, args[1])
	if err != nil {
		return err
	}

	resp, err := client.DeleteRepoActionRun(owner, repo, runID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "workflow run", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "run": runID}, func() error {
		printer.Printf("Deleted workflow run %d\n", runID)
		return nil
	})
}

func runActionJobs(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	page, _ := cmd.Flags().GetInt("page")
	perPage, _ := cmd.Flags().GetInt("per-page")

	opts := gitea.ListRepoActionJobsOptions{
		ListOptions: gitea.ListOptions{
			Page:     page,
			PageSize: perPage,
		},
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would list jobs for workflow run %s in %s/%s\n", args[1], owner, repo)
		return nil
	}

	runID, err := resolveActionRunID(client, owner, repo, args[1])
	if err != nil {
		return err
	}

	jobs, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.ActionWorkflowJob, *gitea.Response, error) {
		pageOpts := opts
		pageOpts.ListOptions = lo
		res, resp, err := client.ListRepoActionRunJobs(owner, repo, runID, pageOpts)
		if err != nil {
			return nil, resp, errors.FromGiteaNotFound(resp, err, "workflow run", args[1])
		}
		return res.Jobs, resp, nil
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(jobs, func() error {
		if printer.Format() == "csv" {
			w := csv.NewWriter(os.Stdout)
			w.Write([]string{"id", "name", "status", "run_id"})
			for _, job := range jobs {
				w.Write([]string{
					strconv.FormatInt(job.ID, 10),
					job.Name,
					job.Status,
					strconv.FormatInt(job.RunID, 10),
				})
			}
			w.Flush()
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "JOB ID\tNAME\tSTATUS\tRUN ID")
		for _, job := range jobs {
			fmt.Fprintf(w, "%d\t%s\t%s\t%d\n",
				job.ID, job.Name, job.Status, job.RunID)
		}
		w.Flush()
		return nil
	})
}

func runActionJob(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	jobIDStr := args[1]
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		return errors.NewValidationError("invalid job ID",
			map[string]interface{}{"value": jobIDStr})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would get job %d for %s/%s\n", jobID, owner, repo)
		return nil
	}

	job, resp, err := client.GetRepoActionJob(owner, repo, jobID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "action job", strconv.FormatInt(jobID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(job, func() error {
		return printActionJobText(printer, job)
	})
}

func runActionLogs(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	jobIDStr := args[1]
	jobID, err := strconv.ParseInt(jobIDStr, 10, 64)
	if err != nil {
		return errors.NewValidationError("invalid job ID",
			map[string]interface{}{"value": jobIDStr})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would get logs for job %d in %s/%s\n", jobID, owner, repo)
		return nil
	}

	logs, resp, err := client.GetRepoActionJobLogs(owner, repo, jobID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "action job", strconv.FormatInt(jobID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"log": string(logs)}, func() error {
		os.Stdout.Write(logs)
		return nil
	})
}
