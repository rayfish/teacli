// Package cmd contains organization commands
package cmd

import (
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var orgCmd = &cobra.Command{
	Use:     "org",
	Aliases: []string{"orgs"},
	Short:   "Manage organizations",
	Long:    `List, create, and manage Gitea organizations`,
}

var orgListCmd = &cobra.Command{
	Use:   "list",
	Short: "List organizations",
	Args:  cobra.NoArgs,
	RunE:  runOrgList,
}

var orgGetCmd = &cobra.Command{
	Use:   "get <org-name>",
	Short: "Get organization details",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgGet,
}

var orgCreateCmd = &cobra.Command{
	Use:   "create <org-name>",
	Short: "Create a new organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgCreate,
}

var orgMembersCmd = &cobra.Command{
	Use:   "members",
	Short: "Manage organization members",
	Long:  `List, add, or remove members from an organization.`,
}

var orgMembersListCmd = &cobra.Command{
	Use:   "list <org-name>",
	Short: "List organization members",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgMembersList,
}

var orgMembersAddCmd = &cobra.Command{
	Use:   "add <org-name> <username>",
	Short: "Add a member to organization",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgMembersAdd,
}

var orgMembersRemoveCmd = &cobra.Command{
	Use:   "remove <org-name> <username>",
	Short: "Remove a member from organization",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgMembersRemove,
}

var orgMembersCheckCmd = &cobra.Command{
	Use:   "check <org-name> <username>",
	Short: "Check if user is a member",
	Args:  cobra.ExactArgs(2),
	RunE:  runOrgMembersCheck,
}

var orgDeleteCmd = &cobra.Command{
	Use:   "delete <org-name>",
	Short: "Delete an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runOrgDelete,
}

func init() {
	RootCmd.AddCommand(orgCmd)
	orgCmd.AddCommand(orgListCmd, orgGetCmd, orgCreateCmd, orgMembersCmd, orgDeleteCmd)
	orgMembersCmd.AddCommand(orgMembersListCmd, orgMembersAddCmd, orgMembersRemoveCmd, orgMembersCheckCmd)

	orgListCmd.Flags().Bool("public", false, "List every public organization instead of your own")
	orgListCmd.Flags().Int("page", 0, "Page number")
	orgListCmd.Flags().Int("per-page", 0, "Results per page")

	orgCreateCmd.Flags().StringP("full-name", "f", "", "Organization full name")
	orgCreateCmd.Flags().StringP("description", "d", "", "Organization description")
	orgCreateCmd.Flags().StringP("website", "w", "", "Organization website URL")
	orgCreateCmd.Flags().StringP("location", "l", "", "Organization location")
	orgCreateCmd.Flags().StringP("visibility", "v", "public", "Organization visibility: public, internal, private")
	orgCreateCmd.Flags().Bool("repo-admin-change-team-access", false, "Allow repo admins to change team access")

	orgMembersAddCmd.Flags().BoolP("public", "p", false, "Make membership visible (publicize)")

	orgDeleteCmd.Flags().BoolP("yes", "y", false, "Skip confirmation prompt")
}

func runOrgList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	public, _ := cmd.Flags().GetBool("public")

	orgs, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Organization, *gitea.Response, error) {
		if public {
			return client.ListOrgs(gitea.ListOrgsOptions{ListOptions: lo})
		}
		return client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return printer.Emit(orgs, func() error {
		headers := []string{"ID", "Name", "Full Name", "Visibility", "Description"}
		var rows [][]string

		for _, org := range orgs {
			rows = append(rows, []string{
				fmt.Sprintf("%d", org.ID),
				org.Name,
				org.FullName,
				org.Visibility,
				org.Description,
			})
		}

		if len(rows) == 0 {
			printer.Println("No organizations found.")
			return nil
		}

		return printer.PrintTable(headers, rows)
	})
}

func runOrgGet(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	if orgName == "" {
		return errors.NewValidationError("organization name required",
			map[string]interface{}{"usage": "teacli org get <org-name>"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	org, resp, err := client.GetOrg(orgName)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization", orgName)
	}

	printer := getPrinter(cmd)
	return printer.Emit(org, func() error {
		return printOrgText(printer, org)
	})
}

func runOrgCreate(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	if orgName == "" {
		return errors.NewValidationError("organization name required",
			map[string]interface{}{"usage": "teacli org create <org-name>"})
	}

	if err := validateOrgName(orgName); err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	fullName, _ := cmd.Flags().GetString("full-name")
	description, _ := cmd.Flags().GetString("description")
	website, _ := cmd.Flags().GetString("website")
	location, _ := cmd.Flags().GetString("location")
	visibility, _ := cmd.Flags().GetString("visibility")
	repoAdminChangeTeamAccess, _ := cmd.Flags().GetBool("repo-admin-change-team-access")

	opts := gitea.CreateOrgOption{
		Name:                      orgName,
		FullName:                  fullName,
		Description:               description,
		Website:                   website,
		Location:                  location,
		RepoAdminChangeTeamAccess: repoAdminChangeTeamAccess,
	}

	switch visibility {
	case "public", "internal", "private":
		opts.Visibility = gitea.VisibleType(visibility)
	default:
		return errors.NewValidationError(fmt.Sprintf("invalid visibility: %s", visibility),
			map[string]interface{}{"valid_values": []string{"public", "internal", "private"}})
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create organization '%s'\n", orgName)
		printer.Printf("  Full Name: %s\n", opts.FullName)
		printer.Printf("  Description: %s\n", opts.Description)
		printer.Printf("  Website: %s\n", opts.Website)
		printer.Printf("  Location: %s\n", opts.Location)
		printer.Printf("  Visibility: %s\n", opts.Visibility)
		printer.Printf("  RepoAdminChangeTeamAccess: %v\n", opts.RepoAdminChangeTeamAccess)
		return nil
	}

	org, resp, err := client.CreateOrg(opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(org, func() error {
		printer.Printf("Successfully created organization '%s'\n", org.Name)
		printer.Printf("  ID: %d\n", org.ID)
		printer.Printf("  Full Name: %s\n", org.FullName)
		return nil
	})
}

func runOrgMembersList(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	if orgName == "" {
		return errors.NewValidationError("organization name required",
			map[string]interface{}{"usage": "teacli org members list <org-name>"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	members, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListOrgMembership(orgName, gitea.ListOrgMembershipOption{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(members, func() error {
		headers := []string{"ID", "Username", "Full Name", "Email", "Avatar"}
		var rows [][]string

		for _, member := range members {
			rows = append(rows, []string{
				fmt.Sprintf("%d", member.ID),
				member.UserName,
				member.FullName,
				member.Email,
				member.AvatarURL,
			})
		}

		if len(rows) == 0 {
			printer.Println(fmt.Sprintf("No members found in organization '%s'.", orgName))
			return nil
		}

		return printer.PrintTable(headers, rows)
	})
}

func runOrgMembersAdd(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	public, _ := cmd.Flags().GetBool("public")

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would add user '%s' to organization '%s'\n", username, orgName)
		if public {
			printer.Printf("DRY-RUN: Would make membership visible\n")
		}
		return nil
	}

	_, userResp, userErr := client.GetUserInfo(username)
	if userErr != nil {
		return errors.FromGiteaNotFound(userResp, userErr, "user", username)
	}

	if public {
		pubResp, pubErr := client.SetPublicOrgMembership(orgName, username, true)
		if pubErr != nil {
			return errors.FromGiteaNotFound(pubResp, pubErr, "organization", orgName)
		}
		printer := getPrinter(cmd)
		return printer.Emit(map[string]any{"status": "publicized", "org": orgName, "user": username}, func() error {
			printer.Printf("Successfully publicized membership of '%s' in organization '%s'\n", username, orgName)
			return nil
		})
	}

	return errors.NewValidationError(
		fmt.Sprintf("adding member '%s' to organization '%s' requires additional steps", username, orgName),
		map[string]interface{}{
			"note": "The Gitea SDK v0.23.2 does not expose a direct 'add member' function.",
			"alternatives": []string{
				"1. Have the user request to join the organization",
				"2. Create a team in the organization and add the user to the team",
				"3. Use the Gitea web UI to add the member",
			},
			"team_command": "teacli team create <org-name> <team-name> && teacli team add-member <team-id> <username>",
		})
}

func runOrgMembersRemove(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would remove user '%s' from organization '%s'\n", username, orgName)
		return nil
	}

	isMember, checkResp, checkErr := client.CheckOrgMembership(orgName, username)
	if checkErr != nil {
		return errors.FromGiteaNotFound(checkResp, checkErr, "organization", orgName)
	}

	if !isMember {
		return errors.NewValidationError(
			fmt.Sprintf("user '%s' is not a member of organization '%s'", username, orgName),
			map[string]interface{}{"user": username, "organization": orgName})
	}

	delResp, delErr := client.DeleteOrgMembership(orgName, username)
	if delErr != nil {
		return errors.FromGiteaNotFound(delResp, delErr, "organization member", orgName+"/"+username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "removed", "org": orgName, "user": username}, func() error {
		printer.Printf("Successfully removed user '%s' from organization '%s'\n", username, orgName)
		return nil
	})
}

func runOrgMembersCheck(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	isMember, resp, err := client.CheckOrgMembership(orgName, username)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "organization", orgName)
	}

	printer := getPrinter(cmd)

	return printer.Emit(map[string]any{
		"status":    "checked",
		"org":       orgName,
		"user":      username,
		"is_member": isMember,
	}, func() error {
		if isMember {
			printer.Printf("User '%s' IS a member of organization '%s'\n", username, orgName)
		} else {
			printer.Printf("User '%s' is NOT a member of organization '%s'\n", username, orgName)
		}

		return nil
	})
}

func runOrgDelete(cmd *cobra.Command, args []string) error {
	orgName := args[0]
	if orgName == "" {
		return errors.NewValidationError("organization name required",
			map[string]interface{}{"usage": "teacli org delete <org-name>"})
	}

	printer := getPrinter(cmd)

	skipConfirm, _ := cmd.Flags().GetBool("yes")
	if !skipConfirm {
		return errors.NewValidationError(
			fmt.Sprintf("deletion of organization '%s' requires confirmation", orgName),
			map[string]interface{}{
				"hint": "Use --yes flag to confirm deletion",
				"warning": "This action is irreversible and will delete all repositories, issues, " +
					"and other data associated with the organization",
			})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would delete organization '%s'\n", orgName)
		printer.Printf("DRY-RUN: This would delete all repositories, issues, and data\n")
		return nil
	}

	delResp, delErr := client.DeleteOrg(orgName)
	if delErr != nil {
		return errors.FromGiteaNotFound(delResp, delErr, "organization", orgName)
	}

	return printer.Emit(map[string]any{"status": "deleted", "org": orgName}, func() error {
		printer.Printf("Successfully deleted organization '%s'\n", orgName)
		return nil
	})
}

func validateOrgName(name string) error {
	if len(name) < 2 {
		return errors.NewValidationError("organization name must be at least 2 characters",
			map[string]interface{}{"min_length": 2})
	}

	if len(name) > 39 {
		return errors.NewValidationError("organization name must be at most 39 characters",
			map[string]interface{}{"max_length": 39})
	}

	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return errors.NewValidationError("organization name can only contain letters, numbers, hyphens, and underscores",
				map[string]interface{}{"invalid_character": string(r)})
		}
	}

	if name[0] == '-' || name[0] == '_' || name[len(name)-1] == '-' || name[len(name)-1] == '_' {
		return errors.NewValidationError("organization name cannot start or end with hyphen or underscore",
			map[string]interface{}{"rule": "name must start and end with alphanumeric character"})
	}

	return nil
}
