package cmd

import (
	"fmt"
	"net/http"
	"net/url"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

var prPinnedCmd = &cobra.Command{
	Use:   "pinned [<owner>/<repo>]",
	Short: "List pinned pull requests",
	Args:  cobra.RangeArgs(0, 1),
	RunE:  runPRPinned,
}
var prFindCmd = &cobra.Command{
	Use:   "find [<owner>/<repo>] <base> <head>",
	Short: "Find a pull request by base and head branches",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runPRFind,
}
var prUpdateBranchCmd = &cobra.Command{
	Use:   "update-branch [<owner>/<repo>] <pr-number>",
	Short: "Update the head branch from the base branch",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRUpdateBranch,
}
var prCancelAutoMergeCmd = &cobra.Command{
	Use:   "cancel-auto-merge [<owner>/<repo>] <pr-number>",
	Short: "Cancel a scheduled automatic merge",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runPRCancelAutoMerge,
}

func init() {
	prCmd.AddCommand(prPinnedCmd, prFindCmd, prUpdateBranchCmd, prCancelAutoMergeCmd)
	prUpdateBranchCmd.Flags().String("style", "merge", "Update style: merge or rebase")
}

func pullRequestPath(owner, repo string, index int64) string {
	return fmt.Sprintf("repos/%s/%s/pulls/%d", url.PathEscape(owner), url.PathEscape(repo), index)
}

func runPRPinned(cmd *cobra.Command, args []string) error {
	owner, repo, _, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	var pulls []*gitea.PullRequest
	path := fmt.Sprintf("repos/%s/%s/pulls/pinned", url.PathEscape(owner), url.PathEscape(repo))
	if err := apiGetInto(cmd, path, &pulls); err != nil {
		return err
	}
	printer := getPrinter(cmd)
	return printer.Emit(pulls, func() error {
		rows := make([][]string, 0, len(pulls))
		for _, pr := range pulls {
			if pr != nil {
				rows = append(rows, []string{fmt.Sprint(pr.Index), pr.Title, string(pr.State), pr.HTMLURL})
			}
		}
		return printer.PrintTable([]string{"NUMBER", "TITLE", "STATE", "URL"}, rows)
	})
}

func runPRFind(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	path := fmt.Sprintf("repos/%s/%s/pulls/%s/%s", url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(args[1]), url.PathEscape(args[2]))
	var pr gitea.PullRequest
	if err := apiGetInto(cmd, path, &pr); err != nil {
		return err
	}
	printer := getPrinter(cmd)
	return printer.Emit(pr, func() error {
		var d detail
		d.always("Number", pr.Index)
		d.add("Title", pr.Title)
		d.add("State", string(pr.State))
		d.add("URL", pr.HTMLURL)
		return d.print(printer)
	})
}

func runPRUpdateBranch(cmd *cobra.Command, args []string) error {
	owner, repo, index, _, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	style, _ := cmd.Flags().GetString("style")
	if style != "merge" && style != "rebase" {
		return errors.NewValidationError("update style must be merge or rebase", nil)
	}
	if _, err := getServer(cmd); err != nil {
		return err
	}
	if dryRunf(cmd, "Would update head branch of %s/%s#%d using %s", owner, repo, index, style) {
		return nil
	}
	if _, err := apiRequest(cmd, http.MethodPost, pullRequestPath(owner, repo, index)+"/update?style="+style, nil, http.StatusOK); err != nil {
		return err
	}
	return emitMessage(cmd, okMessage("pull request branch updated"), "Updated branch for PR #%d", index)
}

func runPRCancelAutoMerge(cmd *cobra.Command, args []string) error {
	owner, repo, index, _, err := repoAndIndex(cmd, args, 1)
	if err != nil {
		return err
	}
	if _, err := getServer(cmd); err != nil {
		return err
	}
	if dryRunf(cmd, "Would cancel automatic merge for %s/%s#%d", owner, repo, index) {
		return nil
	}
	if _, err := apiRequest(cmd, http.MethodDelete, pullRequestPath(owner, repo, index)+"/merge", nil, http.StatusNoContent); err != nil {
		return err
	}
	return emitMessage(cmd, okMessage("automatic merge cancelled"), "Cancelled automatic merge for PR #%d", index)
}
