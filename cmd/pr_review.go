// Package cmd contains pull request review commands
package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prReviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Submit pull request review with inline comments",
	Long:  `Submit a review with inline comments on specific lines in a single operation.`,
}

var prReviewSubmitCmd = &cobra.Command{
	Use:   "submit [<owner>/<repo>] <pr-number> --state approve|request-changes|comment [-b body] [--comment file:line:body]...",
	Short: "Submit review with inline comments",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRReviewSubmit,
}

var inlineComments []string

func init() {
	prCmd.AddCommand(prReviewCmd)
	prReviewCmd.AddCommand(prReviewSubmitCmd)

	prReviewSubmitCmd.Flags().StringP("state", "s", "comment", "Review state: approve, request-changes, comment")
	prReviewSubmitCmd.MarkFlagRequired("state")
	prReviewSubmitCmd.Flags().StringP("body", "b", "", "Review summary body")
	prReviewSubmitCmd.Flags().StringArrayVar(&inlineComments, "comment", []string{}, "Inline comment in format 'file:line:body' (can be repeated)")
}

func runPRReviewSubmit(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	prNum := utils.ParseInt64(args[1])
	state, _ := cmd.Flags().GetString("state")
	body, _ := cmd.Flags().GetString("body")

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
	default:
		return errors.NewValidationError("invalid state",
			map[string]interface{}{"valid_values": []string{"approve", "request-changes", "comment"}})
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
		if parseErr != nil {
			return errors.NewValidationError(fmt.Sprintf("invalid line number in comment: %v", parseErr), nil)
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
		Body:     body,
		Comments: comments,
	}

	review, resp, err := client.CreatePullReview(owner, repo, prNum, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
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
