// Package cmd contains pull request review commands
package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Submit pull request review with inline comments",
	Long:  `Submit a review with inline comments on specific lines in a single operation.`,
}

var prReviewSubmitCmd = &cobra.Command{
	Use:   "submit [<owner>/<repo>] <pr-number> --state approve|request-changes|comment|pending [-b body] [--review-id id] [--comment file:line:body]...",
	Short: "Submit review with inline comments",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReviewSubmit,
}

var inlineComments []string

func init() {
	prCmd.AddCommand(prReviewCmd)
	prReviewCmd.AddCommand(prReviewSubmitCmd)

	prReviewSubmitCmd.Flags().StringP("state", "s", "comment", "Review state: approve, request-changes, comment, pending")
	prReviewSubmitCmd.MarkFlagRequired("state")
	prReviewSubmitCmd.Flags().StringP("body", "b", "", "Review summary body")
	prReviewSubmitCmd.Flags().String("body-file", "", "Read the summary from a file, or - for stdin")
	prReviewSubmitCmd.Flags().Int64("review-id", 0, "Submit this pending review instead of creating a new one")
	prReviewSubmitCmd.Flags().String("commit", "", "Commit SHA to review when creating a review")
	prReviewSubmitCmd.Flags().StringArrayVar(&inlineComments, "comment", []string{}, "Inline comment in format 'file:line:body' (can be repeated)")
}

func runPRReviewSubmit(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum, err := int64Arg(args[1], "pull request number")
	if err != nil {
		return err
	}
	state, _ := cmd.Flags().GetString("state")
	body, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}
	reviewID, _ := cmd.Flags().GetInt64("review-id")
	commit, _ := cmd.Flags().GetString("commit")
	if reviewID < 0 {
		return errors.NewValidationError("review id must be positive", nil)
	}
	if cmd.Flags().Changed("review-id") && reviewID == 0 {
		return errors.NewValidationError("review id must be positive", nil)
	}
	if reviewID != 0 && (len(inlineComments) != 0 || commit != "") {
		return errors.NewValidationError("--review-id cannot be combined with --comment or --commit", nil)
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// Map state string to SDK enum
	var reviewState gitea.ReviewStateType
	switch state {
	case "approve":
		reviewState = "APPROVED"
	case "request-changes", "request_changes":
		reviewState = "REQUEST_CHANGES"
	case "comment":
		reviewState = "COMMENT"
	case "pending":
		if reviewID != 0 {
			return errors.NewValidationError("a pending review must be submitted as approve, request-changes, or comment", nil)
		}
		reviewState = "PENDING"
	default:
		return errors.NewValidationError("invalid state",
			map[string]interface{}{"valid_values": []string{"approve", "request-changes", "comment", "pending"}})
	}

	// Parse inline comments from --comment flags (format: "file:line:body")
	var comments []gitea.CreatePullReviewComment
	for _, commentStr := range inlineComments {
		parts := strings.SplitN(commentStr, ":", 3)
		if len(parts) != 3 {
			return errors.NewValidationError("invalid comment format, use 'file:line:body'",
				map[string]interface{}{"example": "--comment Cargo.toml:41:This needs fixing"})
		}
		filePath := parts[0]
		lineNum, parseErr := strconv.ParseInt(parts[1], 10, 64)
		if parseErr != nil || lineNum <= 0 {
			return errors.NewValidationError("comment line number must be a positive integer", nil)
		}
		commentBody := strings.ReplaceAll(parts[2], "\\n", "\n") // Convert \n to actual newlines

		comments = append(comments, gitea.CreatePullReviewComment{
			Path:       filePath,
			Body:       commentBody,
			NewLineNum: lineNum,
		})
	}

	opts := gitea.CreatePullReviewOptions{
		State:    reviewState,
		CommitID: commit,
		Body:     body,
		Comments: comments,
	}

	if dryRunf(cmd, "Would submit a %s review on %s/%s#%d", state, owner, repo, prNum) {
		return nil
	}

	var review *gitea.PullReview
	if reviewID != 0 {
		payload := gitea.SubmitPullReviewOptions{State: reviewState, Body: body}
		data, err := apiRequest(cmd, http.MethodPost, fmt.Sprintf("%s/reviews/%d", pullRequestPath(owner, repo, prNum), reviewID), payload, http.StatusOK)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &review); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to decode review: %v", err))
		}
		if review == nil {
			return errors.NewGeneralError("server returned no review")
		}
	} else {
		var resp *gitea.Response
		review, resp, err = client.CreatePullReview(owner, repo, prNum, opts)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
	}

	printer := getPrinter(cmd)
	return printer.Emit(review, func() error {
		printer.Printf("Submitted review #%d as %s with %d inline comments\n", review.ID, review.State, len(comments))
		if review.HTMLURL != "" {
			printer.Printf("View at: %s\n", review.HTMLURL)
		}
		return nil
	})
}
