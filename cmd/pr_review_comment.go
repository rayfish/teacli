package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

var prReviewCommentCmd = &cobra.Command{
	Use:   "comment",
	Short: "Manage individual review comments (Forgejo API)",
}
var prReviewCommentGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <pr-number> <review-id> <comment-id>",
	Short: "Get a review comment (Forgejo)",
	Args:  cobra.RangeArgs(3, 4),
	RunE:  runPRReviewCommentGet,
}
var prReviewCommentCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <pr-number> <review-id>",
	Short: "Add a comment to an existing review (Forgejo)",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRReviewCommentCreate,
}
var prReviewCommentDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <pr-number> <review-id> <comment-id>",
	Short: "Delete a review comment (Forgejo)",
	Args:  cobra.RangeArgs(3, 4),
	RunE:  runPRReviewCommentDelete,
}

func init() {
	prReviewCmd.AddCommand(prReviewCommentCmd)
	prReviewCommentCmd.AddCommand(prReviewCommentGetCmd, prReviewCommentCreateCmd, prReviewCommentDeleteCmd)
	prReviewCommentCreateCmd.Flags().StringP("body", "b", "", "Comment body")
	prReviewCommentCreateCmd.Flags().String("body-file", "", "Read the body from a file, or - for stdin")
	prReviewCommentCreateCmd.Flags().String("file", "", "Path of the file being reviewed")
	prReviewCommentCreateCmd.MarkFlagRequired("file")
	prReviewCommentCreateCmd.Flags().Int64("line", 0, "Line number to comment on")
	prReviewCommentCreateCmd.MarkFlagRequired("line")
	prReviewCommentCreateCmd.Flags().Bool("old", false, "Comment on the old version of the file")
	prReviewCommentCreateCmd.Flags().Int64("extra-lines", 0, "Additional lines after the commented line")
}

func reviewCommentPath(cmd *cobra.Command, args []string) (string, error) {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 3)
	if err != nil {
		return "", err
	}
	reviewID, err := int64Arg(args[2], "review id")
	if err != nil {
		return "", err
	}
	commentID, err := int64Arg(args[3], "comment id")
	if err != nil {
		return "", err
	}
	if reviewID <= 0 || commentID <= 0 {
		return "", errors.NewValidationError("review and comment IDs must be positive", nil)
	}
	return fmt.Sprintf("%s/reviews/%d/comments/%d", pullRequestPath(owner, repo, index), reviewID, commentID), nil
}

func runPRReviewCommentGet(cmd *cobra.Command, args []string) error {
	path, err := reviewCommentPath(cmd, args)
	if err != nil {
		return err
	}
	data, err := apiRequest(cmd, http.MethodGet, path, nil, http.StatusOK)
	if err != nil {
		return err
	}
	return emitPRReviewComment(cmd, data)
}

func runPRReviewCommentDelete(cmd *cobra.Command, args []string) error {
	path, err := reviewCommentPath(cmd, args)
	if err != nil {
		return err
	}
	if _, err := getServer(cmd); err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete review comment at %s", path) {
		return nil
	}
	if _, err := apiRequest(cmd, http.MethodDelete, path, nil, http.StatusNoContent); err != nil {
		return err
	}
	return emitMessage(cmd, okMessage("review comment deleted"), "Deleted review comment")
}

func runPRReviewCommentCreate(cmd *cobra.Command, args []string) error {
	owner, repo, index, args, err := repoAndIndex(cmd, args, 2)
	if err != nil {
		return err
	}
	reviewID, err := int64Arg(args[2], "review id")
	if err != nil {
		return err
	}
	body, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}
	file, _ := cmd.Flags().GetString("file")
	line, _ := cmd.Flags().GetInt64("line")
	extra, _ := cmd.Flags().GetInt64("extra-lines")
	old, _ := cmd.Flags().GetBool("old")
	if reviewID <= 0 || line <= 0 || extra < 0 || file == "" || body == "" {
		return errors.NewValidationError("positive review ID and --line, nonempty --file and body, and nonnegative --extra-lines are required", nil)
	}
	payload := struct {
		Body        string `json:"body"`
		Path        string `json:"path"`
		NewPosition int64  `json:"new_position"`
		OldPosition int64  `json:"old_position"`
		ExtraLines  int64  `json:"extra_lines_count"`
	}{Body: body, Path: file, ExtraLines: extra}
	if old {
		payload.OldPosition = line
	} else {
		payload.NewPosition = line
	}
	if _, err := getServer(cmd); err != nil {
		return err
	}
	if dryRunf(cmd, "Would add comment to review #%d on %s/%s#%d", reviewID, owner, repo, index) {
		return nil
	}
	data, err := apiRequest(cmd, http.MethodPost, fmt.Sprintf("%s/reviews/%d/comments", pullRequestPath(owner, repo, index), reviewID), payload, http.StatusOK)
	if err != nil {
		return err
	}
	return emitPRReviewComment(cmd, data)
}

func emitPRReviewComment(cmd *cobra.Command, data []byte) error {
	var comment gitea.PullReviewComment
	if err := json.Unmarshal(data, &comment); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to decode review comment: %v", err))
	}
	printer := getPrinter(cmd)
	return printer.Emit(comment, func() error {
		var d detail
		d.always("ID", comment.ID)
		d.add("Author", userName(comment.Reviewer))
		d.add("File", comment.Path)
		d.always("Line", comment.LineNum)
		d.add("Body", comment.Body)
		return d.print(printer)
	})
}
