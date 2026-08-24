// Package cmd contains time tracking and stopwatch commands.
package cmd

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var timeCmd = &cobra.Command{
	Use:     "time",
	Aliases: []string{"times"},
	Short:   "Track time spent on issues",
}

var timeListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] [<issue-number>]",
	Short: "List tracked times for a repository or one issue",
	Args:  cobra.RangeArgs(0, 2),
	RunE:  runTimeList,
}

var timeMineCmd = &cobra.Command{
	Use:   "mine",
	Short: "List your own tracked times across every repository",
	Args:  cobra.NoArgs,
	RunE:  runTimeMine,
}

var timeAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> <duration>",
	Short: "Add tracked time to an issue",
	Long: `Add tracked time to an issue.

The duration accepts a plain number of seconds, or a Go duration such as 90m,
1h30m or 2h.`,
	Args: cobra.RangeArgs(2, 3),
	RunE: runTimeAdd,
}

var timeDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <issue-number> <time-id>",
	Aliases: []string{"rm"},
	Short:   "Delete one tracked time entry",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runTimeDelete,
}

var timeResetCmd = &cobra.Command{
	Use:   "reset [<owner>/<repo>] <issue-number>",
	Short: "Delete every tracked time entry on an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runTimeReset,
}

var stopwatchCmd = &cobra.Command{
	Use:     "stopwatch",
	Aliases: []string{"stopwatches", "sw"},
	Short:   "Start, stop and inspect issue stopwatches",
}

var stopwatchListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your running stopwatches",
	Args:  cobra.NoArgs,
	RunE:  runStopwatchList,
}

var stopwatchStartCmd = &cobra.Command{
	Use:   "start [<owner>/<repo>] <issue-number>",
	Short: "Start a stopwatch on an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStopwatchStart,
}

var stopwatchStopCmd = &cobra.Command{
	Use:   "stop [<owner>/<repo>] <issue-number>",
	Short: "Stop a stopwatch and record the elapsed time",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStopwatchStop,
}

var stopwatchCancelCmd = &cobra.Command{
	Use:     "cancel [<owner>/<repo>] <issue-number>",
	Aliases: []string{"delete"},
	Short:   "Cancel a stopwatch without recording any time",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runStopwatchCancel,
}

func init() {
	RootCmd.AddCommand(timeCmd, stopwatchCmd)
	timeCmd.AddCommand(timeListCmd, timeMineCmd, timeAddCmd, timeDeleteCmd, timeResetCmd)
	stopwatchCmd.AddCommand(stopwatchListCmd, stopwatchStartCmd, stopwatchStopCmd, stopwatchCancelCmd)

	addPageFlags(timeListCmd, timeMineCmd, stopwatchListCmd)
	timeListCmd.Flags().StringP("user", "u", "", "Only entries by this user (repository-wide listing only)")
	for _, c := range []*cobra.Command{timeListCmd, timeMineCmd} {
		c.Flags().String("since", "", "Only entries at or after this RFC3339 time")
		c.Flags().String("before", "", "Only entries before this RFC3339 time")
	}
	timeAddCmd.Flags().StringP("user", "u", "", "Record the time against this user (requires permission)")
}

// parseDuration accepts either a bare number of seconds or a Go duration
// string, so "3600", "1h" and "90m" all work.
func parseDuration(s string) (int64, error) {
	if seconds, err := strconv.ParseInt(s, 10, 64); err == nil {
		if seconds <= 0 {
			return 0, errors.NewValidationError("duration must be positive", nil)
		}
		return seconds, nil
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, errors.NewValidationError(fmt.Sprintf("invalid duration: %s", s),
			map[string]interface{}{"expected": "seconds, or a duration such as 90m or 1h30m"})
	}
	if d <= 0 {
		return 0, errors.NewValidationError("duration must be positive", nil)
	}
	return int64(d.Seconds()), nil
}

// formatSeconds renders a tracked duration as hours, minutes and seconds,
// dropping the units that are zero. Go's Duration.String would give "1h0m0s"
// where "1h" reads better in a table.
func formatSeconds(seconds int64) string {
	if seconds <= 0 {
		return "0s"
	}

	hours := seconds / 3600
	minutes := (seconds % 3600) / 60
	secs := seconds % 60

	var b strings.Builder
	if hours > 0 {
		fmt.Fprintf(&b, "%dh", hours)
	}
	if minutes > 0 {
		fmt.Fprintf(&b, "%dm", minutes)
	}
	if secs > 0 || b.Len() == 0 {
		fmt.Fprintf(&b, "%ds", secs)
	}
	return b.String()
}

func emitTrackedTimes(cmd *cobra.Command, times []*gitea.TrackedTime) error {
	printer := getPrinter(cmd)
	return printer.Emit(times, func() error {
		if len(times) == 0 {
			printer.Println("No tracked time found.")
			return nil
		}
		var total int64
		rows := make([][]string, 0, len(times))
		for _, t := range times {
			total += t.Time
			issue := ""
			if t.Issue != nil {
				issue = fmt.Sprintf("#%d %s", t.Issue.Index, t.Issue.Title)
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", t.ID), t.UserName, formatSeconds(t.Time),
				renderValue(t.Created), issue,
			})
		}
		if err := printer.PrintTable([]string{"ID", "User", "Time", "Created", "Issue"}, rows); err != nil {
			return err
		}
		printer.Printf("\nTotal: %s\n", formatSeconds(total))
		return nil
	})
}

func trackedTimeFilters(cmd *cobra.Command) (time.Time, time.Time, error) {
	since, err := timeFlag(cmd, "since")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	before, err := timeFlag(cmd, "before")
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return since, before, nil
}

func runTimeList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	since, before, err := trackedTimeFilters(cmd)
	if err != nil {
		return err
	}
	user, _ := cmd.Flags().GetString("user")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var index int64 = -1
	if len(args) == 2 {
		if index, err = int64Arg(args[1], "issue number"); err != nil {
			return err
		}
	}

	times, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.TrackedTime, *gitea.Response, error) {
		opts := gitea.ListTrackedTimesOptions{ListOptions: lo, Since: since, Before: before}
		if index >= 0 {
			// The per-issue endpoint ignores the user filter, so do not send it.
			return client.ListIssueTrackedTimes(owner, repo, index, opts)
		}
		opts.User = user
		return client.ListRepoTrackedTimes(owner, repo, opts)
	})
	if err != nil {
		return err
	}

	return emitTrackedTimes(cmd, times)
}

func runTimeMine(cmd *cobra.Command, args []string) error {
	since, before, err := trackedTimeFilters(cmd)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	times, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.TrackedTime, *gitea.Response, error) {
		return client.ListMyTrackedTimes(gitea.ListTrackedTimesOptions{
			ListOptions: lo, Since: since, Before: before,
		})
	})
	if err != nil {
		return err
	}

	return emitTrackedTimes(cmd, times)
}

func runTimeAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}
	seconds, err := parseDuration(args[2])
	if err != nil {
		return err
	}
	user, _ := cmd.Flags().GetString("user")

	if dryRunf(cmd, "Would add %s to %s/%s#%d", formatSeconds(seconds), owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tracked, resp, err := client.AddTime(owner, repo, index, gitea.AddTimeOption{Time: seconds, User: user})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}

	return emitMessage(cmd, tracked, "Added %s to %s/%s#%d", formatSeconds(tracked.Time), owner, repo, index)
}

func runTimeDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}
	timeID, err := int64Arg(args[2], "time id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete time entry %d on %s/%s#%d", timeID, owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.DeleteTime(owner, repo, index, timeID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tracked time", args[2])
	}

	return emitMessage(cmd, okMessage("tracked time deleted"), "Deleted time entry %d", timeID)
}

func runTimeReset(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete every time entry on %s/%s#%d", owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.ResetIssueTime(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}

	return emitMessage(cmd, okMessage("tracked time reset"), "Reset tracked time on %s/%s#%d", owner, repo, index)
}

func runStopwatchList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	watches, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.StopWatch, *gitea.Response, error) {
		return client.ListMyStopwatches(gitea.ListStopwatchesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(watches, func() error {
		if len(watches) == 0 {
			printer.Println("No stopwatches running.")
			return nil
		}
		rows := make([][]string, 0, len(watches))
		for _, w := range watches {
			rows = append(rows, []string{
				fmt.Sprintf("%s/%s#%d", w.RepoOwnerName, w.RepoName, w.IssueIndex),
				w.IssueTitle, w.Duration, renderValue(w.Created),
			})
		}
		return printer.PrintTable([]string{"Issue", "Title", "Elapsed", "Started"}, rows)
	})
}

func runStopwatchStart(cmd *cobra.Command, args []string) error {
	return stopwatchAction(cmd, args, "start")
}

func runStopwatchStop(cmd *cobra.Command, args []string) error {
	return stopwatchAction(cmd, args, "stop")
}

func runStopwatchCancel(cmd *cobra.Command, args []string) error {
	return stopwatchAction(cmd, args, "cancel")
}

func stopwatchAction(cmd *cobra.Command, args []string, action string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would %s the stopwatch on %s/%s#%d", action, owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var resp *gitea.Response
	switch action {
	case "start":
		resp, err = client.StartIssueStopWatch(owner, repo, index)
	case "stop":
		resp, err = client.StopIssueStopWatch(owner, repo, index)
	default:
		resp, err = client.DeleteIssueStopwatch(owner, repo, index)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}

	past := map[string]string{"start": "Started", "stop": "Stopped", "cancel": "Cancelled"}[action]
	return emitMessage(cmd, okMessage("stopwatch "+action),
		"%s the stopwatch on %s/%s#%d", past, owner, repo, index)
}
