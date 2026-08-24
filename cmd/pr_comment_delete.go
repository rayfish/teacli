// Package cmd contains pull request comment deletion command
package cmd

import (
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var prCommentDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <comment-id>",
	Short: "Delete a PR/issue comment",
	Long:  `Delete a comment on a pull request or issue by comment ID.`,
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRCommentDelete,
}

func init() {
	prCommentsCmd.AddCommand(prCommentDeleteCmd)
}

func runPRCommentDelete(cmd *cobra.Command, args []string) error {
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
		printer.Printf("DRY-RUN: Would delete comment #%d from %s/%s\n", commentID, owner, repo)
		return nil
	}

	resp, err := client.DeleteIssueComment(owner, repo, commentID)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "comment": commentID}, func() error {
		printer.Printf("Deleted comment #%d from %s/%s\n", commentID, owner, repo)
		return nil
	})
}
