// Package cmd contains the remaining repository and account features: push
// mirrors, server-side git hooks, avatars, archives, repository teams, issue and
// comment attachments, and OAuth2 applications.
package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var mirrorCmd = &cobra.Command{
	Use:     "mirror",
	Aliases: []string{"mirrors", "push-mirror"},
	Short:   "Manage a repository's push mirrors",
}

var mirrorListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List a repository's push mirrors",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runMirrorList,
}

var mirrorGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <remote-name>",
	Short: "Get one push mirror",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMirrorGet,
}

var mirrorCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <remote-url>",
	Short: "Add a push mirror to a repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runMirrorCreate,
}

var mirrorDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <remote-name>",
	Aliases: []string{"rm"},
	Short:   "Delete a push mirror",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runMirrorDelete,
}

var gitHookCmd = &cobra.Command{
	Use:     "hook",
	Aliases: []string{"hooks"},
	Short:   "Manage a repository's server-side git hooks (admin only)",
}

var gitHookListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List a repository's git hooks",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGitHookList,
}

var gitHookGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <hook-name>",
	Short: "Print a git hook's script",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runGitHookGet,
}

var gitHookSetCmd = &cobra.Command{
	Use:     "set [<owner>/<repo>] <hook-name>",
	Aliases: []string{"update", "edit"},
	Short:   "Replace a git hook's script",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runGitHookSet,
}

var gitHookDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <hook-name>",
	Aliases: []string{"rm"},
	Short:   "Clear a git hook's script",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runGitHookDelete,
}

var repoArchiveCmd = &cobra.Command{
	Use:   "archive [<owner>/<repo>] <ref>",
	Short: "Download a repository archive at a ref",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoArchive,
}

var repoAvatarCmd = &cobra.Command{
	Use:   "avatar [<owner>/<repo>]",
	Short: "Set or clear a repository's avatar",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoAvatar,
}

var orgAvatarCmd = &cobra.Command{
	Use:   "avatar <org-name>",
	Short: "Set or clear an organization's avatar",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgAvatar,
}

var userAvatarCmd = &cobra.Command{
	Use:   "avatar",
	Short: "Set or clear your account avatar",
	Args:  cobra.NoArgs,
	RunE:  runUserAvatar,
}

var repoTeamCmd = &cobra.Command{
	Use:     "team",
	Aliases: []string{"teams"},
	Short:   "Manage the teams that can reach a repository",
}

var repoTeamListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List the teams with access to a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoTeamList,
}

var repoTeamAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <team>",
	Short: "Give a team access to a repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoTeamAdd,
}

var repoTeamRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <team>",
	Aliases: []string{"rm"},
	Short:   "Remove a team's access to a repository",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runRepoTeamRemove,
}

var repoTeamCheckCmd = &cobra.Command{
	Use:   "check [<owner>/<repo>] <team>",
	Short: "Report whether a team can reach a repository (exit 3 if not)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoTeamCheck,
}

var attachmentCmd = &cobra.Command{
	Use:     "attachment",
	Aliases: []string{"attachments"},
	Short:   "Manage the files attached to issue comments",
}

var attachmentListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <comment-id>",
	Short: "List a comment's attachments",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runAttachmentList,
}

var attachmentGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <comment-id> <attachment-id>",
	Short: "Get one comment attachment",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runAttachmentGet,
}

var attachmentRenameCmd = &cobra.Command{
	Use:     "rename [<owner>/<repo>] <comment-id> <attachment-id> <new-name>",
	Aliases: []string{"update", "edit"},
	Short:   "Rename a comment attachment",
	Args:    cobra.RangeArgs(3, 4),
	RunE:    runAttachmentRename,
}

var attachmentDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <comment-id> <attachment-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a comment attachment",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runAttachmentDelete,
}

var issueTemplateCmd = &cobra.Command{
	Use:     "templates [<owner>/<repo>]",
	Aliases: []string{"template"},
	Short:   "List a repository's issue templates",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runIssueTemplates,
}

var oauthCmd = &cobra.Command{
	Use:     "oauth",
	Aliases: []string{"oauth2", "app", "apps"},
	Short:   "Manage your OAuth2 applications",
}

var oauthListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your OAuth2 applications",
	Args:  cobra.NoArgs,
	RunE:  runOAuthList,
}

var oauthGetCmd = &cobra.Command{
	Use:   "get <app-id>",
	Short: "Get one OAuth2 application",
	Args:  cobra.ExactArgs(1),
	RunE:  runOAuthGet,
}

var oauthCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an OAuth2 application",
	Args:  cobra.ExactArgs(1),
	RunE:  runOAuthCreate,
}

var oauthUpdateCmd = &cobra.Command{
	Use:     "update <app-id> <name>",
	Aliases: []string{"edit"},
	Short:   "Edit an OAuth2 application",
	Args:    cobra.ExactArgs(2),
	RunE:    runOAuthUpdate,
}

var oauthDeleteCmd = &cobra.Command{
	Use:     "delete <app-id>",
	Aliases: []string{"rm"},
	Short:   "Delete an OAuth2 application",
	Args:    cobra.ExactArgs(1),
	RunE:    runOAuthDelete,
}

func init() {
	RootCmd.AddCommand(attachmentCmd, oauthCmd)
	repoCmd.AddCommand(mirrorCmd, gitHookCmd, repoArchiveCmd, repoAvatarCmd, repoTeamCmd)
	orgCmd.AddCommand(orgAvatarCmd)
	userCmd.AddCommand(userAvatarCmd)
	issueCmd.AddCommand(issueTemplateCmd)

	mirrorCmd.AddCommand(mirrorListCmd, mirrorGetCmd, mirrorCreateCmd, mirrorDeleteCmd)
	gitHookCmd.AddCommand(gitHookListCmd, gitHookGetCmd, gitHookSetCmd, gitHookDeleteCmd)
	repoTeamCmd.AddCommand(repoTeamListCmd, repoTeamAddCmd, repoTeamRemoveCmd, repoTeamCheckCmd)
	attachmentCmd.AddCommand(attachmentListCmd, attachmentGetCmd, attachmentRenameCmd, attachmentDeleteCmd)
	oauthCmd.AddCommand(oauthListCmd, oauthGetCmd, oauthCreateCmd, oauthUpdateCmd, oauthDeleteCmd)

	addPageFlags(mirrorListCmd, gitHookListCmd, oauthListCmd)

	mirrorCreateCmd.Flags().StringP("interval", "i", "8h0m0s", "Sync interval, e.g. 8h0m0s")
	mirrorCreateCmd.Flags().StringP("username", "u", "", "Username for the remote")
	mirrorCreateCmd.Flags().StringP("password", "p", "", "Password or token for the remote")
	mirrorCreateCmd.Flags().Bool("sync-on-commit", false, "Push as soon as a commit lands")

	gitHookSetCmd.Flags().StringP("content", "c", "", "Hook script")
	gitHookSetCmd.Flags().StringP("file", "f", "", "Read the hook script from this file, or - for stdin")

	repoArchiveCmd.Flags().StringP("format", "F", "zip", "Archive format: zip or tar.gz")
	repoArchiveCmd.Flags().StringP("out", "o", "", "Write to this path (default: <repo>-<ref>.<ext>)")

	for _, c := range []*cobra.Command{repoAvatarCmd, orgAvatarCmd, userAvatarCmd} {
		c.Flags().StringP("image", "i", "", "Path to the new avatar image")
		c.Flags().Bool("delete", false, "Remove the current avatar instead of setting one")
	}

	oauthCreateCmd.Flags().StringSliceP("redirect-uri", "r", nil,
		"Redirect URI, repeatable (required)")
	oauthCreateCmd.MarkFlagRequired("redirect-uri")
	oauthCreateCmd.Flags().Bool("confidential", true, "The client can keep a secret")
	oauthUpdateCmd.Flags().StringSliceP("redirect-uri", "r", nil, "Redirect URI, repeatable")
	oauthUpdateCmd.Flags().Bool("confidential", true, "The client can keep a secret")
}

func runMirrorList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	mirrors, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.PushMirrorResponse, *gitea.Response, error) {
		return client.ListPushMirrors(owner, repo, lo)
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(mirrors, func() error {
		if len(mirrors) == 0 {
			printer.Println("No push mirrors.")
			return nil
		}
		rows := make([][]string, 0, len(mirrors))
		for _, m := range mirrors {
			rows = append(rows, []string{
				m.RemoteName, m.RemoteAddress, m.Interval, m.LastUpdate, cell(m.LastError, 30),
			})
		}
		return printer.PrintTable([]string{"Name", "Remote", "Interval", "Last update", "Last error"}, rows)
	})
}

func runMirrorGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	mirror, resp, err := client.GetPushMirrorByRemoteName(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "push mirror", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(mirror, func() error {
		var d detail
		d.add("Name", mirror.RemoteName)
		d.add("Remote", mirror.RemoteAddress)
		d.add("Repository", mirror.RepoName)
		d.add("Interval", mirror.Interval)
		d.always("Sync on commit", mirror.SyncONCommit)
		d.add("Created", mirror.Created)
		d.add("Last update", mirror.LastUpdate)
		d.add("Last error", mirror.LastError)
		return d.print(printer)
	})
}

func runMirrorCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	remote := args[1]

	interval, _ := cmd.Flags().GetString("interval")
	username, _ := cmd.Flags().GetString("username")
	password, _ := cmd.Flags().GetString("password")
	syncOnCommit, _ := cmd.Flags().GetBool("sync-on-commit")

	if dryRunf(cmd, "Would push-mirror %s/%s to %s", owner, repo, remote) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	mirror, resp, err := client.PushMirrors(owner, repo, gitea.CreatePushMirrorOption{
		RemoteAddress:  remote,
		Interval:       interval,
		RemoteUsername: username,
		RemotePassword: password,
		SyncONCommit:   syncOnCommit,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, mirror, "Added push mirror %s to %s/%s", mirror.RemoteName, owner, repo)
}

func runMirrorDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete push mirror %s from %s/%s", args[1], owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeletePushMirror(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "push mirror", args[1])
	}
	return emitMessage(cmd, okMessage("mirror deleted"), "Deleted push mirror %s", args[1])
}

func runGitHookList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hooks, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.GitHook, *gitea.Response, error) {
		return client.ListRepoGitHooks(owner, repo, gitea.ListRepoGitHooksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(hooks, func() error {
		if len(hooks) == 0 {
			printer.Println("No git hooks.")
			return nil
		}
		rows := make([][]string, 0, len(hooks))
		for _, h := range hooks {
			rows = append(rows, []string{h.Name, yesNo(h.IsActive)})
		}
		return printer.PrintTable([]string{"Name", "Active"}, rows)
	})
}

func runGitHookGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hook, resp, err := client.GetRepoGitHook(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "git hook", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(hook, func() error {
		printer.Println(strings.TrimRight(hook.Content, "\n"))
		return nil
	})
}

func runGitHookSet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	content, err := bodyFromFlags(cmd, "content", "file")
	if err != nil {
		return err
	}
	if content == "" {
		return errors.NewValidationError("hook script required",
			map[string]interface{}{"hint": "pass --content <script> or --file <path>"})
	}

	if dryRunf(cmd, "Would set the %s hook on %s/%s", args[1], owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.EditRepoGitHook(owner, repo, args[1], gitea.EditGitHookOption{Content: content})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "git hook", args[1])
	}

	return emitMessage(cmd, okMessage("git hook set"), "Set the %s hook on %s/%s", args[1], owner, repo)
}

func runGitHookDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would clear the %s hook on %s/%s", args[1], owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteRepoGitHook(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "git hook", args[1])
	}
	return emitMessage(cmd, okMessage("git hook cleared"), "Cleared the %s hook", args[1])
}

func runRepoArchive(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	ref := args[1]

	format, _ := cmd.Flags().GetString("format")
	var ext gitea.ArchiveType
	switch strings.TrimPrefix(format, ".") {
	case "zip":
		ext = gitea.ZipArchive
	case "tar.gz", "targz", "tgz":
		ext = gitea.TarGZArchive
	default:
		return errors.NewValidationError(fmt.Sprintf("invalid format: %s", format),
			map[string]interface{}{"expected": "zip or tar.gz"})
	}

	out, _ := cmd.Flags().GetString("out")
	if out == "" {
		out = fmt.Sprintf("%s-%s%s", repo, strings.ReplaceAll(ref, "/", "-"), ext)
	}

	if dryRunf(cmd, "Would write the %s archive of %s/%s@%s to %s", format, owner, repo, ref, out) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	data, resp, err := client.GetArchive(owner, repo, ref, ext)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "ref", ref)
	}

	if err := os.WriteFile(out, data, 0o644); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to write %s: %v", out, err))
	}

	return emitMessage(cmd, map[string]any{"path": out, "bytes": len(data)},
		"Wrote %s (%d bytes)", out, len(data))
}

// avatarImage reads --image and returns it base64-encoded, the form the avatar
// endpoints take.
func avatarImage(cmd *cobra.Command) (string, error) {
	path, _ := cmd.Flags().GetString("image")
	if path == "" {
		return "", errors.NewValidationError("an image is required",
			map[string]interface{}{"hint": "pass --image <path>, or --delete to clear the avatar"})
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.NewValidationError(fmt.Sprintf("failed to read %s: %v", path, err), nil)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func runRepoAvatar(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if remove, _ := cmd.Flags().GetBool("delete"); remove {
		if dryRunf(cmd, "Would clear the avatar of %s/%s", owner, repo) {
			return nil
		}
		resp, err := client.DeleteRepoAvatar(owner, repo)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "repository", args[0])
		}
		return emitMessage(cmd, okMessage("avatar cleared"), "Cleared the avatar of %s/%s", owner, repo)
	}

	image, err := avatarImage(cmd)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would set the avatar of %s/%s", owner, repo) {
		return nil
	}

	resp, err := client.UpdateRepoAvatar(owner, repo, gitea.UpdateRepoAvatarOption{Image: image})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("avatar set"), "Set the avatar of %s/%s", owner, repo)
}

func runOrgAvatar(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if remove, _ := cmd.Flags().GetBool("delete"); remove {
		if dryRunf(cmd, "Would clear the avatar of %s", args[0]) {
			return nil
		}
		resp, err := client.DeleteOrgAvatar(args[0])
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "organization", args[0])
		}
		return emitMessage(cmd, okMessage("avatar cleared"), "Cleared the avatar of %s", args[0])
	}

	image, err := avatarImage(cmd)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would set the avatar of %s", args[0]) {
		return nil
	}

	resp, err := client.UpdateOrgAvatar(args[0], gitea.UpdateUserAvatarOption{Image: image})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization", args[0])
	}
	return emitMessage(cmd, okMessage("avatar set"), "Set the avatar of %s", args[0])
}

func runUserAvatar(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if remove, _ := cmd.Flags().GetBool("delete"); remove {
		if dryRunf(cmd, "Would clear your avatar") {
			return nil
		}
		resp, err := client.DeleteUserAvatar()
		if err != nil {
			return errors.FromGitea(resp, err)
		}
		return emitMessage(cmd, okMessage("avatar cleared"), "Cleared your avatar")
	}

	image, err := avatarImage(cmd)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would set your avatar") {
		return nil
	}

	resp, err := client.UpdateUserAvatar(gitea.UpdateUserAvatarOption{Image: image})
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return emitMessage(cmd, okMessage("avatar set"), "Set your avatar")
}

func runRepoTeamList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	teams, resp, err := client.GetRepoTeams(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitTeams(cmd, teams)
}

func runRepoTeamAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would give team %s access to %s/%s", args[1], owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.AddRepoTeam(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team", args[1])
	}
	return emitMessage(cmd, okMessage("team added"), "Gave team %s access to %s/%s", args[1], owner, repo)
}

func runRepoTeamRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would remove team %s's access to %s/%s", args[1], owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.RemoveRepoTeam(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team", args[1])
	}
	return emitMessage(cmd, okMessage("team removed"),
		"Removed team %s's access to %s/%s", args[1], owner, repo)
}

func runRepoTeamCheck(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hasAccess, resp, err := client.CheckRepoTeam(owner, repo, args[1])
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	if hasAccess == nil {
		return errors.NewNotFoundError("team", args[1])
	}

	return emitMessage(cmd, hasAccess, "Team %s can reach %s/%s", args[1], owner, repo)
}

func runAttachmentList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	assets, resp, err := client.ListIssueCommentAttachments(owner, repo, commentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comment", args[1])
	}

	return emitAssets(cmd, assets)
}

func runAttachmentGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	attachmentID, err := int64Arg(args[2], "attachment id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	asset, resp, err := client.GetIssueCommentAttachment(owner, repo, commentID, attachmentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "attachment", args[2])
	}

	printer := getPrinter(cmd)
	return printer.Emit(asset, func() error {
		var d detail
		d.always("ID", asset.ID)
		d.add("Name", asset.Name)
		d.always("Size", asset.Size)
		d.always("Downloads", asset.DownloadCount)
		d.add("Created", asset.Created)
		d.add("URL", asset.DownloadURL)
		return d.print(printer)
	})
}

func runAttachmentRename(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 3)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	attachmentID, err := int64Arg(args[2], "attachment id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would rename attachment %d to %s", attachmentID, args[3]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	asset, resp, err := client.EditIssueCommentAttachment(owner, repo, commentID, attachmentID,
		gitea.EditAttachmentOptions{Name: args[3]})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "attachment", args[2])
	}

	return emitMessage(cmd, asset, "Renamed attachment %d to %s", attachmentID, asset.Name)
}

func runAttachmentDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	attachmentID, err := int64Arg(args[2], "attachment id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete attachment %d", attachmentID) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteIssueCommentAttachment(owner, repo, commentID, attachmentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "attachment", args[2])
	}

	return emitMessage(cmd, okMessage("attachment deleted"), "Deleted attachment %d", attachmentID)
}

func runIssueTemplates(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	templates, resp, err := client.GetIssueTemplates(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(templates, func() error {
		if len(templates) == 0 {
			printer.Println("No issue templates.")
			return nil
		}
		rows := make([][]string, 0, len(templates))
		for _, t := range templates {
			rows = append(rows, []string{
				t.Filename, t.Name, cell(t.About, 40), renderValue(t.IssueLabels),
			})
		}
		return printer.PrintTable([]string{"File", "Name", "About", "Labels"}, rows)
	})
}

func runOAuthList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	apps, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Oauth2, *gitea.Response, error) {
		return client.ListOauth2(gitea.ListOauth2Option{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(apps, func() error {
		if len(apps) == 0 {
			printer.Println("No OAuth2 applications.")
			return nil
		}
		rows := make([][]string, 0, len(apps))
		for _, a := range apps {
			rows = append(rows, []string{
				fmt.Sprintf("%d", a.ID), a.Name, a.ClientID,
				renderValue(a.RedirectURIs), renderValue(a.Created),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Client ID", "Redirect URIs", "Created"}, rows)
	})
}

func printOAuthApp(cmd *cobra.Command, app *gitea.Oauth2) error {
	printer := getPrinter(cmd)
	return printer.Emit(app, func() error {
		var d detail
		d.always("ID", app.ID)
		d.add("Name", app.Name)
		d.add("Client ID", app.ClientID)
		d.add("Client secret", app.ClientSecret)
		d.add("Redirect URIs", app.RedirectURIs)
		d.add("Created", app.Created)
		return d.print(printer)
	})
}

func runOAuthGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "application id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	app, resp, err := client.GetOauth2(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "oauth2 application", args[0])
	}
	return printOAuthApp(cmd, app)
}

func runOAuthCreate(cmd *cobra.Command, args []string) error {
	uris, _ := cmd.Flags().GetStringSlice("redirect-uri")
	confidential, _ := cmd.Flags().GetBool("confidential")

	if dryRunf(cmd, "Would create OAuth2 application %q", args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	app, resp, err := client.CreateOauth2(gitea.CreateOauth2Option{
		Name: args[0], RedirectURIs: uris, ConfidentialClient: confidential,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	// The client secret is returned once, at creation, so print the whole app.
	return printOAuthApp(cmd, app)
}

func runOAuthUpdate(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "application id")
	if err != nil {
		return err
	}
	uris, _ := cmd.Flags().GetStringSlice("redirect-uri")
	confidential, _ := cmd.Flags().GetBool("confidential")

	if dryRunf(cmd, "Would edit OAuth2 application %d", id) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	app, resp, err := client.UpdateOauth2(id, gitea.CreateOauth2Option{
		Name: args[1], RedirectURIs: uris, ConfidentialClient: confidential,
	})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "oauth2 application", args[0])
	}
	return printOAuthApp(cmd, app)
}

func runOAuthDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "application id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete OAuth2 application %d", id) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteOauth2(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "oauth2 application", args[0])
	}
	return emitMessage(cmd, okMessage("application deleted"), "Deleted OAuth2 application %d", id)
}
