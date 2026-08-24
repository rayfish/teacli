// Package cmd contains repository collaborator and access commands.
package cmd

import (
	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var collaboratorCmd = &cobra.Command{
	Use:     "collaborator",
	Aliases: []string{"collaborators", "collab"},
	Short:   "Manage repository collaborators",
}

var collaboratorListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List the collaborators on a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCollaboratorList,
}

var collaboratorAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <username>...",
	Short: "Add collaborators to a repository",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runCollaboratorAdd,
}

var collaboratorRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <username>...",
	Aliases: []string{"rm"},
	Short:   "Remove collaborators from a repository",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runCollaboratorRemove,
}

var collaboratorCheckCmd = &cobra.Command{
	Use:   "check [<owner>/<repo>] <username>",
	Short: "Report whether a user is a collaborator (exit 3 if not)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runCollaboratorCheck,
}

var collaboratorPermissionCmd = &cobra.Command{
	Use:     "permission [<owner>/<repo>] <username>",
	Aliases: []string{"perm"},
	Short:   "Show a user's permission level on a repository",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runCollaboratorPermission,
}

var reviewerListCmd = &cobra.Command{
	Use:   "reviewers [<owner>/<repo>]",
	Short: "List the users who can review pull requests on a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReviewerList,
}

var assigneeListCmd = &cobra.Command{
	Use:   "assignees [<owner>/<repo>]",
	Short: "List the users who can be assigned issues on a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAssigneeList,
}

func init() {
	RootCmd.AddCommand(collaboratorCmd)
	collaboratorCmd.AddCommand(collaboratorListCmd, collaboratorAddCmd, collaboratorRemoveCmd,
		collaboratorCheckCmd, collaboratorPermissionCmd)
	repoCmd.AddCommand(reviewerListCmd, assigneeListCmd)

	addPageFlags(collaboratorListCmd)
	collaboratorAddCmd.Flags().StringP("permission", "p", "write",
		"Access level to grant: read, write, or admin")
}

func runCollaboratorList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListCollaborators(owner, repo, gitea.ListCollaboratorsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No collaborators found.")
}

func runCollaboratorAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	users := args[1:]

	permission, _ := cmd.Flags().GetString("permission")
	mode, err := parseAccessMode(permission)
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would add %d collaborator(s) to %s/%s with %s access", len(users), owner, repo, mode) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.AddCollaboratorOption{Permission: &mode}
	for _, u := range users {
		resp, err := client.AddCollaborator(owner, repo, u, opts)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "user", u)
		}
	}

	return emitMessage(cmd, okMessage("collaborators added"),
		"Added %d collaborator(s) to %s/%s", len(users), owner, repo)
}

func runCollaboratorRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	users := args[1:]

	if dryRunf(cmd, "Would remove %d collaborator(s) from %s/%s", len(users), owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	for _, u := range users {
		resp, err := client.DeleteCollaborator(owner, repo, u)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "collaborator", u)
		}
	}

	return emitMessage(cmd, okMessage("collaborators removed"),
		"Removed %d collaborator(s) from %s/%s", len(users), owner, repo)
}

func runCollaboratorCheck(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	isCollaborator, resp, err := client.IsCollaborator(owner, repo, username)
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	// A "no" is reported as exit 3 so a script can branch on the exit code
	// rather than parsing the message.
	if !isCollaborator {
		return errors.NewNotFoundError("collaborator", username)
	}

	return emitMessage(cmd, map[string]any{"collaborator": true, "user": username},
		"%s is a collaborator on %s/%s", username, owner, repo)
}

func runCollaboratorPermission(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	perm, resp, err := client.CollaboratorPermission(owner, repo, username)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(perm, func() error {
		var d detail
		d.add("User", userName(perm.User))
		d.add("Permission", string(perm.Permission))
		d.add("Role", perm.Role)
		return d.print(printer)
	})
}

func runReviewerList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, resp, err := client.GetReviewers(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitUserTable(cmd, users, "No reviewers available.")
}

func runAssigneeList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, resp, err := client.GetAssignees(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitUserTable(cmd, users, "No assignees available.")
}
