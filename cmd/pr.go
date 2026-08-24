// Package cmd contains pull request commands
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prCmd = &cobra.Command{
	Use:     "pr",
	Aliases: []string{"pull", "pulls", "pull-request", "pull-requests"},
	Short:   "Manage pull requests",
	Long:    `List, create, merge, close, and manage pull requests`,
}

var prListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List pull requests",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPRList,
}

var prGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <pr-number>",
	Short: "Get a specific pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRGet,
}

var prCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new pull request",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runPRCreate,
}

var prMergeCmd = &cobra.Command{
	Use:   "merge [<owner>/<repo>] <pr-number>",
	Short: "Merge a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRMerge,
}

var prCloseCmd = &cobra.Command{
	Use:   "close [<owner>/<repo>] <pr-number>",
	Short: "Close a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRClose,
}

var prReopenCmd = &cobra.Command{
	Use:   "reopen [<owner>/<repo>] <pr-number>",
	Short: "Reopen a closed pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReopen,
}

var prCommentsCmd = &cobra.Command{
	Use:   "comments [<owner>/<repo>] <pr-number>",
	Short: "List comments on a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRComments,
}

var prReviewsCmd = &cobra.Command{
	Use:   "reviews [<owner>/<repo>] <pr-number>",
	Short: "List reviews on a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReviews,
}

var prReviewCommentsCmd = &cobra.Command{
	Use:   "review-comments [<owner>/<repo>] <pr-number> <review-id>",
	Short: "List review comments for a specific review",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRReviewComments,
}

var prReplyCmd = &cobra.Command{
	Use:   "reply [<owner>/<repo>] <pr-number>",
	Short: "Reply to a PR comment or review comment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReply,
}

var prApproveCmd = &cobra.Command{
	Use:   "approve [<owner>/<repo>] <pr-number>",
	Short: "Approve a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRApprove,
}

var prRejectCmd = &cobra.Command{
	Use:   "reject [<owner>/<repo>] <pr-number>",
	Short: "Request changes on a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReject,
}

var prDismissCmd = &cobra.Command{
	Use:   "dismiss [<owner>/<repo>] <pr-number> <review-id>",
	Short: "Dismiss a review",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRDismiss,
}

var prDiffCmd = &cobra.Command{
	Use:   "diff [<owner>/<repo>] <pr-number>",
	Short: "Get PR diff",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRDiff,
}

func init() {
	RootCmd.AddCommand(prCmd)
	prCmd.AddCommand(prListCmd, prGetCmd, prCreateCmd, prMergeCmd, prCloseCmd, prReopenCmd, prCommentsCmd, prReviewsCmd, prReviewCommentsCmd, prReplyCmd, prApproveCmd, prRejectCmd, prDismissCmd, prDiffCmd)

	prCreateCmd.Flags().StringP("head", "H", "", "Source branch (required)")
	prCreateCmd.MarkFlagRequired("head")
	prCreateCmd.Flags().StringP("base", "B", "", "Target branch (required)")
	prCreateCmd.MarkFlagRequired("base")
	prCreateCmd.Flags().StringP("title", "t", "", "PR title (required)")
	prCreateCmd.MarkFlagRequired("title")
	prCreateCmd.Flags().StringP("body", "b", "", "PR description")

	prMergeCmd.Flags().StringP("method", "m", "merge", "Merge method: merge, rebase, rebase-merge, squash")
	prMergeCmd.Flags().Bool("delete-branch", false, "Delete the head branch after merging")

	prCommentsCmd.Flags().Int("page", 0, "Page number")
	prCommentsCmd.Flags().Int("per-page", 0, "Results per page")

	prReviewsCmd.Flags().Int("page", 0, "Page number")
	prReviewsCmd.Flags().Int("per-page", 0, "Results per page")

	prListCmd.Flags().StringP("state", "s", "open", "Filter by state: open, closed, all")
	prListCmd.Flags().Int("page", 0, "Page number")
	prListCmd.Flags().Int("per-page", 0, "Results per page")

	prReplyCmd.Flags().StringP("body", "b", "", "Reply text (required)")
	prReplyCmd.MarkFlagRequired("body")
	prReplyCmd.Flags().Int("comment-id", 0, "ID of comment to reference (for inline review comments)")
	prReplyCmd.Flags().String("file", "", "File path for inline review comment (required with --comment-id)")
	prReplyCmd.Flags().Int("line", 0, "Line number for inline review comment (required with --comment-id)")

	prApproveCmd.Flags().StringP("body", "b", "", "Approval message")
	prRejectCmd.Flags().StringP("body", "b", "", "Rejection/request changes message")
	prDismissCmd.Flags().StringP("message", "m", "", "Dismissal message")
}

func runPRList(cmd *cobra.Command, args []string) error {
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

	prs, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.PullRequest, *gitea.Response, error) {
		return client.ListRepoPullRequests(owner, repo, gitea.ListPullRequestsOptions{
			ListOptions: lo,
			State:       stateType,
		})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	headers := []string{"#", "Title", "State"}
	var rows [][]string

	for _, pr := range prs {
		rows = append(rows, []string{
			fmt.Sprintf("%d", pr.Index),
			utils.Truncate(pr.Title, 40),
			string(pr.State),
		})
	}

	return printer.Emit(prs, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runPRGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	pr, resp, err := client.GetPullRequest(owner, repo, prNum)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)

	return printer.Emit(pr, func() error {
		fmt.Printf("PR #%d\n", pr.Index)
		fmt.Printf("Title:       %s\n", pr.Title)
		fmt.Printf("State:       %s\n", pr.State)
		fmt.Printf("URL:         %s\n", pr.HTMLURL)
		fmt.Printf("Diff URL:    %s\n", pr.DiffURL)
		fmt.Printf("Patch URL:   %s\n", pr.PatchURL)
		fmt.Printf("Draft:       %t\n", pr.Draft)
		fmt.Printf("Is Locked:   %t\n", pr.IsLocked)
		fmt.Printf("Mergeable:   %t\n", pr.Mergeable)
		fmt.Printf("Comments:    %d\n", pr.Comments)
		fmt.Printf("Review Cmts: %d\n", pr.ReviewComments)

		if pr.Head != nil {
			fmt.Printf("\nHead:\n")
			fmt.Printf("  Branch:  %s\n", pr.Head.Ref)
			fmt.Printf("  SHA:     %s\n", pr.Head.Sha)
			if pr.Head.Repository != nil {
				fmt.Printf("  Repo:    %s\n", pr.Head.Repository.FullName)
			}
		}

		if pr.Base != nil {
			fmt.Printf("\nBase:\n")
			fmt.Printf("  Branch:  %s\n", pr.Base.Ref)
			fmt.Printf("  SHA:     %s\n", pr.Base.Sha)
			if pr.Base.Repository != nil {
				fmt.Printf("  Repo:    %s\n", pr.Base.Repository.FullName)
			}
		}

		if pr.Poster != nil {
			fmt.Printf("\nAuthor:      %s (%s)\n", pr.Poster.FullName, pr.Poster.UserName)
		}

		if pr.HasMerged {
			fmt.Printf("\nMerged:      true\n")
			if pr.Merged != nil {
				fmt.Printf("Merged At:   %s\n", pr.Merged.Format("2006-01-02 15:04:05"))
			}
			if pr.MergedBy != nil {
				fmt.Printf("Merged By:   %s\n", pr.MergedBy.FullName)
			}
		}

		if pr.Created != nil {
			fmt.Printf("\nCreated:     %s\n", pr.Created.Format("2006-01-02 15:04:05"))
		}
		if pr.Updated != nil {
			fmt.Printf("Updated:     %s\n", pr.Updated.Format("2006-01-02 15:04:05"))
		}
		if pr.Closed != nil {
			fmt.Printf("Closed:      %s\n", pr.Closed.Format("2006-01-02 15:04:05"))
		}

		return nil
	})
}

func runPRCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	head, _ := cmd.Flags().GetString("head")
	base, _ := cmd.Flags().GetString("base")
	title, _ := cmd.Flags().GetString("title")
	body, _ := cmd.Flags().GetString("body")

	if head == "" || base == "" || title == "" {
		return errors.NewValidationError("--head, --base, and --title are required",
			map[string]interface{}{"missing_flags": []string{"--head", "--base", "--title"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreatePullRequestOption{
		Head:  head,
		Base:  base,
		Title: title,
		Body:  body,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create PR from '%s' to '%s'\n", head, base)
		printer.Printf("  Title: %s\n", title)
		if body != "" {
			printer.Printf("  Body: %s\n", utils.Truncate(body, 80))
		}
		return nil
	}

	pr, resp, err := client.CreatePullRequest(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(pr, func() error {
		printer.Printf("%d %s\n", pr.Index, pr.HTMLURL)
		return nil
	})
}

func runPRMerge(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	method, _ := cmd.Flags().GetString("method")

	var mergeMethod gitea.MergeStyle
	switch method {
	case "merge":
		mergeMethod = gitea.MergeStyleMerge
	case "rebase":
		mergeMethod = gitea.MergeStyleRebase
	case "rebase-merge":
		mergeMethod = gitea.MergeStyleRebaseMerge
	case "squash":
		mergeMethod = gitea.MergeStyleSquash
	default:
		return errors.NewValidationError(fmt.Sprintf("invalid merge method: %s", method),
			map[string]interface{}{"valid": []string{"merge", "rebase", "rebase-merge", "squash"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	deleteBranch, _ := cmd.Flags().GetBool("delete-branch")

	opts := gitea.MergePullRequestOption{
		Style:                  mergeMethod,
		DeleteBranchAfterMerge: deleteBranch,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would merge PR #%d using %s method\n", prNum, method)
		printer.Printf("  Delete source branch: %t\n", deleteBranch)
		return nil
	}

	merged, resp, err := client.MergePullRequest(owner, repo, prNum, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "merged", "number": prNum, "merged": merged}, func() error {
		printer.Printf("Merged PR #%d\n", prNum)
		return nil
	})
}

func runPRClose(cmd *cobra.Command, args []string) error {
	return updatePRState(cmd, args, "closed")
}

func runPRReopen(cmd *cobra.Command, args []string) error {
	return updatePRState(cmd, args, "open")
}

func updatePRState(cmd *cobra.Command, args []string, state string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	st := gitea.StateType(state)
	editOpts := gitea.EditPullRequestOption{
		State: &st,
	}

	pr, resp, err := client.EditPullRequest(owner, repo, prNum, editOpts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)
	return printer.Emit(pr, func() error {
		printer.Printf("%s PR #%d\n", strings.ToUpper(state), pr.Index)
		return nil
	})
}

func runPRComments(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	comments, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Comment, *gitea.Response, error) {
		return client.ListIssueComments(owner, repo, prNum, gitea.ListIssueCommentOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	headers := []string{"#", "Author", "Created", "Body"}
	var rows [][]string

	for _, c := range comments {
		body := utils.Truncate(strings.ReplaceAll(c.Body, "\n", " "), 50)
		created := c.Created.Format("2006-01-02")
		author := ""
		if c.Poster != nil {
			author = c.Poster.UserName
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", c.ID),
			author,
			created,
			body,
		})
	}

	return printer.Emit(comments, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runPRReviews(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	reviews, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.PullReview, *gitea.Response, error) {
		return client.ListPullReviews(owner, repo, prNum, gitea.ListPullReviewsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	headers := []string{"#", "Author", "State", "Submitted", "Body"}
	var rows [][]string

	for _, r := range reviews {
		body := ""
		if r.Body != "" {
			body = utils.Truncate(strings.ReplaceAll(r.Body, "\n", " "), 40)
		} else {
			body = "-"
		}
		submitted := r.Submitted.Format("2006-01-02")
		author := ""
		if r.Reviewer != nil {
			author = r.Reviewer.UserName
		}
		rows = append(rows, []string{
			fmt.Sprintf("%d", r.ID),
			author,
			string(r.State),
			submitted,
			body,
		})
	}

	return printer.Emit(reviews, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runPRReviewComments(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	reviewID := utils.ParseInt64(args[2])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	comments, resp, err := client.ListPullReviewComments(owner, repo, prNum, reviewID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request review", fmt.Sprintf("%d", reviewID))
	}

	printer := getPrinter(cmd)

	headers := []string{"File", "Line", "Author", "Status", "Comment"}

	sort.Slice(comments, func(i, j int) bool {
		if comments[i].Path != comments[j].Path {
			return comments[i].Path < comments[j].Path
		}
		return comments[i].LineNum < comments[j].LineNum
	})

	var rows [][]string

	for _, c := range comments {
		file := c.Path
		line := fmt.Sprintf("%d", c.LineNum)
		status := "open"
		if c.Resolver != nil {
			status = fmt.Sprintf("resolved by %s", c.Resolver.UserName)
		}
		comment := utils.Truncate(strings.ReplaceAll(c.Body, "\n", " "), 30)
		author := ""
		if c.Reviewer != nil {
			author = c.Reviewer.UserName
		}
		rows = append(rows, []string{file, line, author, status, comment})
	}

	return printer.Emit(comments, func() error {
		return printer.PrintTable(headers, rows)
	})
}

func runPRReply(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	body, _ := cmd.Flags().GetString("body")
	commentID, _ := cmd.Flags().GetInt("comment-id")
	filePath, _ := cmd.Flags().GetString("file")
	line, _ := cmd.Flags().GetInt("line")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if commentID > 0 {
		if filePath == "" || line == 0 {
			return errors.NewValidationError("when replying to a review comment, you must specify --file and --line",
				map[string]interface{}{"missing": []string{"--file", "--line"}})
		}

		opts := gitea.CreatePullReviewOptions{
			State: "COMMENT",
			Body:  body,
			Comments: []gitea.CreatePullReviewComment{
				{
					Path:       filePath,
					Body:       body,
					NewLineNum: int64(line),
				},
			},
		}

		review, resp, err := client.CreatePullReview(owner, repo, prNum, opts)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
		}

		printer := getPrinter(cmd)
		return printer.Emit(review, func() error {
			printer.Printf("Review comment created successfully!\n")
			printer.Printf("Review ID: %d\n", review.ID)
			printer.Printf("File:      %s\n", filePath)
			printer.Printf("Line:      %d\n", line)
			printer.Printf("\n%s\n", body)
			return nil
		})
	}

	opts := gitea.CreateIssueCommentOption{
		Body: body,
	}

	comment, resp, err := client.CreateIssueComment(owner, repo, prNum, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)
	return printer.Emit(comment, func() error {
		printer.Printf("Comment created successfully!\n")
		printer.Printf("ID:        %d\n", comment.ID)
		printer.Printf("URL:       %s\n", comment.HTMLURL)
		printer.Printf("Created:   %s\n", comment.Created.Format("2006-01-02 15:04:05"))
		printer.Printf("\n%s\n", comment.Body)
		return nil
	})
}

func runPRApprove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	body, _ := cmd.Flags().GetString("body")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreatePullReviewOptions{
		State: "APPROVE",
		Body:  body,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would approve PR #%d\n", prNum)
		if body != "" {
			printer.Printf("  Message: %s\n", body)
		}
		return nil
	}

	review, resp, err := client.CreatePullReview(owner, repo, prNum, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)
	return printer.Emit(review, func() error {
		printer.Printf("Approved PR #%d (Review ID: %d)\n", prNum, review.ID)
		return nil
	})
}

func runPRReject(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	body, _ := cmd.Flags().GetString("body")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreatePullReviewOptions{
		State: "REQUEST_CHANGES",
		Body:  body,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would request changes on PR #%d\n", prNum)
		if body != "" {
			printer.Printf("  Message: %s\n", body)
		}
		return nil
	}

	review, resp, err := client.CreatePullReview(owner, repo, prNum, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	printer := getPrinter(cmd)
	return printer.Emit(review, func() error {
		printer.Printf("Requested changes on PR #%d (Review ID: %d)\n", prNum, review.ID)
		return nil
	})
}

func runPRDismiss(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	reviewID := utils.ParseInt64(args[2])
	message, _ := cmd.Flags().GetString("message")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.DismissPullReviewOptions{
		Message: message,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would dismiss review #%d on PR #%d\n", reviewID, prNum)
		if message != "" {
			printer.Printf("  Message: %s\n", message)
		}
		return nil
	}

	resp, err := client.DismissPullReview(owner, repo, prNum, reviewID, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request review", fmt.Sprintf("%d", reviewID))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "dismissed", "number": prNum, "review_id": reviewID}, func() error {
		printer.Printf("Dismissed review #%d on PR #%d\n", reviewID, prNum)
		return nil
	})
}

func runPRDiff(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	diff, resp, err := client.GetPullRequestDiff(owner, repo, prNum, gitea.PullRequestDiffOptions{})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", fmt.Sprintf("%d", prNum))
	}

	// Parse and annotate diff with line numbers
	annotatedDiff := annotateDiffWithLineNumbers(string(diff))

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"diff": annotatedDiff}, func() error {
		fmt.Print(annotatedDiff)
		return nil
	})
}

// annotateDiffWithLineNumbers parses unified diff and adds line number annotations
func annotateDiffWithLineNumbers(diff string) string {
	lines := strings.Split(diff, "\n")
	var result []string

	inHunk := false
	hunkOldStart := 0
	hunkNewStart := 0

	oldLinePos := 0 // position in old file
	newLinePos := 0 // position in new file

	for _, line := range lines {
		switch {
		case strings.HasPrefix(line, "diff --git"):
			// New file, reset counters
			result = append(result, line)
			oldLinePos = 0
			newLinePos = 0
			inHunk = false

		case strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ "):
			result = append(result, line)

		case strings.HasPrefix(line, "@@ "):
			// Hunk header - parse line numbers
			inHunk = true
			parseHunkHeader(line, &hunkOldStart, nil, &hunkNewStart, nil)
			oldLinePos = hunkOldStart
			newLinePos = hunkNewStart
			result = append(result, line)

		case strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++"):
			// Added line - show new file line number
			if inHunk {
				result = append(result, fmt.Sprintf("%d: %s", newLinePos, line))
				newLinePos++
			} else {
				result = append(result, line)
			}

		case strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---"):
			// Removed line - show old file line number
			if inHunk {
				result = append(result, fmt.Sprintf("%d: %s", oldLinePos, line))
				oldLinePos++
			} else {
				result = append(result, line)
			}

		case strings.HasPrefix(line, " "):
			// Unchanged line - no annotation needed
			if inHunk {
				oldLinePos++
				newLinePos++
			}
			result = append(result, line)

		default:
			// Other lines (index, mode, etc.)
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}

// parseHunkHeader parses "@@ -oldStart,oldLen +newStart,newLen @@"
func parseHunkHeader(line string, oldStart, oldLen, newStart, newLen *int) {
	// Format: @@ -oldStart,oldLen +newStart,newLen @@
	parts := strings.Fields(line)
	if len(parts) < 2 {
		return
	}

	// Parse old side: -oldStart,oldLen or -oldStart
	oldPart := strings.TrimPrefix(parts[1], "-")
	if commaIdx := strings.Index(oldPart, ","); commaIdx != -1 {
		if oldStart != nil {
			fmt.Sscanf(oldPart[:commaIdx], "%d", oldStart)
		}
		if oldLen != nil {
			fmt.Sscanf(oldPart[commaIdx+1:], "%d", oldLen)
		}
	} else {
		if oldStart != nil {
			fmt.Sscanf(oldPart, "%d", oldStart)
		}
		if oldLen != nil {
			*oldLen = 1
		}
	}

	// Parse new side: +newStart,newLen or +newStart
	newPart := strings.TrimPrefix(parts[2], "+")
	if commaIdx := strings.Index(newPart, ","); commaIdx != -1 {
		if newStart != nil {
			fmt.Sscanf(newPart[:commaIdx], "%d", newStart)
		}
		if newLen != nil {
			fmt.Sscanf(newPart[commaIdx+1:], "%d", newLen)
		}
	} else {
		if newStart != nil {
			fmt.Sscanf(newPart, "%d", newStart)
		}
		if newLen != nil {
			*newLen = 1
		}
	}
}
