// Package cmd contains the account-level commands: emails, settings, search,
// activity, heatmap, and the organization and account webhooks.
package cmd

import (
	"fmt"
	"strings"
	"time"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var userSearchCmd = &cobra.Command{
	Use:   "search <query>",
	Short: "Search users by name or email",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserSearch,
}

var userOrgsCmd = &cobra.Command{
	Use:   "orgs [<username>]",
	Short: "List the organizations you, or another user, belong to",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUserOrgs,
}

var userActivityCmd = &cobra.Command{
	Use:     "activity <username>",
	Aliases: []string{"feed"},
	Short:   "Show a user's activity feed",
	Args:    cobra.ExactArgs(1),
	RunE:    runUserActivity,
}

var userHeatmapCmd = &cobra.Command{
	Use:   "heatmap <username>",
	Short: "Show a user's contribution heatmap data",
	Args:  cobra.ExactArgs(1),
	RunE:  runUserHeatmap,
}

var userEmailCmd = &cobra.Command{
	Use:     "email",
	Aliases: []string{"emails"},
	Short:   "Manage the email addresses on your account",
}

var userEmailListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your email addresses",
	Args:  cobra.NoArgs,
	RunE:  runUserEmailList,
}

var userEmailAddCmd = &cobra.Command{
	Use:   "add <email>...",
	Short: "Add email addresses to your account",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runUserEmailAdd,
}

var userEmailRemoveCmd = &cobra.Command{
	Use:     "remove <email>...",
	Aliases: []string{"rm"},
	Short:   "Remove email addresses from your account",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runUserEmailRemove,
}

var userSettingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show or change your profile settings",
	Args:  cobra.NoArgs,
	RunE:  runUserSettings,
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Manage organization and account webhooks (see 'webhook' for repository ones)",
}

var hookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your account's webhooks, or an organization's",
	Args:  cobra.NoArgs,
	RunE:  runHookList,
}

var hookGetCmd = &cobra.Command{
	Use:   "get <hook-id>",
	Short: "Get one webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runHookGet,
}

var hookCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a webhook on your account or an organization",
	Args:  cobra.NoArgs,
	RunE:  runHookCreate,
}

var hookDeleteCmd = &cobra.Command{
	Use:     "delete <hook-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a webhook",
	Args:    cobra.ExactArgs(1),
	RunE:    runHookDelete,
}

func init() {
	userCmd.AddCommand(userSearchCmd, userOrgsCmd, userActivityCmd, userHeatmapCmd,
		userEmailCmd, userSettingsCmd)
	userEmailCmd.AddCommand(userEmailListCmd, userEmailAddCmd, userEmailRemoveCmd)

	RootCmd.AddCommand(hookCmd)
	hookCmd.AddCommand(hookListCmd, hookGetCmd, hookCreateCmd, hookDeleteCmd)

	addPageFlags(userSearchCmd, userOrgsCmd, userActivityCmd, userEmailListCmd, hookListCmd)

	userSettingsCmd.Flags().String("full-name", "", "New full name")
	userSettingsCmd.Flags().String("website", "", "New website")
	userSettingsCmd.Flags().String("location", "", "New location")
	userSettingsCmd.Flags().String("description", "", "New profile description")
	userSettingsCmd.Flags().String("language", "", "New interface language, e.g. en-US")
	userSettingsCmd.Flags().String("theme", "", "New theme name")
	userSettingsCmd.Flags().Bool("hide-email", false, "Hide your email address")
	userSettingsCmd.Flags().Bool("show-email", false, "Show your email address")
	userSettingsCmd.Flags().Bool("hide-activity", false, "Hide your activity from your profile")
	userSettingsCmd.Flags().Bool("show-activity", false, "Show your activity on your profile")

	for _, c := range []*cobra.Command{hookListCmd, hookGetCmd, hookCreateCmd, hookDeleteCmd} {
		c.Flags().String("org", "", "Act on this organization's webhooks instead of your account's")
	}
	hookCreateCmd.Flags().StringP("url", "u", "", "Payload URL (required)")
	hookCreateCmd.MarkFlagRequired("url")
	hookCreateCmd.Flags().StringP("type", "t", "gitea", "Webhook type: gitea, slack, discord, ...")
	hookCreateCmd.Flags().StringSliceP("events", "e", []string{"push"}, "Events to deliver")
	hookCreateCmd.Flags().StringP("secret", "s", "", "Shared secret")
	hookCreateCmd.Flags().String("content-type", "json", "Payload content type: json or form")
	hookCreateCmd.Flags().Bool("inactive", false, "Create the webhook disabled")
	hookCreateCmd.Flags().String("branch-filter", "", "Only deliver events for branches matching this glob")
}

func runUserSearch(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.SearchUsers(gitea.SearchUsersOption{ListOptions: lo, KeyWord: args[0]})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No users found.")
}

func runUserOrgs(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	orgs, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Organization, *gitea.Response, error) {
		if len(args) == 1 {
			return client.ListUserOrgs(args[0], gitea.ListOrgsOptions{ListOptions: lo})
		}
		return client.ListMyOrgs(gitea.ListOrgsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(orgs, func() error {
		if len(orgs) == 0 {
			printer.Println("No organizations found.")
			return nil
		}
		rows := make([][]string, 0, len(orgs))
		for _, o := range orgs {
			rows = append(rows, []string{
				fmt.Sprintf("%d", o.ID), o.Name, o.FullName, o.Visibility, cell(o.Description, 40),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Full Name", "Visibility", "Description"}, rows)
	})
}

func runUserActivity(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	activities, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Activity, *gitea.Response, error) {
		return client.ListUserActivityFeeds(args[0], gitea.ListUserActivityFeedsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitActivities(cmd, activities)
}

func runUserHeatmap(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	heatmap, resp, err := client.GetUserHeatmap(args[0])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(heatmap, func() error {
		if len(heatmap) == 0 {
			printer.Println("No contributions.")
			return nil
		}
		var total int64
		rows := make([][]string, 0, len(heatmap))
		for _, h := range heatmap {
			total += h.Contributions
			rows = append(rows, []string{
				renderValue(time.Unix(int64(h.Timestamp), 0).UTC()), fmt.Sprintf("%d", h.Contributions),
			})
		}
		if err := printer.PrintTable([]string{"Date", "Contributions"}, rows); err != nil {
			return err
		}
		printer.Printf("\nTotal: %d\n", total)
		return nil
	})
}

func runUserEmailList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	emails, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Email, *gitea.Response, error) {
		return client.ListEmails(gitea.ListEmailsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(emails, func() error {
		if len(emails) == 0 {
			printer.Println("No email addresses.")
			return nil
		}
		rows := make([][]string, 0, len(emails))
		for _, e := range emails {
			rows = append(rows, []string{e.Email, yesNo(e.Primary), yesNo(e.Verified)})
		}
		return printer.PrintTable([]string{"Email", "Primary", "Verified"}, rows)
	})
}

func runUserEmailAdd(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would add %s to your account", strings.Join(args, ", ")) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	emails, resp, err := client.AddEmail(gitea.CreateEmailOption{Emails: args})
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return emitMessage(cmd, emails, "Added %d email address(es)", len(args))
}

func runUserEmailRemove(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would remove %s from your account", strings.Join(args, ", ")) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteEmail(gitea.DeleteEmailOption{Emails: args})
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return emitMessage(cmd, okMessage("emails removed"), "Removed %d email address(es)", len(args))
}

func runUserSettings(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.UserSettingsOptions{
		FullName:      stringFlagPtr(cmd, "full-name"),
		Website:       stringFlagPtr(cmd, "website"),
		Location:      stringFlagPtr(cmd, "location"),
		Description:   stringFlagPtr(cmd, "description"),
		Language:      stringFlagPtr(cmd, "language"),
		Theme:         stringFlagPtr(cmd, "theme"),
		HideEmail:     boolPairFlag(cmd, "hide-email", "show-email"),
		HideActivity:  boolPairFlag(cmd, "hide-activity", "show-activity"),
		DiffViewStyle: nil,
	}

	changing := opts.FullName != nil || opts.Website != nil || opts.Location != nil ||
		opts.Description != nil || opts.Language != nil || opts.Theme != nil ||
		opts.HideEmail != nil || opts.HideActivity != nil

	settings := &gitea.UserSettings{}
	if changing {
		if dryRunf(cmd, "Would update your profile settings") {
			return nil
		}
		updated, resp, err := client.UpdateUserSettings(opts)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
		settings = updated
	} else {
		current, resp, err := client.GetUserSettings()
		if err != nil {
			return errors.FromGitea(resp, err)
		}
		settings = current
	}

	printer := getPrinter(cmd)
	return printer.Emit(settings, func() error {
		var d detail
		d.add("Full name", settings.FullName)
		d.add("Website", settings.Website)
		d.add("Location", settings.Location)
		d.add("Description", settings.Description)
		d.add("Language", settings.Language)
		d.add("Theme", settings.Theme)
		d.always("Hide email", settings.HideEmail)
		d.always("Hide activity", settings.HideActivity)
		d.add("Diff view style", settings.DiffViewStyle)
		return d.print(printer)
	})
}

func runHookList(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hooks, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Hook, *gitea.Response, error) {
		if org != "" {
			return client.ListOrgHooks(org, gitea.ListHooksOptions{ListOptions: lo})
		}
		return client.ListMyHooks(gitea.ListHooksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitHookTable(cmd, hooks)
}

func runHookGet(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	id, err := int64Arg(args[0], "hook id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		hook *gitea.Hook
		resp *gitea.Response
	)
	if org != "" {
		hook, resp, err = client.GetOrgHook(org, id)
	} else {
		hook, resp, err = client.GetMyHook(id)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(hook, func() error {
		return printHookText(printer, hook)
	})
}

func runHookCreate(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	url, _ := cmd.Flags().GetString("url")
	hookType, _ := cmd.Flags().GetString("type")
	events, _ := cmd.Flags().GetStringSlice("events")
	secret, _ := cmd.Flags().GetString("secret")
	contentType, _ := cmd.Flags().GetString("content-type")
	inactive, _ := cmd.Flags().GetBool("inactive")
	branchFilter, _ := cmd.Flags().GetString("branch-filter")

	target := "your account"
	if org != "" {
		target = "organization " + org
	}
	if dryRunf(cmd, "Would create a webhook on %s pointing at %s", target, url) {
		return nil
	}

	config := map[string]string{"url": url, "content_type": contentType}
	if secret != "" {
		config["secret"] = secret
	}

	opts := gitea.CreateHookOption{
		Type:         gitea.HookType(hookType),
		Config:       config,
		Events:       events,
		BranchFilter: branchFilter,
		Active:       !inactive,
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		hook *gitea.Hook
		resp *gitea.Response
	)
	if org != "" {
		hook, resp, err = client.CreateOrgHook(org, opts)
	} else {
		hook, resp, err = client.CreateMyHook(opts)
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, hook, "Created webhook %d on %s", hook.ID, target)
}

func runHookDelete(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	id, err := int64Arg(args[0], "hook id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete webhook %d", id) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var resp *gitea.Response
	if org != "" {
		resp, err = client.DeleteOrgHook(org, id)
	} else {
		resp, err = client.DeleteMyHook(id)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", args[0])
	}

	return emitMessage(cmd, okMessage("webhook deleted"), "Deleted webhook %d", id)
}
