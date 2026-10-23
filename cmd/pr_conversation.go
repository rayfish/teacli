package cmd

import (
	"fmt"
	"io"
	"net/http"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

var prResolveCmd = &cobra.Command{
	Use:   "resolve [<owner>/<repo>] <comment-id>",
	Short: "Resolve a pull request review conversation",
	Long:  "Resolve a review conversation using its first comment's ID. Requires the resolve-comment API (Gitea 1.26 or a compatible server).",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRResolve,
}

var prUnresolveCmd = &cobra.Command{
	Use:   "unresolve [<owner>/<repo>] <comment-id>",
	Short: "Reopen a pull request review conversation",
	Long:  "Reopen a review conversation using its first comment's ID. Requires the unresolve-comment API (Gitea 1.26 or a compatible server).",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRUnresolve,
}

func init() {
	prCmd.AddCommand(prResolveCmd, prUnresolveCmd)
}

func runPRResolve(cmd *cobra.Command, args []string) error {
	return setPRConversationResolved(cmd, args, true)
}

func runPRUnresolve(cmd *cobra.Command, args []string) error {
	return setPRConversationResolved(cmd, args, false)
}

func setPRConversationResolved(cmd *cobra.Command, args []string, resolved bool) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	if commentID <= 0 {
		return errors.NewValidationError("comment id must be positive", nil)
	}
	if _, err := getServer(cmd); err != nil {
		return err
	}
	action, status := "resolve", "resolved"
	if !resolved {
		action, status = "unresolve", "unresolved"
	}
	if dryRunf(cmd, "Would %s review conversation #%d in %s/%s", action, commentID, owner, repo) {
		return nil
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/comments/%d/%s", owner, repo, commentID, action)
	resp, err := apiDo(cmd, http.MethodPost, path, nil, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to read response: %v", err))
		}
		if resp.StatusCode >= http.StatusBadRequest {
			apiErr := apiError(resp.StatusCode, data)
			if resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusMethodNotAllowed {
				apiErr.Message += "; verify the comment ID and server support for the resolve-comment API (Gitea 1.26 or compatible); older servers require the web UI"
			}
			return apiErr
		}
		return errors.NewGeneralError(fmt.Sprintf("server did not confirm conversation %s: expected HTTP 204, got %d", action, resp.StatusCode))
	}
	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": status, "comment": commentID, "repository": owner + "/" + repo}, func() error {
		printer.Printf("Conversation #%d %s\n", commentID, status)
		return nil
	})
}
