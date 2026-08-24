// Package cmd contains admin commands and the org/user webhook variants.
package cmd

import (
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var adminCmd = &cobra.Command{
	Use:   "admin",
	Short: "Server administration (every subcommand needs admin privileges)",
}

var adminOrgListCmd = &cobra.Command{
	Use:   "orgs",
	Short: "List every organization on the server",
	Args:  cobra.NoArgs,
	RunE:  runAdminOrgList,
}

var adminOrgCreateCmd = &cobra.Command{
	Use:   "create-org <owner> <org-name>",
	Short: "Create an organization owned by a given user",
	Args:  cobra.ExactArgs(2),
	RunE:  runAdminOrgCreate,
}

var adminRepoCreateCmd = &cobra.Command{
	Use:   "create-repo <owner> <name>",
	Short: "Create a repository on behalf of a user",
	Args:  cobra.ExactArgs(2),
	RunE:  runAdminRepoCreate,
}

var adminUnadoptedCmd = &cobra.Command{
	Use:   "unadopted",
	Short: "List repositories on disk that Gitea has no record of",
	Args:  cobra.NoArgs,
	RunE:  runAdminUnadopted,
}

// An unadopted repository is one the server has files for but no database entry,
// so there is no clone of it to stand in and nothing sensible to infer. Both of
// these keep their argument required.
var adminAdoptCmd = &cobra.Command{
	Use:   "adopt <owner>/<repo>",
	Short: "Adopt an unadopted repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminAdopt,
}

var adminDropUnadoptedCmd = &cobra.Command{
	Use:   "drop-unadopted <owner>/<repo>",
	Short: "Delete an unadopted repository from disk",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminDropUnadopted,
}

var adminCronListCmd = &cobra.Command{
	Use:   "cron",
	Short: "List the server's scheduled tasks",
	Args:  cobra.NoArgs,
	RunE:  runAdminCronList,
}

var adminCronRunCmd = &cobra.Command{
	Use:   "run-cron <task-name>",
	Short: "Run a scheduled task now",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminCronRun,
}

var adminEmailListCmd = &cobra.Command{
	Use:   "emails",
	Short: "List every email address on the server",
	Args:  cobra.NoArgs,
	RunE:  runAdminEmailList,
}

var adminHookCmd = &cobra.Command{
	Use:     "hook",
	Aliases: []string{"hooks"},
	Short:   "Manage system-wide webhooks",
}

var adminHookListCmd = &cobra.Command{
	Use:   "list",
	Short: "List system webhooks",
	Args:  cobra.NoArgs,
	RunE:  runAdminHookList,
}

var adminHookGetCmd = &cobra.Command{
	Use:   "get <hook-id>",
	Short: "Get one system webhook",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminHookGet,
}

var adminHookDeleteCmd = &cobra.Command{
	Use:     "delete <hook-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a system webhook",
	Args:    cobra.ExactArgs(1),
	RunE:    runAdminHookDelete,
}

var adminBadgeCmd = &cobra.Command{
	Use:     "badge",
	Aliases: []string{"badges"},
	Short:   "Manage the badges on a user's profile",
}

var adminBadgeListCmd = &cobra.Command{
	Use:   "list <username>",
	Short: "List a user's badges",
	Args:  cobra.ExactArgs(1),
	RunE:  runAdminBadgeList,
}

var adminBadgeAddCmd = &cobra.Command{
	Use:   "add <username> <badge-slug>...",
	Short: "Give badges to a user",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runAdminBadgeAdd,
}

var adminBadgeRemoveCmd = &cobra.Command{
	Use:     "remove <username> <badge-slug>...",
	Aliases: []string{"rm"},
	Short:   "Take badges away from a user",
	Args:    cobra.MinimumNArgs(2),
	RunE:    runAdminBadgeRemove,
}

var adminRenameUserCmd = &cobra.Command{
	Use:   "rename-user <username> <new-username>",
	Short: "Rename a user account",
	Args:  cobra.ExactArgs(2),
	RunE:  runAdminRenameUser,
}

func init() {
	RootCmd.AddCommand(adminCmd)
	adminCmd.AddCommand(adminOrgListCmd, adminOrgCreateCmd, adminRepoCreateCmd,
		adminUnadoptedCmd, adminAdoptCmd, adminDropUnadoptedCmd,
		adminCronListCmd, adminCronRunCmd, adminEmailListCmd, adminHookCmd,
		adminBadgeCmd, adminRenameUserCmd)
	adminHookCmd.AddCommand(adminHookListCmd, adminHookGetCmd, adminHookDeleteCmd)
	adminBadgeCmd.AddCommand(adminBadgeListCmd, adminBadgeAddCmd, adminBadgeRemoveCmd)

	addPageFlags(adminOrgListCmd, adminUnadoptedCmd, adminCronListCmd,
		adminEmailListCmd, adminHookListCmd)

	adminOrgCreateCmd.Flags().StringP("full-name", "f", "", "Organization full name")
	adminOrgCreateCmd.Flags().StringP("description", "d", "", "Organization description")
	adminOrgCreateCmd.Flags().StringP("visibility", "v", "public", "Visibility: public, limited, or private")

	adminRepoCreateCmd.Flags().StringP("description", "d", "", "Repository description")
	adminRepoCreateCmd.Flags().BoolP("private", "p", false, "Create the repository as private")
	adminRepoCreateCmd.Flags().Bool("no-auto-init", false, "Do not create an initial commit")

	adminUnadoptedCmd.Flags().StringP("pattern", "q", "", "Only list paths matching this pattern")
}

func runAdminOrgList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	orgs, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Organization, *gitea.Response, error) {
		return client.AdminListOrgs(gitea.AdminListOrgsOptions{ListOptions: lo})
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

func runAdminOrgCreate(cmd *cobra.Command, args []string) error {
	owner, name := args[0], args[1]

	fullName, _ := cmd.Flags().GetString("full-name")
	description, _ := cmd.Flags().GetString("description")
	visibility, _ := cmd.Flags().GetString("visibility")

	if dryRunf(cmd, "Would create organization %s owned by %s", name, owner) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	org, resp, err := client.AdminCreateOrg(owner, gitea.CreateOrgOption{
		Name: name, FullName: fullName, Description: description,
		Visibility: gitea.VisibleType(visibility),
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, org, "Created organization %s (id %d)", org.Name, org.ID)
}

func runAdminRepoCreate(cmd *cobra.Command, args []string) error {
	owner, name := args[0], args[1]

	description, _ := cmd.Flags().GetString("description")
	private, _ := cmd.Flags().GetBool("private")
	noAutoInit, _ := cmd.Flags().GetBool("no-auto-init")

	if dryRunf(cmd, "Would create repository %s/%s", owner, name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.AdminCreateRepo(owner, gitea.CreateRepoOption{
		Name: name, Description: description, Private: private, AutoInit: !noAutoInit,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, repo, "Created %s (%s)", repo.FullName, repo.HTMLURL)
}

func runAdminUnadopted(cmd *cobra.Command, args []string) error {
	pattern, _ := cmd.Flags().GetString("pattern")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	paths, err := fetchList(cmd, func(lo gitea.ListOptions) ([]string, *gitea.Response, error) {
		return client.ListUnadoptedRepos(gitea.ListUnadoptedReposOptions{
			ListOptions: lo, Pattern: pattern,
		})
	})
	if err != nil {
		return err
	}

	return emitStringList(cmd, paths, "No unadopted repositories.")
}

func runAdminAdopt(cmd *cobra.Command, args []string) error {
	owner, repo, err := repoArg(args[0])
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would adopt %s/%s", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.AdoptUnadoptedRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "unadopted repository", args[0])
	}
	return emitMessage(cmd, okMessage("repository adopted"), "Adopted %s/%s", owner, repo)
}

func runAdminDropUnadopted(cmd *cobra.Command, args []string) error {
	owner, repo, err := repoArg(args[0])
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete the unadopted repository %s/%s from disk", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteUnadoptedRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "unadopted repository", args[0])
	}
	return emitMessage(cmd, okMessage("repository deleted"), "Deleted %s/%s from disk", owner, repo)
}

func runAdminCronList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tasks, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.CronTask, *gitea.Response, error) {
		return client.ListCronTasks(gitea.ListCronTaskOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(tasks, func() error {
		if len(tasks) == 0 {
			printer.Println("No cron tasks.")
			return nil
		}
		rows := make([][]string, 0, len(tasks))
		for _, t := range tasks {
			rows = append(rows, []string{
				t.Name, t.Schedule, renderValue(t.Prev), renderValue(t.Next),
				fmt.Sprintf("%d", t.ExecTimes),
			})
		}
		return printer.PrintTable([]string{"Name", "Schedule", "Last run", "Next run", "Runs"}, rows)
	})
}

func runAdminCronRun(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would run cron task %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.RunCronTasks(args[0])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "cron task", args[0])
	}
	return emitMessage(cmd, okMessage("cron task started"), "Started cron task %s", args[0])
}

func runAdminEmailList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	emails, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Email, *gitea.Response, error) {
		return client.ListAdminEmails(gitea.ListAdminEmailsOptions{ListOptions: lo})
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
			rows = append(rows, []string{
				e.Email, e.Username, yesNo(e.Primary), yesNo(e.Verified),
			})
		}
		return printer.PrintTable([]string{"Email", "User", "Primary", "Verified"}, rows)
	})
}

func runAdminHookList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hooks, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Hook, *gitea.Response, error) {
		return client.ListAdminHooks(gitea.ListAdminHooksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitHookTable(cmd, hooks)
}

// emitHookTable renders a webhook list, shared by the repo, org, user and
// system hook commands.
func emitHookTable(cmd *cobra.Command, hooks []*gitea.Hook) error {
	printer := getPrinter(cmd)
	return printer.Emit(hooks, func() error {
		if len(hooks) == 0 {
			printer.Println("No webhooks found.")
			return nil
		}
		rows := make([][]string, 0, len(hooks))
		for _, h := range hooks {
			rows = append(rows, []string{
				fmt.Sprintf("%d", h.ID), h.Type, h.Config["url"],
				yesNo(h.Active), cell(renderValue(h.Events), 40),
			})
		}
		return printer.PrintTable([]string{"ID", "Type", "URL", "Active", "Events"}, rows)
	})
}

func runAdminHookGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "hook id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	hook, resp, err := client.GetAdminHook(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(hook, func() error {
		return printHookText(printer, hook)
	})
}

func runAdminHookDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "hook id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete system webhook %d", id) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteAdminHook(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "webhook", args[0])
	}
	return emitMessage(cmd, okMessage("webhook deleted"), "Deleted system webhook %d", id)
}

func runAdminBadgeList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	badges, resp, err := client.ListUserBadges(args[0])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(badges, func() error {
		if len(badges) == 0 {
			printer.Println("No badges.")
			return nil
		}
		rows := make([][]string, 0, len(badges))
		for _, b := range badges {
			rows = append(rows, []string{
				fmt.Sprintf("%d", b.ID), b.Slug, b.ImageURL, cell(b.Description, 40),
			})
		}
		return printer.PrintTable([]string{"ID", "Slug", "Image", "Description"}, rows)
	})
}

func runAdminBadgeAdd(cmd *cobra.Command, args []string) error {
	return changeUserBadges(cmd, args, true)
}

func runAdminBadgeRemove(cmd *cobra.Command, args []string) error {
	return changeUserBadges(cmd, args, false)
}

func changeUserBadges(cmd *cobra.Command, args []string, add bool) error {
	username, slugs := args[0], args[1:]

	verb := "take"
	if add {
		verb = "give"
	}
	if dryRunf(cmd, "Would %s %d badge(s) %s %s", verb, len(slugs), map[bool]string{true: "to", false: "from"}[add], username) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opt := gitea.UserBadgeOption{BadgeSlugs: slugs}

	var resp *gitea.Response
	if add {
		resp, err = client.AddUserBadges(username, opt)
	} else {
		resp, err = client.DeleteUserBadge(username, opt)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", username)
	}

	past := "Removed"
	if add {
		past = "Added"
	}
	return emitMessage(cmd, okMessage("badges updated"), "%s %d badge(s) for %s", past, len(slugs), username)
}

func runAdminRenameUser(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would rename user %s to %s", args[0], args[1]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.AdminRenameUser(args[0], gitea.RenameUserOption{NewUsername: args[1]})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}
	return emitMessage(cmd, okMessage("user renamed"), "Renamed %s to %s", args[0], args[1])
}
