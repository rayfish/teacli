// Package cmd contains pull request review deletion command
package cmd

import (
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prReviewDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <pr-number> <review-id>",
	Short: "Delete a review (admin only)",
	Long:  `Delete a pull request review. Requires admin privileges on the repository.`,
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRReviewDelete,
}

func init() {
	prCmd.AddCommand(prReviewDeleteCmd)
}

func runPRReviewDelete(cmd *cobra.Command, args []string) error {
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

	resp, err := client.DeletePullReview(owner, repo, prNum, reviewID)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "review": reviewID, "number": prNum}, func() error {
		printer.Printf("Deleted review #%d from PR #%d\n", reviewID, prNum)
		return nil
	})
}
