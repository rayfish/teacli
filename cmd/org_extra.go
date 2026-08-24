// Package cmd contains the organization commands beyond create/members, plus
// organization labels and activity feeds.
package cmd

import (
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var orgUpdateCmd = &cobra.Command{
	Use:     "update <org-name>",
	Aliases: []string{"edit"},
	Short:   "Edit an organization",
	Args:    cobra.ExactArgs(1),
	RunE:    runOrgUpdate,
}

var orgRenameCmd = &cobra.Command{
	Use:   "rename <org-name> <new-name>",
	Short: "Rename an organization",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgRename,
}

var orgPermissionCmd = &cobra.Command{
	Use:     "permission <org-name> <username>",
	Aliases: []string{"perm"},
	Short:   "Show a user's permissions in an organization",
	Args:    cobra.ExactArgs(2),
	RunE:    runOrgPermission,
}

var orgReposCmd = &cobra.Command{
	Use:   "repos <org-name>",
	Short: "List an organization's repositories",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgRepos,
}

var orgPublicMembersCmd = &cobra.Command{
	Use:   "public-members <org-name>",
	Short: "List an organization's public members",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgPublicMembers,
}

var orgPublicizeCmd = &cobra.Command{
	Use:   "publicize <org-name> <username>",
	Short: "Make a membership public",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgPublicize,
}

var orgConcealCmd = &cobra.Command{
	Use:   "conceal <org-name> <username>",
	Short: "Make a membership private",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgConceal,
}

var orgLabelCmd = &cobra.Command{
	Use:     "label",
	Aliases: []string{"labels"},
	Short:   "Manage organization-wide labels",
}

var orgLabelListCmd = &cobra.Command{
	Use:   "list <org-name>",
	Short: "List an organization's labels",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgLabelList,
}

var orgLabelGetCmd = &cobra.Command{
	Use:   "get <org-name> <label-id>",
	Short: "Get one organization label",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgLabelGet,
}

var orgLabelCreateCmd = &cobra.Command{
	Use:   "create <org-name>",
	Short: "Create an organization label",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgLabelCreate,
}

var orgLabelUpdateCmd = &cobra.Command{
	Use:     "update <org-name> <label-id>",
	Aliases: []string{"edit"},
	Short:   "Edit an organization label",
	Args:    cobra.ExactArgs(2),
	RunE:    runOrgLabelUpdate,
}

var orgLabelDeleteCmd = &cobra.Command{
	Use:     "delete <org-name> <label-id>",
	Aliases: []string{"rm"},
	Short:   "Delete an organization label",
	Args:    cobra.ExactArgs(2),
	RunE:    runOrgLabelDelete,
}

var orgActivityCmd = &cobra.Command{
	Use:     "activity <org-name>",
	Aliases: []string{"feed"},
	Short:   "Show an organization's activity feed",
	Args:    cobra.ExactArgs(1),
	RunE:    runOrgActivity,
}

func init() {
	orgCmd.AddCommand(orgUpdateCmd, orgRenameCmd, orgPermissionCmd, orgReposCmd,
		orgPublicMembersCmd, orgPublicizeCmd, orgConcealCmd, orgLabelCmd, orgActivityCmd)
	orgLabelCmd.AddCommand(orgLabelListCmd, orgLabelGetCmd, orgLabelCreateCmd,
		orgLabelUpdateCmd, orgLabelDeleteCmd)

	addPageFlags(orgReposCmd, orgPublicMembersCmd, orgLabelListCmd, orgActivityCmd)

	orgUpdateCmd.Flags().StringP("full-name", "f", "", "New full name")
	orgUpdateCmd.Flags().StringP("description", "d", "", "New description")
	orgUpdateCmd.Flags().StringP("website", "w", "", "New website")
	orgUpdateCmd.Flags().StringP("location", "l", "", "New location")
	orgUpdateCmd.Flags().StringP("email", "e", "", "New contact email")
	orgUpdateCmd.Flags().StringP("visibility", "v", "", "Visibility: public, limited, or private")
	orgUpdateCmd.Flags().Bool("repo-admin-change-team-access", false,
		"Let repository admins change team access")
	orgUpdateCmd.Flags().Bool("no-repo-admin-change-team-access", false,
		"Stop repository admins changing team access")

	for _, c := range []*cobra.Command{orgLabelCreateCmd, orgLabelUpdateCmd} {
		c.Flags().StringP("name", "n", "", "Label name")
		c.Flags().StringP("color", "c", "", "Label colour, as a hex value such as #ff0000")
		c.Flags().StringP("description", "d", "", "Label description")
		c.Flags().Bool("exclusive", false, "Make the label exclusive within its scope")
	}
}

func runOrgUpdate(cmd *cobra.Command, args []string) error {
	org := args[0]

	visibility, _ := cmd.Flags().GetString("visibility")
	if visibility != "" && visibility != "public" && visibility != "limited" && visibility != "private" {
		return errors.NewValidationError(fmt.Sprintf("invalid visibility: %s", visibility),
			map[string]interface{}{"expected": "public, limited, or private"})
	}

	fullName, _ := cmd.Flags().GetString("full-name")
	description, _ := cmd.Flags().GetString("description")
	website, _ := cmd.Flags().GetString("website")
	location, _ := cmd.Flags().GetString("location")
	email, _ := cmd.Flags().GetString("email")

	opts := gitea.EditOrgOption{
		FullName:    fullName,
		Description: description,
		Website:     website,
		Location:    location,
		Email:       email,
		Visibility:  gitea.VisibleType(visibility),
		RepoAdminChangeTeamAccess: boolPairFlag(cmd,
			"repo-admin-change-team-access", "no-repo-admin-change-team-access"),
	}

	if fullName == "" && description == "" && website == "" && location == "" &&
		email == "" && visibility == "" && opts.RepoAdminChangeTeamAccess == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --description"})
	}

	if dryRunf(cmd, "Would edit organization %s", org) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.EditOrg(org, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization", org)
	}

	return emitMessage(cmd, okMessage("organization updated"), "Updated organization %s", org)
}

func runOrgRename(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would rename organization %s to %s", args[0], args[1]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.RenameOrg(args[0], gitea.RenameOrgOption{NewName: args[1]})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization", args[0])
	}
	return emitMessage(cmd, okMessage("organization renamed"), "Renamed %s to %s", args[0], args[1])
}

func runOrgPermission(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	perm, resp, err := client.GetOrgPermissions(args[0], args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization membership", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(perm, func() error {
		var d detail
		d.always("Is owner", perm.IsOwner)
		d.always("Is admin", perm.IsAdmin)
		d.always("Can write", perm.CanWrite)
		d.always("Can read", perm.CanRead)
		d.always("Can create repository", perm.CanCreateRepository)
		return d.print(printer)
	})
}

func runOrgRepos(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repos, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Repository, *gitea.Response, error) {
		return client.ListOrgRepos(args[0], gitea.ListOrgReposOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitRepoTable(cmd, repos)
}

func runOrgPublicMembers(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListPublicOrgMembership(args[0], gitea.ListOrgMembershipOption{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No public members.")
}

func runOrgPublicize(cmd *cobra.Command, args []string) error {
	return setOrgMembershipVisibility(cmd, args, true)
}

func runOrgConceal(cmd *cobra.Command, args []string) error {
	return setOrgMembershipVisibility(cmd, args, false)
}

func setOrgMembershipVisibility(cmd *cobra.Command, args []string, public bool) error {
	verb := "conceal"
	if public {
		verb = "publicize"
	}
	if dryRunf(cmd, "Would %s %s's membership of %s", verb, args[1], args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.SetPublicOrgMembership(args[0], args[1], public)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization membership", args[1])
	}

	state := "private"
	if public {
		state = "public"
	}
	return emitMessage(cmd, okMessage("membership "+state),
		"%s's membership of %s is now %s", args[1], args[0], state)
}

func emitLabels(cmd *cobra.Command, labels []*gitea.Label) error {
	printer := getPrinter(cmd)
	return printer.Emit(labels, func() error {
		if len(labels) == 0 {
			printer.Println("No labels found.")
			return nil
		}
		rows := make([][]string, 0, len(labels))
		for _, l := range labels {
			rows = append(rows, []string{
				fmt.Sprintf("%d", l.ID), l.Name, l.Color, cell(l.Description, 50),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Color", "Description"}, rows)
	})
}

func runOrgLabelList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	labels, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Label, *gitea.Response, error) {
		return client.ListOrgLabels(args[0], gitea.ListOrgLabelsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitLabels(cmd, labels)
}

func runOrgLabelGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[1], "label id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	label, resp, err := client.GetOrgLabel(args[0], id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(label, func() error {
		var d detail
		d.always("ID", label.ID)
		d.add("Name", label.Name)
		d.add("Color", label.Color)
		d.add("Description", label.Description)
		return d.print(printer)
	})
}

func runOrgLabelCreate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	color, _ := cmd.Flags().GetString("color")
	description, _ := cmd.Flags().GetString("description")
	exclusive, _ := cmd.Flags().GetBool("exclusive")

	if name == "" || color == "" {
		return errors.NewValidationError("--name and --color are required", nil)
	}

	if dryRunf(cmd, "Would create label %q in %s", name, args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	label, resp, err := client.CreateOrgLabel(args[0], gitea.CreateOrgLabelOption{
		Name: name, Color: color, Description: description, Exclusive: exclusive,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, label, "Created label %s (id %d)", label.Name, label.ID)
}

func runOrgLabelUpdate(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[1], "label id")
	if err != nil {
		return err
	}

	opts := gitea.EditOrgLabelOption{
		Name:        stringFlagPtr(cmd, "name"),
		Color:       stringFlagPtr(cmd, "color"),
		Description: stringFlagPtr(cmd, "description"),
	}
	if cmd.Flags().Changed("exclusive") {
		exclusive, _ := cmd.Flags().GetBool("exclusive")
		opts.Exclusive = &exclusive
	}
	if opts.Name == nil && opts.Color == nil && opts.Description == nil && opts.Exclusive == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass --name, --color, --description or --exclusive"})
	}

	if dryRunf(cmd, "Would edit label %d in %s", id, args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	label, resp, err := client.EditOrgLabel(args[0], id, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", args[1])
	}

	return emitMessage(cmd, label, "Updated label %s (id %d)", label.Name, label.ID)
}

func runOrgLabelDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[1], "label id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete label %d from %s", id, args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteOrgLabel(args[0], id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "label", args[1])
	}
	return emitMessage(cmd, okMessage("label deleted"), "Deleted label %d from %s", id, args[0])
}

func runOrgActivity(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	activities, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Activity, *gitea.Response, error) {
		return client.ListOrgActivityFeeds(args[0], gitea.ListOrgActivityFeedsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitActivities(cmd, activities)
}

// emitActivities renders an activity feed, shared by the org and user feeds.
func emitActivities(cmd *cobra.Command, activities []*gitea.Activity) error {
	printer := getPrinter(cmd)
	return printer.Emit(activities, func() error {
		if len(activities) == 0 {
			printer.Println("No activity.")
			return nil
		}
		rows := make([][]string, 0, len(activities))
		for _, a := range activities {
			repo := ""
			if a.Repo != nil {
				repo = a.Repo.FullName
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", a.ID), string(a.OpType), userName(a.ActUser),
				repo, renderValue(a.Created), cell(a.Content, 40),
			})
		}
		return printer.PrintTable([]string{"ID", "Type", "User", "Repo", "When", "Detail"}, rows)
	})
}
