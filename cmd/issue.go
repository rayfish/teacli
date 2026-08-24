// Package cmd contains issue commands
package cmd

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var issueCmd = &cobra.Command{
	Use:     "issue",
	Aliases: []string{"issues"},
	Short:   "Manage issues",
	Long:    `List, create, close, reopen, and manage issues in repositories.`,
}

var issueListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List issues",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIssueList,
}

var issueGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <issue-number>",
	Short: "Get a specific issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueGet,
}

var issueCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new issue",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runIssueCreate,
}

var issueCloseCmd = &cobra.Command{
	Use:   "close [<owner>/<repo>] <issue-number>",
	Short: "Close an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueClose,
}

var issueReopenCmd = &cobra.Command{
	Use:   "reopen [<owner>/<repo>] <issue-number>",
	Short: "Reopen an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueReopen,
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage issue comments",
	Long:  `List, create, and delete comments on issues.`,
}

var issueCommentListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List comments on an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueCommentList,
}

var issueCommentCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <issue-number> --body <text>",
	Short: "Create a comment on an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueCommentCreate,
}

var issueCommentDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <comment-id>",
	Short: "Delete a comment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueCommentDelete,
}

var issueLabelCmd = &cobra.Command{
	Use:   "label",
	Short: "Manage labels on an issue",
	Long:  `List, add, remove, and clear the labels applied to an issue.`,
}

var issueLabelListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List labels on an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueLabelList,
}

var issueLabelAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> <label>...",
	Short: "Add labels to an issue (by name or ID)",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runIssueLabelAdd,
}

var issueLabelRemoveCmd = &cobra.Command{
	Use:   "remove [<owner>/<repo>] <issue-number> <label>...",
	Short: "Remove labels from an issue (by name or ID)",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runIssueLabelRemove,
}

var issueLabelClearCmd = &cobra.Command{
	Use:   "clear [<owner>/<repo>] <issue-number>",
	Short: "Remove every label from an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runIssueLabelClear,
}

var issueAssignCmd = &cobra.Command{
	Use:   "assign [<owner>/<repo>] <issue-number> <username>...",
	Short: "Add assignees to an issue",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runIssueAssign,
}

var issueUnassignCmd = &cobra.Command{
	Use:   "unassign [<owner>/<repo>] <issue-number> <username>...",
	Short: "Remove assignees from an issue",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runIssueUnassign,
}

func init() {
	RootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd, issueGetCmd, issueCreateCmd, issueCloseCmd, issueReopenCmd,
		issueCommentCmd, issueLabelCmd, issueAssignCmd, issueUnassignCmd)
	issueCommentCmd.AddCommand(issueCommentListCmd, issueCommentCreateCmd, issueCommentDeleteCmd)
	issueLabelCmd.AddCommand(issueLabelListCmd, issueLabelAddCmd, issueLabelRemoveCmd, issueLabelClearCmd)

	issueListCmd.Flags().StringP("state", "s", "open", "Filter by state: open, closed, all")
	issueListCmd.Flags().StringSlice("label", nil, "Filter by label name (repeatable)")
	issueListCmd.Flags().StringSlice("milestone", nil, "Filter by milestone title (repeatable)")
	issueListCmd.Flags().String("assignee", "", "Filter by assignee username")
	issueListCmd.Flags().String("creator", "", "Filter by creator username")
	issueListCmd.Flags().StringP("keyword", "q", "", "Search issue titles and bodies")
	issueListCmd.Flags().Int("page", 0, "Page number")
	issueListCmd.Flags().Int("per-page", 0, "Results per page")

	issueCreateCmd.Flags().StringP("title", "t", "", "Issue title (required)")
	issueCreateCmd.MarkFlagRequired("title")
	issueCreateCmd.Flags().StringP("body", "b", "", "Issue description")
	issueCreateCmd.Flags().StringSlice("label", nil, "Label to apply, by name or ID (repeatable)")
	issueCreateCmd.Flags().StringSlice("assignee", nil, "Assignee username (repeatable)")

	issueCommentCreateCmd.Flags().StringP("body", "b", "", "Comment body (required)")
	issueCommentCreateCmd.MarkFlagRequired("body")

	issueCommentListCmd.Flags().Int("page", 0, "Page number")
	issueCommentListCmd.Flags().Int("per-page", 0, "Results per page")
}

func runIssueList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	state, _ := cmd.Flags().GetString("state")
	stateType, err := parseStateFilter(state)
	if err != nil {
		return err
	}

	labels, _ := cmd.Flags().GetStringSlice("label")
	milestones, _ := cmd.Flags().GetStringSlice("milestone")
	assignee, _ := cmd.Flags().GetString("assignee")
	creator, _ := cmd.Flags().GetString("creator")
	keyword, _ := cmd.Flags().GetString("keyword")

	issues, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Issue, *gitea.Response, error) {
		return client.ListRepoIssues(owner, repo, gitea.ListIssueOption{
			ListOptions: lo,
			State:       stateType,
			Type:        gitea.IssueTypeIssue,
			Labels:      labels,
			Milestones:  milestones,
			AssignedBy:  assignee,
			CreatedBy:   creator,
			KeyWord:     keyword,
		})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	headers := []string{"ID", "Title", "State", "Labels", "Assignees"}
	var rows [][]string

	for _, issue := range issues {
		var labelNames []string
		for _, l := range issue.Labels {
			labelNames = append(labelNames, l.Name)
		}
		var assignees []string
		for _, a := range issue.Assignees {
			assignees = append(assignees, a.UserName)
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", issue.Index),
			utils.Truncate(issue.Title, 40),
			string(issue.State),
			utils.Truncate(strings.Join(labelNames, ","), 24),
			strings.Join(assignees, ","),
		})
	}

	return printer.Emit(issues, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runIssueGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issue, resp, err := client.GetIssue(owner, repo, issueNum)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(issue, func() error {
		return printIssueText(printer, issue)
	})
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	title, _ := cmd.Flags().GetString("title")
	body, _ := cmd.Flags().GetString("body")
	labelArgs, _ := cmd.Flags().GetStringSlice("label")
	assignees, _ := cmd.Flags().GetStringSlice("assignee")

	if title == "" {
		return errors.NewValidationError("--title is required",
			map[string]interface{}{"missing_flags": []string{"--title"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var labelIDs []int64
	if len(labelArgs) > 0 {
		labelIDs, err = resolveLabelIDs(cmd, client, owner, repo, labelArgs)
		if err != nil {
			return err
		}
	}

	opts := gitea.CreateIssueOption{
		Title:     title,
		Body:      body,
		Labels:    labelIDs,
		Assignees: assignees,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create issue\n")
		printer.Printf("  Title: %s\n", title)
		if body != "" {
			printer.Printf("  Body: %s\n", utils.Truncate(body, 80))
		}
		if len(labelArgs) > 0 {
			printer.Printf("  Labels: %s\n", strings.Join(labelArgs, ", "))
		}
		if len(assignees) > 0 {
			printer.Printf("  Assignees: %s\n", strings.Join(assignees, ", "))
		}
		return nil
	}

	issue, resp, err := client.CreateIssue(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(issue, func() error {
		printer.Printf("Created issue #%d: %s\n", issue.Index, issue.HTMLURL)
		return nil
	})
}

func runIssueClose(cmd *cobra.Command, args []string) error {
	return updateIssueState(cmd, args, "closed")
}

func runIssueReopen(cmd *cobra.Command, args []string) error {
	return updateIssueState(cmd, args, "open")
}

func updateIssueState(cmd *cobra.Command, args []string, state string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	st := gitea.StateType(state)
	editOpts := gitea.EditIssueOption{
		State: &st,
	}

	issue, resp, err := client.EditIssue(owner, repo, issueNum, editOpts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(issue, func() error {
		printer.Printf("%s issue #%d\n", strings.ToUpper(state), issue.Index)
		return nil
	})
}

func runIssueCommentList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	comments, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Comment, *gitea.Response, error) {
		return client.ListIssueComments(owner, repo, issueNum, gitea.ListIssueCommentOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	headers := []string{"ID", "Author", "Created", "Body"}
	var rows [][]string

	for _, c := range comments {
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.ID),
			c.Poster.UserName,
			c.Created.Format("2006-01-02 15:04"),
			utils.Truncate(c.Body, 30),
		})
	}

	return printer.Emit(comments, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runIssueCommentCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])
	body, _ := cmd.Flags().GetString("body")

	if body == "" {
		return errors.NewValidationError("--body is required",
			map[string]interface{}{"missing_flags": []string{"--body"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateIssueCommentOption{
		Body: body,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create comment on issue #%d\n", issueNum)
		printer.Printf("  Body: %s\n", utils.Truncate(body, 80))
		return nil
	}

	comment, resp, err := client.CreateIssueComment(owner, repo, issueNum, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(comment, func() error {
		printer.Printf("Created comment #%d on issue #%d\n", comment.ID, issueNum)
		return nil
	})
}

// parseStateFilter maps a --state value onto the SDK's state type.
func parseStateFilter(state string) (gitea.StateType, error) {
	switch strings.ToLower(state) {
	case "", "open":
		return gitea.StateOpen, nil
	case "closed":
		return gitea.StateClosed, nil
	case "all":
		return gitea.StateAll, nil
	default:
		return "", errors.NewValidationError("invalid --state",
			map[string]interface{}{"value": state, "valid": []string{"open", "closed", "all"}})
	}
}

// resolveLabelIDs turns label names or numeric IDs into label IDs. Names are
// matched case-insensitively against the repo's labels.
func resolveLabelIDs(cmd *cobra.Command, client *gitea.Client, owner, repo string, refs []string) ([]int64, error) {
	byName := map[string]int64{}
	needsLookup := false
	for _, ref := range refs {
		if _, err := strconv.ParseInt(ref, 10, 64); err != nil {
			needsLookup = true
			break
		}
	}

	if needsLookup {
		labels, resp, err := client.ListRepoLabels(owner, repo, gitea.ListLabelsOptions{
			ListOptions: gitea.ListOptions{Page: 1, PageSize: defaultPageSize},
		})
		if err != nil {
			return nil, errors.FromGitea(resp, err)
		}
		for _, l := range labels {
			byName[strings.ToLower(l.Name)] = l.ID
		}
	}

	ids := make([]int64, 0, len(refs))
	for _, ref := range refs {
		if id, err := strconv.ParseInt(ref, 10, 64); err == nil {
			ids = append(ids, id)
			continue
		}
		id, ok := byName[strings.ToLower(ref)]
		if !ok {
			return nil, errors.NewNotFoundError("label", ref)
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func runIssueLabelList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	labels, resp, err := client.GetIssueLabels(owner, repo, issueNum, gitea.ListLabelsOptions{})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	printer := getPrinter(cmd)
	headers := []string{"ID", "Name", "Color"}
	var rows [][]string
	for _, l := range labels {
		rows = append(rows, []string{fmt.Sprintf("%d", l.ID), l.Name, l.Color})
	}

	return printer.Emit(labels, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runIssueLabelAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])
	refs := args[2:]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	ids, err := resolveLabelIDs(cmd, client, owner, repo, refs)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would add labels %s to issue #%d\n", strings.Join(refs, ", "), issueNum)
		return nil
	}

	labels, resp, err := client.AddIssueLabels(owner, repo, issueNum, gitea.IssueLabelsOption{Labels: ids})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	return printer.Emit(labels, func() error {
		printer.Printf("Added %d label(s) to issue #%d\n", len(ids), issueNum)
		return nil
	})
}

func runIssueLabelRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])
	refs := args[2:]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	ids, err := resolveLabelIDs(cmd, client, owner, repo, refs)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would remove labels %s from issue #%d\n", strings.Join(refs, ", "), issueNum)
		return nil
	}

	for _, id := range ids {
		resp, err := client.DeleteIssueLabel(owner, repo, issueNum, id)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
		}
	}

	return printer.Emit(map[string]any{"status": "removed", "issue": issueNum, "labels": ids}, func() error {
		printer.Printf("Removed %d label(s) from issue #%d\n", len(ids), issueNum)
		return nil
	})
}

func runIssueLabelClear(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would clear all labels from issue #%d\n", issueNum)
		return nil
	}

	resp, err := client.ClearIssueLabels(owner, repo, issueNum)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	return printer.Emit(map[string]any{"status": "cleared", "issue": issueNum}, func() error {
		printer.Printf("Cleared all labels from issue #%d\n", issueNum)
		return nil
	})
}

func runIssueAssign(cmd *cobra.Command, args []string) error {
	return updateAssignees(cmd, args, true)
}

func runIssueUnassign(cmd *cobra.Command, args []string) error {
	return updateAssignees(cmd, args, false)
}

// updateAssignees adds or removes assignees. The API replaces the whole set, so
// the current assignees are read first and merged.
func updateAssignees(cmd *cobra.Command, args []string, add bool) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}

	issueNum := utils.ParseInt64(args[1])
	users := args[2:]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	issue, resp, err := client.GetIssue(owner, repo, issueNum)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	current := map[string]bool{}
	for _, a := range issue.Assignees {
		current[a.UserName] = true
	}
	for _, u := range users {
		if add {
			current[u] = true
		} else {
			delete(current, u)
		}
	}

	assignees := make([]string, 0, len(current))
	for u := range current {
		assignees = append(assignees, u)
	}
	sort.Strings(assignees)

	printer := getPrinter(cmd)

	verb := "assign"
	if !add {
		verb = "unassign"
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would %s %s on issue #%d\n", verb, strings.Join(users, ", "), issueNum)
		printer.Printf("  Resulting assignees: %s\n", strings.Join(assignees, ", "))
		return nil
	}

	updated, resp, err := client.EditIssue(owner, repo, issueNum, gitea.EditIssueOption{Assignees: assignees})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", strconv.FormatInt(issueNum, 10))
	}

	return printer.Emit(updated, func() error {
		printer.Printf("Issue #%d assignees: %s\n", issueNum, strings.Join(assignees, ", "))
		return nil
	})
}

func runIssueCommentDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	commentID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete comment #%d\n", commentID)
		return nil
	}

	resp, err := client.DeleteIssueComment(owner, repo, commentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue comment", strconv.FormatInt(commentID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "comment": commentID}, func() error {
		printer.Printf("Deleted comment #%d\n", commentID)
		return nil
	})
}
