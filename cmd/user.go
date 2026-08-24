// Package cmd contains user commands
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/output"

	"github.com/spf13/cobra"
)

var userCmd = &cobra.Command{
	Use:     "user",
	Aliases: []string{"users"},
	Short:   "Manage users",
	Long:    `Get, create, update, and delete users (admin functions require admin privileges)`,
}

var userCurrentCmd = &cobra.Command{
	Use:     "current",
	Aliases: []string{"whoami"},
	Short:   "Get current authenticated user",
	Args:    cobra.NoArgs,
	RunE:    runUserCurrent,
}

// whoamiCmd is a top-level alias for `user current`, the name tea and gh both
// use for it.
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Get the current authenticated user (alias for 'user current')",
	Args:  cobra.NoArgs,
	RunE:  runUserCurrent,
}

var userGetCmd = &cobra.Command{
	Use:   "get <username>",
	Short: "Get user details by username",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserGet,
}

var userCreateCmd = &cobra.Command{
	Use:   "create <username>",
	Short: "Create a new user (admin function)",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserCreate,
}

var userUpdateCmd = &cobra.Command{
	Use:   "update <username>",
	Short: "Update user details (admin function)",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserUpdate,
}

var userDeleteCmd = &cobra.Command{
	Use:   "delete <username>",
	Short: "Delete a user (admin function)",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserDelete,
}

var userListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all users (admin function)",
	Args:  cobra.NoArgs,
	RunE:  runUserList,
}

func init() {
	RootCmd.AddCommand(userCmd)
	RootCmd.AddCommand(whoamiCmd)
	userCmd.AddCommand(userCurrentCmd, userGetCmd, userCreateCmd, userUpdateCmd, userDeleteCmd, userListCmd)

	userCreateCmd.Flags().StringP("email", "e", "", "User email address (required)")
	userCreateCmd.MarkFlagRequired("email")
	userCreateCmd.Flags().StringP("password", "p", "", "User password (required)")
	userCreateCmd.MarkFlagRequired("password")
	userCreateCmd.Flags().StringP("full-name", "f", "", "User full name")
	userCreateCmd.Flags().BoolP("admin", "a", false, "Grant admin privileges")
	userCreateCmd.Flags().Bool("send-notify", false, "Send notification email to new user")
	userCreateCmd.Flags().Bool("must-change-password", false, "Force user to change password on first login")

	userUpdateCmd.Flags().StringP("email", "e", "", "New email address")
	userUpdateCmd.Flags().StringP("full-name", "f", "", "New full name")
	userUpdateCmd.Flags().StringP("password", "p", "", "New password")
	userUpdateCmd.Flags().StringP("description", "d", "", "User description")
	userUpdateCmd.Flags().StringP("website", "w", "", "User website")
	userUpdateCmd.Flags().StringP("location", "l", "", "User location")
	userUpdateCmd.Flags().BoolP("admin", "a", false, "Grant admin privileges")
	userUpdateCmd.Flags().Bool("no-admin", false, "Remove admin privileges")
	userUpdateCmd.Flags().Bool("active", false, "Activate user account")
	userUpdateCmd.Flags().Bool("inactive", false, "Deactivate user account")
	userUpdateCmd.Flags().Bool("restricted", false, "Set user as restricted")
	userUpdateCmd.Flags().Bool("unrestricted", false, "Remove restriction from user")
	userUpdateCmd.Flags().Bool("prohibit-login", false, "Prohibit user from logging in")
	userUpdateCmd.Flags().Bool("allow-login", false, "Allow user to log in")
	userUpdateCmd.Flags().Bool("must-change-password", false, "Force password change on next login")
	userUpdateCmd.Flags().Bool("no-must-change-password", false, "Remove password change requirement")

	userListCmd.Flags().StringP("query", "q", "", "Search query for username or email")
	userListCmd.Flags().Bool("admin", false, "Filter to only admin users")
	userListCmd.Flags().Bool("active", false, "Filter to only active users")
	userListCmd.Flags().Bool("restricted", false, "Filter to only restricted users")
	userListCmd.Flags().StringP("sort", "s", "", "Sort by: name, created, updated, id")
	userListCmd.Flags().String("order", "", "Sort order: asc, desc")
	userListCmd.Flags().Int("page", 0, "Page number")
	userListCmd.Flags().Int("per-page", 0, "Results per page")
}

func runUserCurrent(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	user, resp, err := client.GetMyUserInfo()
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(user, func() error {
		return printUserText(printer, user)
	})
}

func runUserGet(cmd *cobra.Command, args []string) error {
	username := args[0]
	if username == "" {
		return errors.NewValidationError("username required",
			map[string]interface{}{"usage": "teacli user get <username>"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	user, resp, err := client.GetUserInfo(username)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(user, func() error {
		return printUserText(printer, user)
	})
}

func runUserCreate(cmd *cobra.Command, args []string) error {
	username := args[0]
	email, _ := cmd.Flags().GetString("email")
	password, _ := cmd.Flags().GetString("password")
	fullName, _ := cmd.Flags().GetString("full-name")
	admin, _ := cmd.Flags().GetBool("admin")
	sendNotify, _ := cmd.Flags().GetBool("send-notify")
	mustChangePassword, _ := cmd.Flags().GetBool("must-change-password")

	if !isValidEmail(email) {
		return errors.NewValidationError(fmt.Sprintf("invalid email format: %s", email),
			map[string]interface{}{"hint": "Email must be a valid email address"})
	}

	if !isValidUsername(username) {
		return errors.NewValidationError(fmt.Sprintf("invalid username format: %s", username),
			map[string]interface{}{"hint": "Username must be 2-39 characters, containing only letters, numbers, hyphens, and underscores"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateUserOption{
		Username:   username,
		Email:      email,
		Password:   password,
		FullName:   fullName,
		SendNotify: sendNotify,
	}

	if mustChangePassword {
		opts.MustChangePassword = &mustChangePassword
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create user '%s'\n", username)
		printer.Printf("  Email: %s\n", email)
		if fullName != "" {
			printer.Printf("  Full Name: %s\n", fullName)
		}
		if admin {
			printer.Printf("  Admin: true\n")
		}
		if sendNotify {
			printer.Printf("  Send Notification: true\n")
		}
		if mustChangePassword {
			printer.Printf("  Must Change Password: true\n")
		}
		return nil
	}

	user, resp, err := client.AdminCreateUser(opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(user, func() error {
		return printUserText(printer, user)
	})
}

func runUserUpdate(cmd *cobra.Command, args []string) error {
	username := args[0]
	if username == "" {
		return errors.NewValidationError("username required",
			map[string]interface{}{"usage": "teacli user update <username> [options]"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.EditUserOption{}
	hasChanges := false

	if cmd.Flags().Changed("email") {
		email, _ := cmd.Flags().GetString("email")
		if email != "" && !isValidEmail(email) {
			return errors.NewValidationError(fmt.Sprintf("invalid email format: %s", email),
				map[string]interface{}{"hint": "Email must be a valid email address"})
		}
		opts.Email = &email
		hasChanges = true
	}

	if cmd.Flags().Changed("full-name") {
		fullName, _ := cmd.Flags().GetString("full-name")
		opts.FullName = &fullName
		hasChanges = true
	}

	if cmd.Flags().Changed("password") {
		password, _ := cmd.Flags().GetString("password")
		opts.Password = password
		hasChanges = true
	}

	if cmd.Flags().Changed("description") {
		desc, _ := cmd.Flags().GetString("description")
		opts.Description = &desc
		hasChanges = true
	}

	if cmd.Flags().Changed("website") {
		website, _ := cmd.Flags().GetString("website")
		opts.Website = &website
		hasChanges = true
	}

	if cmd.Flags().Changed("location") {
		location, _ := cmd.Flags().GetString("location")
		opts.Location = &location
		hasChanges = true
	}

	if cmd.Flags().Changed("admin") {
		admin := true
		opts.Admin = &admin
		hasChanges = true
	}
	if cmd.Flags().Changed("no-admin") {
		admin := false
		opts.Admin = &admin
		hasChanges = true
	}

	if cmd.Flags().Changed("active") {
		active := true
		opts.Active = &active
		hasChanges = true
	}
	if cmd.Flags().Changed("inactive") {
		active := false
		opts.Active = &active
		hasChanges = true
	}

	if cmd.Flags().Changed("restricted") {
		restricted := true
		opts.Restricted = &restricted
		hasChanges = true
	}
	if cmd.Flags().Changed("unrestricted") {
		restricted := false
		opts.Restricted = &restricted
		hasChanges = true
	}

	if cmd.Flags().Changed("prohibit-login") {
		prohibit := true
		opts.ProhibitLogin = &prohibit
		hasChanges = true
	}
	if cmd.Flags().Changed("allow-login") {
		prohibit := false
		opts.ProhibitLogin = &prohibit
		hasChanges = true
	}

	if cmd.Flags().Changed("must-change-password") {
		mustChange := true
		opts.MustChangePassword = &mustChange
		hasChanges = true
	}
	if cmd.Flags().Changed("no-must-change-password") {
		mustChange := false
		opts.MustChangePassword = &mustChange
		hasChanges = true
	}

	if !hasChanges {
		return errors.NewValidationError("at least one update option required",
			map[string]interface{}{
				"usage": "teacli user update <username> [options]",
				"options": []string{
					"--email", "--full-name", "--password", "--description",
					"--website", "--location", "--admin", "--no-admin",
					"--active", "--inactive", "--restricted", "--unrestricted",
					"--prohibit-login", "--allow-login", "--must-change-password",
					"--no-must-change-password",
				},
			})
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would update user '%s'\n", username)
		printChanges(&opts, printer)
		return nil
	}

	resp, err := client.AdminEditUser(username, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "updated", "user": username}, func() error {
		printer.Printf("User '%s' updated successfully\n", username)
		return nil
	})
}

func runUserDelete(cmd *cobra.Command, args []string) error {
	username := args[0]
	if username == "" {
		return errors.NewValidationError("username required",
			map[string]interface{}{"usage": "teacli user delete <username>"})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete user '%s'\n", username)
		return nil
	}

	resp, err := client.AdminDeleteUser(username)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "user": username}, func() error {
		printer.Printf("User '%s' deleted successfully\n", username)
		return nil
	})
}

func runUserList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.AdminListUsersOptions{
		ListOptions: gitea.ListOptions{
			Page:     1,
			PageSize: 50,
		},
	}

	if cmd.Flags().Changed("query") {
		query, _ := cmd.Flags().GetString("query")
		opts.Query = query
	}
	if cmd.Flags().Changed("sort") {
		sort, _ := cmd.Flags().GetString("sort")
		opts.Sort = sort
	}
	if cmd.Flags().Changed("order") {
		order, _ := cmd.Flags().GetString("order")
		opts.Order = order
	}

	if cmd.Flags().Changed("admin") {
		admin, _ := cmd.Flags().GetBool("admin")
		opts.IsAdmin = &admin
	}
	if cmd.Flags().Changed("active") {
		active, _ := cmd.Flags().GetBool("active")
		opts.IsActive = &active
	}
	if cmd.Flags().Changed("restricted") {
		restricted, _ := cmd.Flags().GetBool("restricted")
		opts.IsRestricted = &restricted
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		pageOpts := opts
		pageOpts.ListOptions = lo
		return client.AdminListUsers(pageOpts)
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(users, func() error {
		headers := []string{"ID", "Username", "Email", "Full Name", "Admin", "Active"}
		var rows [][]string

		for _, user := range users {
			rows = append(rows, []string{
				fmt.Sprintf("%d", user.ID),
				user.UserName,
				user.Email,
				user.FullName,
				fmt.Sprintf("%v", user.IsAdmin),
				fmt.Sprintf("%v", user.IsActive),
			})
		}

		return printer.PrintTable(headers, rows)
	})
}

func isValidEmail(email string) bool {
	if email == "" {
		return false
	}
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return false
	}
	local := parts[0]
	domain := parts[1]
	if len(local) == 0 || len(domain) == 0 {
		return false
	}
	if !strings.Contains(domain, ".") {
		return false
	}
	return true
}

func isValidUsername(username string) bool {
	if len(username) < 2 || len(username) > 39 {
		return false
	}
	for _, c := range username {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') ||
			(c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func printChanges(opts *gitea.EditUserOption, printer *output.Printer) {
	if opts.Email != nil {
		printer.Printf("  Email: %s\n", *opts.Email)
	}
	if opts.FullName != nil {
		printer.Printf("  Full Name: %s\n", *opts.FullName)
	}
	if opts.Password != "" {
		printer.Printf("  Password: [changed]\n")
	}
	if opts.Description != nil {
		printer.Printf("  Description: %s\n", *opts.Description)
	}
	if opts.Website != nil {
		printer.Printf("  Website: %s\n", *opts.Website)
	}
	if opts.Location != nil {
		printer.Printf("  Location: %s\n", *opts.Location)
	}
	if opts.Admin != nil {
		printer.Printf("  Admin: %v\n", *opts.Admin)
	}
	if opts.Active != nil {
		printer.Printf("  Active: %v\n", *opts.Active)
	}
	if opts.Restricted != nil {
		printer.Printf("  Restricted: %v\n", *opts.Restricted)
	}
	if opts.ProhibitLogin != nil {
		printer.Printf("  Prohibit Login: %v\n", *opts.ProhibitLogin)
	}
	if opts.MustChangePassword != nil {
		printer.Printf("  Must Change Password: %v\n", *opts.MustChangePassword)
	}
}
