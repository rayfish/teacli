// Package cmd contains the pull request commands beyond create/merge: edit,
// commits, files, patch, review requests and merge checks.
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <pr-number>",
	Aliases: []string{"edit"},
	Short:   "Edit a pull request's title, body, base branch or assignees",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runPRUpdate,
}

var prCommitsCmd = &cobra.Command{
	Use:   "commits [<owner>/<repo>] <pr-number>",
	Short: "List the commits in a pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRCommits,
}

var prFilesCmd = &cobra.Command{
	Use:   "files [<owner>/<repo>] <pr-number>",
	Short: "List the files a pull request changes",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRFiles,
}

var prPatchCmd = &cobra.Command{
	Use:   "patch [<owner>/<repo>] <pr-number>",
	Short: "Print a pull request as a git patch",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRPatch,
}

var prMergedCmd = &cobra.Command{
	Use:   "merged [<owner>/<repo>] <pr-number>",
	Short: "Report whether a pull request is merged (exit 3 if not)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRMerged,
}

var prReviewRequestCmd = &cobra.Command{
	Use:     "request-review [<owner>/<repo>] <pr-number> <reviewer>...",
	Aliases: []string{"request"},
	Short:   "Request reviews on a pull request",
	Args:    cobra.MinimumNArgs(2),
	RunE:    runPRRequestReview,
}

var prReviewUnrequestCmd = &cobra.Command{
	Use:     "unrequest-review [<owner>/<repo>] <pr-number> <reviewer>...",
	Aliases: []string{"unrequest"},
	Short:   "Cancel review requests on a pull request",
	Args:    cobra.MinimumNArgs(2),
	RunE:    runPRUnrequestReview,
}

var prReviewGetCmd = &cobra.Command{
	Use:   "review-get [<owner>/<repo>] <pr-number> <review-id>",
	Short: "Get one review",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRReviewGet,
}

var prUndismissCmd = &cobra.Command{
	Use:   "undismiss [<owner>/<repo>] <pr-number> <review-id>",
	Short: "Undo the dismissal of a review",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRUndismiss,
}

func init() {
	prCmd.AddCommand(prUpdateCmd, prCommitsCmd, prFilesCmd, prPatchCmd, prMergedCmd,
		prReviewRequestCmd, prReviewUnrequestCmd, prReviewGetCmd, prUndismissCmd)

	addPageFlags(prCommitsCmd, prFilesCmd)

	prUpdateCmd.Flags().StringP("title", "t", "", "New title")
	prUpdateCmd.Flags().StringP("body", "b", "", "New body")
	prUpdateCmd.Flags().String("body-file", "", "Read the new body from this file, or - for stdin")
	prUpdateCmd.Flags().String("base", "", "New base branch")
	prUpdateCmd.Flags().StringSliceP("assignee", "a", nil,
		"Replace the assignee set with these users (repeatable)")
	prUpdateCmd.Flags().StringSliceP("label", "l", nil, "Replace the label set with these label IDs")
	prUpdateCmd.Flags().Int64P("milestone", "m", 0, "Milestone ID to set")
	prUpdateCmd.Flags().StringP("state", "s", "", "New state: open or closed")
	prUpdateCmd.Flags().Bool("allow-maintainer-edit", false, "Let maintainers push to the head branch")
	prUpdateCmd.Flags().Bool("no-allow-maintainer-edit", false, "Stop maintainers pushing to the head branch")

	prReviewRequestCmd.Flags().Bool("team", false, "Treat the reviewer arguments as team names")
	prReviewUnrequestCmd.Flags().Bool("team", false, "Treat the reviewer arguments as team names")
}

func runPRUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}

	body, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}

	opts := gitea.EditPullRequestOption{}
	if cmd.Flags().Changed("title") {
		opts.Title, _ = cmd.Flags().GetString("title")
	}
	if body != "" {
		opts.Body = &body
	}
	if cmd.Flags().Changed("base") {
		opts.Base, _ = cmd.Flags().GetString("base")
	}
	if cmd.Flags().Changed("assignee") {
		opts.Assignees, _ = cmd.Flags().GetStringSlice("assignee")
	}
	if cmd.Flags().Changed("label") {
		raw, _ := cmd.Flags().GetStringSlice("label")
		labels := make([]int64, 0, len(raw))
		for _, l := range raw {
			id, err := int64Arg(l, "label id")
			if err != nil {
				return err
			}
			labels = append(labels, id)
		}
		opts.Labels = labels
	}
	if cmd.Flags().Changed("milestone") {
		ms, _ := cmd.Flags().GetInt64("milestone")
		opts.Milestone = ms
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
	opts.AllowMaintainerEdit = boolPairFlag(cmd, "allow-maintainer-edit", "no-allow-maintainer-edit")

	if opts.Title == "" && opts.Body == nil && opts.Base == "" && opts.Assignees == nil &&
		opts.Labels == nil && opts.Milestone == 0 && opts.State == nil && opts.AllowMaintainerEdit == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --title"})
	}

	if dryRunf(cmd, "Would edit pull request %s/%s#%d", owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	pr, resp, err := client.EditPullRequest(owner, repo, index, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", args[1])
	}

	return emitMessage(cmd, pr, "Updated %s/%s#%d: %s", owner, repo, index, pr.Title)
}

func runPRCommits(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	commits, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Commit, *gitea.Response, error) {
		return client.ListPullRequestCommits(owner, repo, index,
			gitea.ListPullRequestCommitsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitCommitTable(cmd, commits)
}

func runPRFiles(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	files, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.ChangedFile, *gitea.Response, error) {
		return client.ListPullRequestFiles(owner, repo, index,
			gitea.ListPullRequestFilesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(files, func() error {
		if len(files) == 0 {
			printer.Println("No changed files.")
			return nil
		}
		var additions, deletions int
		rows := make([][]string, 0, len(files))
		for _, f := range files {
			additions += f.Additions
			deletions += f.Deletions
			rows = append(rows, []string{
				f.Status, fmt.Sprintf("+%d", f.Additions), fmt.Sprintf("-%d", f.Deletions), f.Filename,
			})
		}
		if err := printer.PrintTable([]string{"Status", "Added", "Removed", "File"}, rows); err != nil {
			return err
		}
		printer.Printf("\n%d file(s), +%d -%d\n", len(files), additions, deletions)
		return nil
	})
}

func runPRPatch(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	patch, resp, err := client.GetPullRequestPatch(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]string{"patch": string(patch)}, func() error {
		printer.Println(strings.TrimRight(string(patch), "\n"))
		return nil
	})
}

func runPRMerged(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	merged, resp, err := client.IsPullRequestMerged(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", args[1])
	}
	// Exit 3 on "not merged" so a script can branch on the code rather than
	// parse the message.
	if !merged {
		return errors.NewNotFoundError("merged pull request", args[1])
	}

	return emitMessage(cmd, map[string]any{"merged": true},
		"%s/%s#%d is merged", owner, repo, index)
}

func runPRRequestReview(cmd *cobra.Command, args []string) error {
	return changeReviewRequests(cmd, args, true)
}

func runPRUnrequestReview(cmd *cobra.Command, args []string) error {
	return changeReviewRequests(cmd, args, false)
}

func changeReviewRequests(cmd *cobra.Command, args []string, add bool) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "pull request number")
	if err != nil {
		return err
	}
	reviewers := args[2:]
	asTeam, _ := cmd.Flags().GetBool("team")

	opts := gitea.PullReviewRequestOptions{}
	if asTeam {
		opts.TeamReviewers = reviewers
	} else {
		opts.Reviewers = reviewers
	}

	verb := "cancel review requests for"
	if add {
		verb = "request reviews from"
	}
	if dryRunf(cmd, "Would %s %s on %s/%s#%d", verb, strings.Join(reviewers, ", "), owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if add {
		resp, err := client.CreateReviewRequests(owner, repo, index, opts)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "pull request", args[1])
		}
		return emitMessage(cmd, okMessage("reviews requested"),
			"Requested %d review(s) on %s/%s#%d", len(reviewers), owner, repo, index)
	}

	resp, err := client.DeleteReviewRequests(owner, repo, index, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "pull request", args[1])
	}
	return emitMessage(cmd, okMessage("review requests cancelled"),
		"Cancelled %d review request(s) on %s/%s#%d", len(reviewers), owner, repo, index)
}

func runPRReviewGet(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 2)
	if err != nil {
		return err
	}
	reviewID, err := int64Arg(args[2], "review id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	review, resp, err := client.GetPullReview(owner, repo, index, reviewID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "review", args[2])
	}

	printer := getPrinter(cmd)
	return printer.Emit(review, func() error {
		var d detail
		d.always("ID", review.ID)
		d.add("Reviewer", userName(review.Reviewer))
		d.add("State", string(review.State))
		d.always("Comments", review.CodeCommentsCount)
		d.always("Stale", review.Stale)
		d.always("Official", review.Official)
		d.always("Dismissed", review.Dismissed)
		d.add("Commit", review.CommitID)
		d.add("Submitted", review.Submitted)
		d.add("URL", review.HTMLURL)
		d.add("Body", review.Body)
		return d.print(printer)
	})
}

func runPRUndismiss(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 2)
	if err != nil {
		return err
	}
	reviewID, err := int64Arg(args[2], "review id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would undismiss review %d on %s/%s#%d", reviewID, owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.UnDismissPullReview(owner, repo, index, reviewID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "review", args[2])
	}

	return emitMessage(cmd, okMessage("review undismissed"), "Undismissed review %d", reviewID)
}
