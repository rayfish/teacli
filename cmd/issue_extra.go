// Package cmd contains the issue commands beyond create/close: edit, delete,
// pin, lock, dependencies, timeline and comment editing.
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var issueUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <issue-number>",
	Aliases: []string{"edit"},
	Short:   "Edit an issue's title, body, milestone, assignees or due date",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runIssueUpdate,
}

var issueDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <issue-number>",
	Aliases: []string{"rm"},
	Short:   "Delete an issue permanently (admin only)",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runIssueDelete,
}

var issuePinCmd = &cobra.Command{
	Use:   "pin [<owner>/<repo>] <issue-number>",
	Short: "Pin an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssuePin,
}

var issueUnpinCmd = &cobra.Command{
	Use:   "unpin [<owner>/<repo>] <issue-number>",
	Short: "Unpin an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueUnpin,
}

var issuePinnedCmd = &cobra.Command{
	Use:   "pinned [<owner>/<repo>]",
	Short: "List the pinned issues of a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIssuePinned,
}

var issueLockCmd = &cobra.Command{
	Use:   "lock [<owner>/<repo>] <issue-number>",
	Short: "Lock an issue's conversation",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueLock,
}

var issueUnlockCmd = &cobra.Command{
	Use:   "unlock [<owner>/<repo>] <issue-number>",
	Short: "Unlock an issue's conversation",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueUnlock,
}

var issueTimelineCmd = &cobra.Command{
	Use:   "timeline [<owner>/<repo>] <issue-number>",
	Short: "Show an issue's full event timeline",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueTimeline,
}

var issueDependencyCmd = &cobra.Command{
	Use:     "dependency",
	Aliases: []string{"dependencies", "deps"},
	Short:   "Manage the issues an issue depends on",
}

var issueDependencyListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List the issues this issue depends on",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueDependencyList,
}

var issueDependencyAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> <depends-on-number>",
	Short: "Make an issue depend on another",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runIssueDependencyAdd,
}

var issueDependencyRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <issue-number> <depends-on-number>",
	Aliases: []string{"rm"},
	Short:   "Remove a dependency",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runIssueDependencyRemove,
}

var issueBlockCmd = &cobra.Command{
	Use:     "blocking",
	Aliases: []string{"blocks"},
	Short:   "Manage the issues an issue blocks",
}

var issueBlockListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List the issues this issue blocks",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueBlockList,
}

var issueBlockAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> <blocked-number>",
	Short: "Mark an issue as blocking another",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runIssueBlockAdd,
}

var issueBlockRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <issue-number> <blocked-number>",
	Aliases: []string{"rm"},
	Short:   "Stop an issue blocking another",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runIssueBlockRemove,
}

var issueCommentUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <comment-id>",
	Aliases: []string{"edit"},
	Short:   "Edit a comment",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runIssueCommentUpdate,
}

var issueCommentGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <comment-id>",
	Short: "Get one comment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueCommentGet,
}

func init() {
	issueCmd.AddCommand(issueUpdateCmd, issueDeleteCmd, issuePinCmd, issueUnpinCmd,
		issuePinnedCmd, issueLockCmd, issueUnlockCmd, issueTimelineCmd,
		issueDependencyCmd, issueBlockCmd)
	issueDependencyCmd.AddCommand(issueDependencyListCmd, issueDependencyAddCmd, issueDependencyRemoveCmd)
	issueBlockCmd.AddCommand(issueBlockListCmd, issueBlockAddCmd, issueBlockRemoveCmd)
	issueCommentCmd.AddCommand(issueCommentUpdateCmd, issueCommentGetCmd)

	addPageFlags(issueTimelineCmd, issueDependencyListCmd, issueBlockListCmd)

	issueUpdateCmd.Flags().StringP("title", "t", "", "New title")
	issueUpdateCmd.Flags().StringP("body", "b", "", "New body")
	issueUpdateCmd.Flags().String("body-file", "", "Read the new body from this file, or - for stdin")
	issueUpdateCmd.Flags().String("ref", "", "New git ref")
	issueUpdateCmd.Flags().StringSliceP("assignee", "a", nil,
		"Replace the assignee set with these users (repeatable)")
	issueUpdateCmd.Flags().Int64P("milestone", "m", 0, "Milestone ID to set, or 0 to clear it")
	issueUpdateCmd.Flags().StringP("state", "s", "", "New state: open or closed")
	issueUpdateCmd.Flags().StringP("due", "D", "", "Due date, as YYYY-MM-DD or RFC3339")
	issueUpdateCmd.Flags().Bool("no-due", false, "Clear the due date")

	issueLockCmd.Flags().StringP("reason", "r", "", "Lock reason, e.g. resolved or spam")

	for _, c := range []*cobra.Command{issueCommentUpdateCmd} {
		c.Flags().StringP("body", "b", "", "New comment body")
		c.Flags().String("body-file", "", "Read the new body from this file, or - for stdin")
	}
}

func runIssueUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}

	body, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}

	opts := gitea.EditIssueOption{}
	if cmd.Flags().Changed("title") {
		opts.Title, _ = cmd.Flags().GetString("title")
	}
	if body != "" {
		opts.Body = &body
	}
	opts.Ref = stringFlagPtr(cmd, "ref")
	if cmd.Flags().Changed("assignee") {
		opts.Assignees, _ = cmd.Flags().GetStringSlice("assignee")
	}
	if cmd.Flags().Changed("milestone") {
		opts.Milestone = int64FlagPtr(cmd, "milestone")
	}
	if cmd.Flags().Changed("state") {
		raw, _ := cmd.Flags().GetString("state")
		state, err := parseState(raw)
		if err != nil {
			return err
		}
		if state == gitea.StateAll {
			return errors.NewValidationError("state must be open or closed", nil)
		}
		opts.State = &state
	}
	if cmd.Flags().Changed("due") {
		raw, _ := cmd.Flags().GetString("due")
		due, err := parseDueDate(raw)
		if err != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid due date: %s", raw),
				map[string]interface{}{"expected": "YYYY-MM-DD or an RFC3339 timestamp"})
		}
		opts.Deadline = &due
	}
	if noDue, _ := cmd.Flags().GetBool("no-due"); noDue {
		opts.RemoveDeadline = &noDue
	}

	if opts.Title == "" && opts.Body == nil && opts.Ref == nil && opts.Assignees == nil &&
		opts.Milestone == nil && opts.State == nil && opts.Deadline == nil && opts.RemoveDeadline == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --title"})
	}

	if dryRunf(cmd, "Would edit %s/%s#%d", owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issue, resp, err := client.EditIssue(owner, repo, index, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(issue, func() error {
		return printIssueText(printer, issue)
	})
}

func runIssueDelete(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete %s/%s#%d", owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteIssue(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, okMessage("issue deleted"), "Deleted %s/%s#%d", owner, repo, index)
}

// repoAndIndex resolves the "[<owner>/<repo>] <number>" argument pair that most
// issue and pull-request commands start with, inferring the repository from the
// current directory when it was left out. wantRest counts the arguments after
// the repository, the number included. It returns args with the repository
// restored at index 0.
func repoAndIndex(cmd *cobra.Command, args []string, wantRest int) (string, string, int64, []string, error) {
	owner, repo, args, err := repoTarget(cmd, args, wantRest)
	if err != nil {
		return "", "", 0, nil, err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return "", "", 0, nil, err
	}
	return owner, repo, index, args, nil
}

func runIssuePin(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would pin %s/%s#%d", owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.PinIssue(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, okMessage("issue pinned"), "Pinned %s/%s#%d", owner, repo, index)
}

func runIssueUnpin(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would unpin %s/%s#%d", owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.UnpinIssue(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, okMessage("issue unpinned"), "Unpinned %s/%s#%d", owner, repo, index)
}

func runIssuePinned(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issues, resp, err := client.ListRepoPinnedIssues(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitIssueTable(cmd, issues, "No pinned issues.")
}

// emitIssueTable renders an issue list, shared by every command that returns
// issues rather than a single one.
func emitIssueTable(cmd *cobra.Command, issues []*gitea.Issue, empty string) error {
	printer := getPrinter(cmd)
	return printer.Emit(issues, func() error {
		if len(issues) == 0 {
			printer.Println(empty)
			return nil
		}
		rows := make([][]string, 0, len(issues))
		for _, i := range issues {
			rows = append(rows, []string{
				fmt.Sprintf("%d", i.Index), cell(i.Title, 60), string(i.State),
				renderValue(labelNames(i.Labels)), renderValue(userNames(i.Assignees)),
			})
		}
		return printer.PrintTable([]string{"Number", "Title", "State", "Labels", "Assignees"}, rows)
	})
}

func runIssueLock(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	reason, _ := cmd.Flags().GetString("reason")

	if dryRunf(cmd, "Would lock %s/%s#%d", owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.LockIssue(owner, repo, index, gitea.LockIssueOption{LockReason: reason})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, okMessage("issue locked"), "Locked %s/%s#%d", owner, repo, index)
}

func runIssueUnlock(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would unlock %s/%s#%d", owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.UnlockIssue(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, okMessage("issue unlocked"), "Unlocked %s/%s#%d", owner, repo, index)
}

func runIssueTimeline(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	events, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.TimelineComment, *gitea.Response, error) {
		return client.ListIssueTimeline(owner, repo, index, gitea.ListIssueCommentOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(events, func() error {
		if len(events) == 0 {
			printer.Println("No timeline events.")
			return nil
		}
		rows := make([][]string, 0, len(events))
		for _, e := range events {
			// Most event types carry no body; show whatever detail they do have
			// so the row is not blank.
			summary := cell(e.Body, 60)
			if summary == "" {
				switch {
				case e.NewTitle != "":
					summary = fmt.Sprintf("%s -> %s", cell(e.OldTitle, 25), cell(e.NewTitle, 25))
				case len(e.Label) > 0:
					summary = strings.Join(labelNames(e.Label), ", ")
				case e.NewMilestone != nil:
					summary = e.NewMilestone.Title
				}
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", e.ID), e.Type, userName(e.Poster), renderValue(e.Created), summary,
			})
		}
		return printer.PrintTable([]string{"ID", "Type", "User", "When", "Detail"}, rows)
	})
}

func runIssueDependencyList(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issues, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Issue, *gitea.Response, error) {
		return client.ListIssueDependencies(owner, repo, index,
			gitea.ListIssueDependenciesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitIssueTable(cmd, issues, "No dependencies.")
}

func runIssueDependencyAdd(cmd *cobra.Command, args []string) error {
	return changeIssueLink(cmd, args, "dependency", true)
}

func runIssueDependencyRemove(cmd *cobra.Command, args []string) error {
	return changeIssueLink(cmd, args, "dependency", false)
}

func runIssueBlockList(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issues, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Issue, *gitea.Response, error) {
		return client.ListIssueBlocks(owner, repo, index, gitea.ListIssueBlocksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitIssueTable(cmd, issues, "This issue blocks nothing.")
}

func runIssueBlockAdd(cmd *cobra.Command, args []string) error {
	return changeIssueLink(cmd, args, "block", true)
}

func runIssueBlockRemove(cmd *cobra.Command, args []string) error {
	return changeIssueLink(cmd, args, "block", false)
}

// changeIssueLink adds or removes a dependency or blocking relation between two
// issues in the same repository.
func changeIssueLink(cmd *cobra.Command, args []string, kind string, add bool) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 2)
	if err != nil {
		return err
	}
	other, err := int64Arg(args[2], "issue number")
	if err != nil {
		return err
	}

	verb := "remove"
	if add {
		verb = "add"
	}
	if dryRunf(cmd, "Would %s %s #%d on %s/%s#%d", verb, kind, other, owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	meta := gitea.IssueMeta{Index: other}
	var resp *gitea.Response
	switch {
	case kind == "dependency" && add:
		_, resp, err = client.CreateIssueDependency(owner, repo, index, meta)
	case kind == "dependency":
		_, resp, err = client.RemoveIssueDependency(owner, repo, index, meta)
	case add:
		_, resp, err = client.CreateIssueBlocking(owner, repo, index, meta)
	default:
		_, resp, err = client.RemoveIssueBlocking(owner, repo, index, meta)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[2])
	}

	past := "Removed"
	if add {
		past = "Added"
	}
	return emitMessage(cmd, okMessage(kind+" "+verb+"d"),
		"%s %s #%d on %s/%s#%d", past, kind, other, owner, repo, index)
}

func runIssueCommentGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	comment, resp, err := client.GetIssueComment(owner, repo, commentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comment", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(comment, func() error {
		var d detail
		d.always("ID", comment.ID)
		d.add("Author", userName(comment.Poster))
		d.add("Created", comment.Created)
		d.add("Updated", comment.Updated)
		d.add("URL", comment.HTMLURL)
		d.add("Body", comment.Body)
		return d.print(printer)
	})
}

func runIssueCommentUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}

	body, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}
	if body == "" {
		return errors.NewValidationError("comment body required",
			map[string]interface{}{"hint": "pass --body <text> or --body-file <path>"})
	}

	if dryRunf(cmd, "Would edit comment %d", commentID) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	comment, resp, err := client.EditIssueComment(owner, repo, commentID,
		gitea.EditIssueCommentOption{Body: body})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comment", args[1])
	}

	return emitMessage(cmd, comment, "Edited comment %d", comment.ID)
}
