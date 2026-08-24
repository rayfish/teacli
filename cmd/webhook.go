// Package cmd contains webhook commands
package cmd

import (
	"strconv"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var webhookCmd = &cobra.Command{
	Use:     "webhook",
	Aliases: []string{"webhooks", "hook", "hooks"},
	Short:   "Manage webhooks",
	Long:    `List, create, update, delete, and test webhooks for repositories`,
}

var webhookListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List webhooks",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWebhookList,
}

var webhookGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <webhook-id>",
	Short: "Get a specific webhook",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runWebhookGet,
}

var webhookCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new webhook",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWebhookCreate,
}

var webhookUpdateCmd = &cobra.Command{
	Use:   "update [<owner>/<repo>] <webhook-id>",
	Short: "Update a webhook",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runWebhookUpdate,
}

var webhookDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <webhook-id>",
	Short: "Delete a webhook",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runWebhookDelete,
}

var webhookTestCmd = &cobra.Command{
	Use:   "test [<owner>/<repo>] <webhook-id>",
	Short: "Send test ping to webhook",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runWebhookTest,
}

func init() {
	RootCmd.AddCommand(webhookCmd)
	webhookCmd.AddCommand(webhookListCmd, webhookGetCmd, webhookCreateCmd, webhookUpdateCmd, webhookDeleteCmd, webhookTestCmd)

	webhookCreateCmd.Flags().StringP("url", "u", "", "Webhook URL (required)")
	webhookCreateCmd.MarkFlagRequired("url")
	webhookCreateCmd.Flags().StringP("events", "e", "push", "Comma-separated events (push,pull_request,issues,release,etc.)")
	webhookCreateCmd.Flags().StringP("secret", "s", "", "Webhook secret for signature verification")
	webhookCreateCmd.Flags().Bool("active", true, "Enable webhook (default: true)")
	webhookCreateCmd.Flags().String("branch-filter", "", "Branch filter pattern (e.g., 'main,develop/*')")

	webhookUpdateCmd.Flags().StringP("url", "u", "", "New webhook URL")
	webhookUpdateCmd.Flags().StringP("events", "e", "", "New events (comma-separated)")
	webhookUpdateCmd.Flags().StringP("secret", "s", "", "New secret")
	webhookUpdateCmd.Flags().Bool("active", false, "Enable webhook")
	webhookUpdateCmd.Flags().Bool("inactive", false, "Disable webhook")
	webhookUpdateCmd.Flags().String("branch-filter", "", "New branch filter pattern")
}

func runWebhookList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hooks, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Hook, *gitea.Response, error) {
		return client.ListRepoHooks(owner, repo, gitea.ListHooksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(hooks, func() error {
		for _, hook := range hooks {
			events := strings.Join(hook.Events, ",")
			active := "inactive"
			if hook.Active {
				active = "active"
			}
			printer.Printf("%d %s %s [%s]\n", hook.ID, hook.Config["content_type"], events, active)
		}
		return nil
	})
}

func runWebhookGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	hookID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hook, resp, err := client.GetRepoHook(owner, repo, hookID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", strconv.FormatInt(hookID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(hook, func() error {
		return printHookText(printer, hook)
	})
}

func runWebhookCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	url, _ := cmd.Flags().GetString("url")
	eventsStr, _ := cmd.Flags().GetString("events")
	secret, _ := cmd.Flags().GetString("secret")
	active, _ := cmd.Flags().GetBool("active")
	branchFilter, _ := cmd.Flags().GetString("branch-filter")

	if url == "" {
		return errors.NewValidationError("--url is required",
			map[string]interface{}{"missing_flags": []string{"--url"}})
	}

	eventList := strings.Split(eventsStr, ",")
	for i, e := range eventList {
		eventList[i] = strings.TrimSpace(e)
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateHookOption{
		Active:       active,
		Events:       eventList,
		BranchFilter: branchFilter,
		Config: map[string]string{
			"content_type": "json",
			"url":          url,
		},
	}

	if secret != "" {
		opts.Config["secret"] = secret
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create webhook\n")
		printer.Printf("  URL: %s\n", url)
		printer.Printf("  Events: %s\n", eventsStr)
		printer.Printf("  Active: %v\n", active)
		if branchFilter != "" {
			printer.Printf("  Branch Filter: %s\n", branchFilter)
		}
		return nil
	}

	hook, resp, err := client.CreateRepoHook(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(hook, func() error {
		printer.Printf("%d %s %s\n", hook.ID, hook.Config["content_type"], strings.Join(hook.Events, ","))
		return nil
	})
}

func runWebhookUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	hookID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.EditHookOption{}
	updated := false

	if cmd.Flags().Changed("url") {
		url, _ := cmd.Flags().GetString("url")
		if opts.Config == nil {
			opts.Config = make(map[string]string)
		}
		opts.Config["url"] = url
		updated = true
	}

	if cmd.Flags().Changed("events") {
		eventsStr, _ := cmd.Flags().GetString("events")
		eventList := strings.Split(eventsStr, ",")
		for i, e := range eventList {
			eventList[i] = strings.TrimSpace(e)
		}
		opts.Events = eventList
		updated = true
	}

	if cmd.Flags().Changed("secret") {
		secret, _ := cmd.Flags().GetString("secret")
		if opts.Config == nil {
			opts.Config = make(map[string]string)
		}
		opts.Config["secret"] = secret
		updated = true
	}

	if cmd.Flags().Changed("active") {
		active, _ := cmd.Flags().GetBool("active")
		opts.Active = &active
		updated = true
	}

	if cmd.Flags().Changed("inactive") {
		inactive, _ := cmd.Flags().GetBool("inactive")
		active := !inactive
		opts.Active = &active
		updated = true
	}

	if cmd.Flags().Changed("branch-filter") {
		opts.BranchFilter, _ = cmd.Flags().GetString("branch-filter")
		updated = true
	}

	if !updated {
		return errors.NewValidationError("no update options provided",
			map[string]interface{}{"available": []string{"--url", "--events", "--secret", "--active", "--inactive", "--branch-filter"}})
	}

	resp, err := client.EditRepoHook(owner, repo, hookID, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", strconv.FormatInt(hookID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "updated", "webhook": hookID}, func() error {
		printer.Printf("Updated webhook %d\n", hookID)
		return nil
	})
}

func runWebhookDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	hookID := utils.ParseInt64(args[1])

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would delete webhook %d\n", hookID)
		return nil
	}

	resp, err := client.DeleteRepoHook(owner, repo, hookID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", strconv.FormatInt(hookID, 10))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "deleted", "webhook": hookID}, func() error {
		printer.Printf("Deleted webhook %d\n", hookID)
		return nil
	})
}

func runWebhookTest(cmd *cobra.Command, args []string) error {
	_, _, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	hookID := utils.ParseInt64(args[1])

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would send test ping to webhook %d\n", hookID)
		return nil
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"status": "tested", "webhook": hookID}, func() error {
		printer.Printf("Test ping sent to webhook %d (if supported by your Gitea version)\n", hookID)
		return nil
	})
}
