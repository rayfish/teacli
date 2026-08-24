// Package cmd contains CI/CD actions; this file blocks until a run finishes.
package cmd

import (
	"fmt"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/output"

	"github.com/spf13/cobra"
)

// inFlightStatuses are the run statuses that mean the run has not settled yet.
// Gitea reports both its internal names (waiting, running, blocked) and the
// GitHub-compatible ones (queued, in_progress) depending on the endpoint, so
// both sets are listed and anything unknown counts as terminal.
var inFlightStatuses = map[string]bool{
	"pending":     true,
	"queued":      true,
	"waiting":     true,
	"running":     true,
	"in_progress": true,
	"blocked":     true,
}

// runInFlight reports whether a run with this status is still going.
func runInFlight(status string) bool {
	return inFlightStatuses[status]
}

// runOutcome is how the run ended. Gitea fills conclusion on completed runs but
// leaves it empty on some, putting the result in status instead.
func runOutcome(run *gitea.ActionWorkflowRun) string {
	if run.Conclusion != "" {
		return run.Conclusion
	}
	return run.Status
}

// watchOutcomeError turns a non-success outcome into an exit-1 error, so
// `teacli action watch 42 && deploy` does the right thing.
func watchOutcomeError(run *gitea.ActionWorkflowRun, input string) error {
	outcome := runOutcome(run)
	if outcome == "success" {
		return nil
	}
	return errors.NewGeneralError(fmt.Sprintf("workflow run %s finished: %s", input, outcome))
}

// validateWatchTiming rejects timings that would never poll or never stop. A
// zero timeout is allowed and means "no limit".
func validateWatchTiming(interval, timeout time.Duration) error {
	if interval <= 0 {
		return errors.NewValidationError("--interval must be greater than zero",
			map[string]interface{}{"value": interval.String()})
	}
	if timeout < 0 {
		return errors.NewValidationError("--timeout cannot be negative",
			map[string]interface{}{"value": timeout.String()})
	}
	return nil
}

// runWatcher polls a single workflow run until it reaches a terminal state. The
// clock and the fetch are injected so tests can drive long waits instantly.
type runWatcher struct {
	fetch    func() (*gitea.ActionWorkflowRun, error)
	interval time.Duration
	timeout  time.Duration
	now      func() time.Time
	sleep    func(time.Duration)
	onChange func(status string)
}

// wait returns the run once it stops running, or an error if it never does
// before the timeout.
func (w runWatcher) wait() (*gitea.ActionWorkflowRun, error) {
	deadline := w.now().Add(w.timeout)
	previous := ""

	for {
		run, err := w.fetch()
		if err != nil {
			return nil, err
		}

		if run.Status != previous {
			previous = run.Status
			if w.onChange != nil {
				w.onChange(run.Status)
			}
		}

		if !runInFlight(run.Status) {
			return run, nil
		}

		wait := w.interval
		if w.timeout > 0 {
			left := deadline.Sub(w.now())
			if left <= 0 {
				return nil, errors.NewGeneralError(fmt.Sprintf(
					"timed out after %s waiting for the workflow run to finish", w.timeout))
			}
			// Trim the last wait so the run is polled once at the deadline
			// rather than skipped past it.
			if left < wait {
				wait = left
			}
		}

		w.sleep(wait)
	}
}

var actionWatchCmd = &cobra.Command{
	Use:     "watch [<owner>/<repo>] <run-id>",
	Aliases: []string{"wait"},
	Short:   "Wait for a workflow run to finish",
	Long: `Poll a workflow run until it finishes.

Exits 0 when the run concludes successfully and 1 for any other conclusion, so
it composes: teacli action watch 42 && teacli release create v1.0.0`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runActionWatch,
}

func init() {
	actionCmd.AddCommand(actionWatchCmd)

	actionWatchCmd.Flags().Duration("interval", 5*time.Second, "Time between polls")
	actionWatchCmd.Flags().Duration("timeout", 30*time.Minute, "Give up after this long (0 waits forever)")
}

func runActionWatch(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	interval, _ := cmd.Flags().GetDuration("interval")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	if err := validateWatchTiming(interval, timeout); err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	if dryRunf(cmd, "Would watch workflow run %s in %s/%s (every %s, up to %s)",
		args[1], owner, repo, interval, timeout) {
		return nil
	}

	runID, err := resolveActionRunID(client, owner, repo, args[1])
	if err != nil {
		return err
	}

	watcher := runWatcher{
		fetch: func() (*gitea.ActionWorkflowRun, error) {
			run, resp, err := client.GetRepoActionRun(owner, repo, runID)
			if err != nil {
				return nil, errors.FromGiteaNotFound(resp, err, "workflow run", args[1])
			}
			return run, nil
		},
		interval: interval,
		timeout:  timeout,
		now:      time.Now,
		sleep:    time.Sleep,
	}

	// Progress goes to stderr, and only on a status change, so stdout stays a
	// single parseable document and a long wait does not fill a log.
	if printer.Format() != output.FormatJSON {
		watcher.onChange = func(status string) {
			fmt.Fprintf(cmd.ErrOrStderr(), "run %s: %s\n", args[1], status)
		}
	}

	finished, err := watcher.wait()
	if err != nil {
		return err
	}

	if err := printer.Emit(finished, func() error {
		return printActionRunText(printer, finished)
	}); err != nil {
		return err
	}

	return watchOutcomeError(finished, args[1])
}
