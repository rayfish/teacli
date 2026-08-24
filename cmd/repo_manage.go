// Package cmd contains the repository lifecycle commands: get, delete, fork,
// migrate, edit, search, transfer, topics and templates.
package cmd

import (
	"fmt"
	"sort"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/output"

	"github.com/spf13/cobra"
)

var repoGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>]",
	Short: "Get repository details",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoGet,
}

// Deleting is permanent and nothing prompts, so this one command keeps its
// repository argument required rather than inferring the repository you happen
// to be standing in.
var repoDeleteCmd = &cobra.Command{
	Use:     "delete <owner>/<repo>",
	Aliases: []string{"rm"},
	Short:   "Delete a repository (permanent, no confirmation prompt)",
	Args:    cobra.ExactArgs(1),
	RunE:    runRepoDelete,
}

var repoEditCmd = &cobra.Command{
	Use:     "edit [<owner>/<repo>]",
	Aliases: []string{"update"},
	Short:   "Edit repository settings",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runRepoEdit,
}

var repoForkCmd = &cobra.Command{
	Use:   "fork [<owner>/<repo>]",
	Short: "Fork a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoFork,
}

var repoForksCmd = &cobra.Command{
	Use:   "forks [<owner>/<repo>]",
	Short: "List the forks of a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoForks,
}

var repoMigrateCmd = &cobra.Command{
	Use:   "migrate <clone-url>",
	Short: "Migrate or mirror a remote repository into Gitea",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoMigrate,
}

var repoSearchCmd = &cobra.Command{
	Use:   "search [<keyword>]",
	Short: "Search repositories across the server",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoSearch,
}

var repoTransferCmd = &cobra.Command{
	Use:   "transfer [<owner>/<repo>] <new-owner>",
	Short: "Transfer a repository to another user or organization",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoTransfer,
}

var repoTransferAcceptCmd = &cobra.Command{
	Use:   "transfer-accept [<owner>/<repo>]",
	Short: "Accept a pending repository transfer",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoTransferAccept,
}

var repoTransferRejectCmd = &cobra.Command{
	Use:   "transfer-reject [<owner>/<repo>]",
	Short: "Reject a pending repository transfer",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoTransferReject,
}

var repoMirrorSyncCmd = &cobra.Command{
	Use:   "mirror-sync [<owner>/<repo>]",
	Short: "Trigger a sync of a mirrored repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoMirrorSync,
}

var repoLanguagesCmd = &cobra.Command{
	Use:   "languages [<owner>/<repo>]",
	Short: "Show the language breakdown of a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoLanguages,
}

var repoTemplateCmd = &cobra.Command{
	Use:   "from-template <template-owner>/<template-repo> <name>",
	Short: "Create a repository from a template repository",
	Args:  cobra.ExactArgs(2),
	RunE:  runRepoFromTemplate,
}

var repoTopicCmd = &cobra.Command{
	Use:     "topic",
	Aliases: []string{"topics"},
	Short:   "Manage repository topics",
}

var repoTopicListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List repository topics",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runRepoTopicList,
}

var repoTopicAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <topic>...",
	Short: "Add topics to a repository",
	Args:  cobra.MinimumNArgs(1),
	RunE:  runRepoTopicAdd,
}

var repoTopicRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <topic>...",
	Aliases: []string{"rm"},
	Short:   "Remove topics from a repository",
	Args:    cobra.MinimumNArgs(1),
	RunE:    runRepoTopicRemove,
}

var repoTopicSetCmd = &cobra.Command{
	Use:   "set [<owner>/<repo>] <topic>...",
	Short: "Replace the topic set of a repository",
	Args:  cobra.MinimumNArgs(0),
	RunE:  runRepoTopicSet,
}

func init() {
	repoCmd.AddCommand(
		repoGetCmd, repoDeleteCmd, repoEditCmd, repoForkCmd, repoForksCmd,
		repoMigrateCmd, repoSearchCmd, repoTransferCmd, repoTransferAcceptCmd,
		repoTransferRejectCmd, repoMirrorSyncCmd, repoLanguagesCmd,
		repoTemplateCmd, repoTopicCmd,
	)
	repoTopicCmd.AddCommand(repoTopicListCmd, repoTopicAddCmd, repoTopicRemoveCmd, repoTopicSetCmd)

	addPageFlags(repoListCmd, repoForksCmd, repoSearchCmd, repoTopicListCmd)

	repoListCmd.Flags().StringP("user", "u", "", "List this user's repositories instead of your own")
	repoListCmd.Flags().StringP("org", "o", "", "List this organization's repositories instead of your own")

	repoCreateCmd.Flags().String("org", "", "Create the repository under this organization")
	repoCreateCmd.Flags().String("default-branch", "", "Name of the initial branch")
	repoCreateCmd.Flags().String("gitignores", "", "Comma-separated .gitignore templates")
	repoCreateCmd.Flags().String("license", "", "License template name")
	repoCreateCmd.Flags().String("readme", "", "Readme template name")
	repoCreateCmd.Flags().String("issue-labels", "", "Issue label template name")
	repoCreateCmd.Flags().Bool("template", false, "Mark the new repository as a template")
	repoCreateCmd.Flags().Bool("no-auto-init", false, "Do not create an initial commit")

	repoEditCmd.Flags().String("name", "", "New repository name")
	repoEditCmd.Flags().StringP("description", "d", "", "New description")
	repoEditCmd.Flags().String("website", "", "New website URL")
	repoEditCmd.Flags().String("default-branch", "", "New default branch")
	repoEditCmd.Flags().Bool("private", false, "Make the repository private")
	repoEditCmd.Flags().Bool("public", false, "Make the repository public")
	repoEditCmd.Flags().Bool("template", false, "Mark as a template repository")
	repoEditCmd.Flags().Bool("no-template", false, "Unmark as a template repository")
	repoEditCmd.Flags().Bool("archived", false, "Archive the repository")
	repoEditCmd.Flags().Bool("unarchived", false, "Unarchive the repository")
	repoEditCmd.Flags().Bool("issues", false, "Enable the issue tracker")
	repoEditCmd.Flags().Bool("no-issues", false, "Disable the issue tracker")
	repoEditCmd.Flags().Bool("wiki", false, "Enable the wiki")
	repoEditCmd.Flags().Bool("no-wiki", false, "Disable the wiki")
	repoEditCmd.Flags().Bool("pull-requests", false, "Enable pull requests")
	repoEditCmd.Flags().Bool("no-pull-requests", false, "Disable pull requests")
	repoEditCmd.Flags().Bool("releases", false, "Enable releases")
	repoEditCmd.Flags().Bool("no-releases", false, "Disable releases")
	repoEditCmd.Flags().Bool("actions", false, "Enable Actions")
	repoEditCmd.Flags().Bool("no-actions", false, "Disable Actions")
	repoEditCmd.Flags().Bool("packages", false, "Enable packages")
	repoEditCmd.Flags().Bool("no-packages", false, "Disable packages")
	repoEditCmd.Flags().Bool("projects", false, "Enable projects")
	repoEditCmd.Flags().Bool("no-projects", false, "Disable projects")

	repoForkCmd.Flags().String("org", "", "Fork into this organization instead of your account")
	repoForkCmd.Flags().String("name", "", "Name for the fork")

	repoMigrateCmd.Flags().String("name", "", "Name of the new repository (required)")
	repoMigrateCmd.MarkFlagRequired("name")
	repoMigrateCmd.Flags().String("owner", "", "Owner of the new repository (defaults to you)")
	repoMigrateCmd.Flags().String("service", "", "Source service: git, github, gitlab, gitea, gogs, onedev, gitbucket, codebase")
	repoMigrateCmd.Flags().String("auth-username", "", "Username for the source repository")
	repoMigrateCmd.Flags().String("auth-password", "", "Password for the source repository")
	repoMigrateCmd.Flags().String("auth-token", "", "Token for the source repository")
	repoMigrateCmd.Flags().StringP("description", "d", "", "Description of the new repository")
	repoMigrateCmd.Flags().Bool("mirror", false, "Keep the repository as a mirror")
	repoMigrateCmd.Flags().String("mirror-interval", "", "Mirror sync interval, e.g. 8h")
	repoMigrateCmd.Flags().Bool("private", false, "Create the repository as private")
	repoMigrateCmd.Flags().Bool("wiki", false, "Migrate the wiki")
	repoMigrateCmd.Flags().Bool("issues", false, "Migrate issues")
	repoMigrateCmd.Flags().Bool("labels", false, "Migrate labels")
	repoMigrateCmd.Flags().Bool("milestones", false, "Migrate milestones")
	repoMigrateCmd.Flags().Bool("pull-requests", false, "Migrate pull requests")
	repoMigrateCmd.Flags().Bool("releases", false, "Migrate releases")
	repoMigrateCmd.Flags().Bool("lfs", false, "Migrate LFS objects")
	repoMigrateCmd.Flags().String("lfs-endpoint", "", "LFS endpoint to migrate from")

	repoSearchCmd.Flags().StringP("owner", "o", "", "Limit to repositories owned by this user or organization")
	repoSearchCmd.Flags().Bool("private", false, "Only private repositories")
	repoSearchCmd.Flags().Bool("public", false, "Only public repositories")
	repoSearchCmd.Flags().Bool("archived", false, "Only archived repositories")
	repoSearchCmd.Flags().Bool("not-archived", false, "Exclude archived repositories")
	repoSearchCmd.Flags().String("type", "", "Filter by kind: fork, source, mirror, collaborative")
	repoSearchCmd.Flags().String("sort", "", "Sort by: alpha, created, updated, size, id")
	repoSearchCmd.Flags().String("order", "", "Sort order: asc or desc")
	repoSearchCmd.Flags().Bool("topic", false, "Match the keyword against topics instead of names")
	repoSearchCmd.Flags().Bool("in-description", false, "Also match the keyword against descriptions")

	repoTemplateCmd.Flags().String("owner", "", "Owner of the new repository (defaults to you)")
	repoTemplateCmd.Flags().StringP("description", "d", "", "Description of the new repository")
	repoTemplateCmd.Flags().Bool("private", false, "Create the repository as private")
	repoTemplateCmd.Flags().Bool("git-content", true, "Copy the git content of the template")
	repoTemplateCmd.Flags().Bool("topics", false, "Copy the template's topics")
	repoTemplateCmd.Flags().Bool("git-hooks", false, "Copy the template's git hooks")
	repoTemplateCmd.Flags().Bool("webhooks", false, "Copy the template's webhooks")
	repoTemplateCmd.Flags().Bool("avatar", false, "Copy the template's avatar")
	repoTemplateCmd.Flags().Bool("labels", false, "Copy the template's labels")
}

// emitRepoTable renders a repository list the same way everywhere it appears.
func emitRepoTable(cmd *cobra.Command, repos []*gitea.Repository) error {
	printer := getPrinter(cmd)
	return printer.Emit(repos, func() error {
		if len(repos) == 0 {
			printer.Println("No repositories found.")
			return nil
		}
		rows := make([][]string, 0, len(repos))
		for _, r := range repos {
			rows = append(rows, []string{
				fmt.Sprintf("%d", r.ID),
				r.FullName,
				yesNo(r.Private),
				fmt.Sprintf("%d", r.Stars),
				cell(r.Description, 50),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Private", "Stars", "Description"}, rows)
	})
}

func printRepoText(printer *output.Printer, repo *gitea.Repository) error {
	var d detail
	d.always("ID", repo.ID)
	d.add("Name", repo.FullName)
	d.add("Description", repo.Description)
	d.always("Private", repo.Private)
	d.always("Fork", repo.Fork)
	d.always("Mirror", repo.Mirror)
	d.always("Archived", repo.Archived)
	d.always("Template", repo.Template)
	d.add("Owner", userName(repo.Owner))
	d.add("Default branch", repo.DefaultBranch)
	d.add("Website", repo.Website)
	d.always("Stars", repo.Stars)
	d.always("Forks", repo.Forks)
	d.always("Open issues", repo.OpenIssues)
	d.always("Open PRs", repo.OpenPulls)
	d.always("Size (KB)", repo.Size)
	d.add("Clone URL", repo.CloneURL)
	d.add("SSH URL", repo.SSHURL)
	d.add("URL", repo.HTMLURL)
	d.add("Created", repo.Created)
	d.add("Updated", repo.Updated)
	if repo.Parent != nil {
		d.add("Forked from", repo.Parent.FullName)
	}
	return d.print(printer)
}

func runRepoGet(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.GetRepo(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		return printRepoText(printer, repo)
	})
}

func runRepoDelete(cmd *cobra.Command, args []string) error {
	owner, name, err := repoArg(args[0])
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete repository %s/%s", owner, name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.DeleteRepo(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitMessage(cmd, okMessage("repository deleted"), "Deleted repository %s/%s", owner, name)
}

// boolPairFlag reads a flag pair such as --wiki/--no-wiki into an optional bool.
// Returning nil leaves the field untouched, which is what the edit endpoints
// expect for "no change".
func boolPairFlag(cmd *cobra.Command, on, off string) *bool {
	if cmd.Flags().Changed(on) {
		v := true
		return &v
	}
	if cmd.Flags().Changed(off) {
		v := false
		return &v
	}
	return nil
}

func stringFlagPtr(cmd *cobra.Command, name string) *string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	v, _ := cmd.Flags().GetString(name)
	return &v
}

func runRepoEdit(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	opts := gitea.EditRepoOption{
		Name:            stringFlagPtr(cmd, "name"),
		Description:     stringFlagPtr(cmd, "description"),
		Website:         stringFlagPtr(cmd, "website"),
		DefaultBranch:   stringFlagPtr(cmd, "default-branch"),
		Private:         boolPairFlag(cmd, "private", "public"),
		Template:        boolPairFlag(cmd, "template", "no-template"),
		Archived:        boolPairFlag(cmd, "archived", "unarchived"),
		HasIssues:       boolPairFlag(cmd, "issues", "no-issues"),
		HasWiki:         boolPairFlag(cmd, "wiki", "no-wiki"),
		HasPullRequests: boolPairFlag(cmd, "pull-requests", "no-pull-requests"),
		HasReleases:     boolPairFlag(cmd, "releases", "no-releases"),
		HasActions:      boolPairFlag(cmd, "actions", "no-actions"),
		HasPackages:     boolPairFlag(cmd, "packages", "no-packages"),
		HasProjects:     boolPairFlag(cmd, "projects", "no-projects"),
	}

	if isZeroEditRepo(opts) {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --description"})
	}

	if dryRunf(cmd, "Would edit repository %s/%s", owner, name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.EditRepo(owner, name, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		return printRepoText(printer, repo)
	})
}

func isZeroEditRepo(o gitea.EditRepoOption) bool {
	return o.Name == nil && o.Description == nil && o.Website == nil && o.DefaultBranch == nil &&
		o.Private == nil && o.Template == nil && o.Archived == nil && o.HasIssues == nil &&
		o.HasWiki == nil && o.HasPullRequests == nil && o.HasReleases == nil &&
		o.HasActions == nil && o.HasPackages == nil && o.HasProjects == nil
}

func runRepoFork(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	opts := gitea.CreateForkOption{
		Organization: stringFlagPtr(cmd, "org"),
		Name:         stringFlagPtr(cmd, "name"),
	}

	target := "your account"
	if opts.Organization != nil {
		target = *opts.Organization
	}
	if dryRunf(cmd, "Would fork %s/%s into %s", owner, name, target) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.CreateFork(owner, name, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		printer.Printf("%s %s\n", repo.FullName, repo.HTMLURL)
		return nil
	})
}

func runRepoForks(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repos, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Repository, *gitea.Response, error) {
		return client.ListForks(owner, name, gitea.ListForksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitRepoTable(cmd, repos)
}

func runRepoMigrate(cmd *cobra.Command, args []string) error {
	name, _ := cmd.Flags().GetString("name")
	repoOwner, _ := cmd.Flags().GetString("owner")
	service, _ := cmd.Flags().GetString("service")
	description, _ := cmd.Flags().GetString("description")
	authUser, _ := cmd.Flags().GetString("auth-username")
	authPass, _ := cmd.Flags().GetString("auth-password")
	authToken, _ := cmd.Flags().GetString("auth-token")
	mirrorInterval, _ := cmd.Flags().GetString("mirror-interval")
	lfsEndpoint, _ := cmd.Flags().GetString("lfs-endpoint")

	flag := func(n string) bool { v, _ := cmd.Flags().GetBool(n); return v }

	opts := gitea.MigrateRepoOption{
		RepoName:       name,
		RepoOwner:      repoOwner,
		CloneAddr:      args[0],
		Service:        gitea.GitServiceType(service),
		AuthUsername:   authUser,
		AuthPassword:   authPass,
		AuthToken:      authToken,
		Description:    description,
		Mirror:         flag("mirror"),
		MirrorInterval: mirrorInterval,
		Private:        flag("private"),
		Wiki:           flag("wiki"),
		Issues:         flag("issues"),
		Labels:         flag("labels"),
		Milestones:     flag("milestones"),
		PullRequests:   flag("pull-requests"),
		Releases:       flag("releases"),
		LFS:            flag("lfs"),
		LFSEndpoint:    lfsEndpoint,
	}

	if dryRunf(cmd, "Would migrate %s into %s", args[0], name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.MigrateRepo(opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		printer.Printf("%s %s\n", repo.FullName, repo.HTMLURL)
		return nil
	})
}

func runRepoSearch(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	keyword := ""
	if len(args) == 1 {
		keyword = args[0]
	}

	ownerName, _ := cmd.Flags().GetString("owner")
	repoType, _ := cmd.Flags().GetString("type")
	sortBy, _ := cmd.Flags().GetString("sort")
	order, _ := cmd.Flags().GetString("order")
	isTopic, _ := cmd.Flags().GetBool("topic")
	inDescription, _ := cmd.Flags().GetBool("in-description")

	base := gitea.SearchRepoOptions{
		Keyword:              keyword,
		KeywordIsTopic:       isTopic,
		KeywordInDescription: inDescription,
		IsPrivate:            boolPairFlag(cmd, "private", "public"),
		IsArchived:           boolPairFlag(cmd, "archived", "not-archived"),
		Type:                 gitea.RepoType(repoType),
		Sort:                 sortBy,
		Order:                order,
	}

	if ownerName != "" {
		user, resp, err := client.GetUserInfo(ownerName)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "user or organization", ownerName)
		}
		base.OwnerID = user.ID
	}

	repos, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Repository, *gitea.Response, error) {
		opts := base
		opts.ListOptions = lo
		return client.SearchRepos(opts)
	})
	if err != nil {
		return err
	}

	return emitRepoTable(cmd, repos)
}

func runRepoTransfer(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	newOwner := args[1]

	if dryRunf(cmd, "Would transfer %s/%s to %s", owner, name, newOwner) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.TransferRepo(owner, name, gitea.TransferRepoOption{NewOwner: newOwner})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	return emitMessage(cmd, repo, "Transfer of %s/%s to %s started", owner, name, newOwner)
}

func runRepoTransferAccept(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would accept the transfer of %s/%s", owner, name) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	repo, resp, err := client.AcceptRepoTransfer(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository transfer", args[0])
	}
	return emitMessage(cmd, repo, "Accepted the transfer of %s", repo.FullName)
}

func runRepoTransferReject(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would reject the transfer of %s/%s", owner, name) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	repo, resp, err := client.RejectRepoTransfer(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository transfer", args[0])
	}
	return emitMessage(cmd, repo, "Rejected the transfer of %s", repo.FullName)
}

func runRepoMirrorSync(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would sync the mirror %s/%s", owner, name) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.MirrorSync(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("mirror sync requested"), "Requested a mirror sync of %s/%s", owner, name)
}

func runRepoLanguages(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	langs, resp, err := client.GetRepoLanguages(owner, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(langs, func() error {
		if len(langs) == 0 {
			printer.Println("No languages detected.")
			return nil
		}
		var total int64
		names := make([]string, 0, len(langs))
		for name, bytes := range langs {
			names = append(names, name)
			total += bytes
		}
		sort.Slice(names, func(i, j int) bool { return langs[names[i]] > langs[names[j]] })

		rows := make([][]string, 0, len(names))
		for _, n := range names {
			share := 0.0
			if total > 0 {
				share = float64(langs[n]) / float64(total) * 100
			}
			rows = append(rows, []string{n, fmt.Sprintf("%d", langs[n]), fmt.Sprintf("%.1f%%", share)})
		}
		return printer.PrintTable([]string{"Language", "Bytes", "Share"}, rows)
	})
}

func runRepoFromTemplate(cmd *cobra.Command, args []string) error {
	templateOwner, templateRepo, err := repoArg(args[0])
	if err != nil {
		return err
	}
	name := args[1]

	repoOwner, _ := cmd.Flags().GetString("owner")
	if repoOwner == "" {
		client, err := getClient(cmd)
		if err != nil {
			return err
		}
		me, resp, err := client.GetMyUserInfo()
		if err != nil {
			return errors.FromGitea(resp, err)
		}
		repoOwner = me.UserName
	}

	description, _ := cmd.Flags().GetString("description")
	flag := func(n string) bool { v, _ := cmd.Flags().GetBool(n); return v }

	opts := gitea.CreateRepoFromTemplateOption{
		Owner:       repoOwner,
		Name:        name,
		Description: description,
		Private:     flag("private"),
		GitContent:  flag("git-content"),
		Topics:      flag("topics"),
		GitHooks:    flag("git-hooks"),
		Webhooks:    flag("webhooks"),
		Avatar:      flag("avatar"),
		Labels:      flag("labels"),
	}

	if dryRunf(cmd, "Would create %s/%s from template %s", repoOwner, name, args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repo, resp, err := client.CreateRepoFromTemplate(templateOwner, templateRepo, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "template repository", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		printer.Printf("%s %s\n", repo.FullName, repo.HTMLURL)
		return nil
	})
}

func runRepoTopicList(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	topics, err := fetchList(cmd, func(lo gitea.ListOptions) ([]string, *gitea.Response, error) {
		return client.ListRepoTopics(owner, name, gitea.ListRepoTopicsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(topics, func() error {
		if len(topics) == 0 {
			printer.Println("No topics set.")
			return nil
		}
		printer.Println(strings.Join(topics, "\n"))
		return nil
	})
}

func runRepoTopicAdd(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	topics := args[1:]

	if dryRunf(cmd, "Would add topics %s to %s/%s", strings.Join(topics, ", "), owner, name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	for _, t := range topics {
		resp, err := client.AddRepoTopic(owner, name, t)
		if err != nil {
			return errors.FromGitea(resp, err)
		}
	}

	return emitMessage(cmd, okMessage("topics added"), "Added %d topic(s) to %s/%s", len(topics), owner, name)
}

func runRepoTopicRemove(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	topics := args[1:]

	if dryRunf(cmd, "Would remove topics %s from %s/%s", strings.Join(topics, ", "), owner, name) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	for _, t := range topics {
		resp, err := client.DeleteRepoTopic(owner, name, t)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "topic", t)
		}
	}

	return emitMessage(cmd, okMessage("topics removed"), "Removed %d topic(s) from %s/%s", len(topics), owner, name)
}

func runRepoTopicSet(cmd *cobra.Command, args []string) error {
	owner, name, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	topics := args[1:]

	if dryRunf(cmd, "Would set the topics of %s/%s to %s", owner, name, strings.Join(topics, ", ")) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.SetRepoTopics(owner, name, topics)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, okMessage("topics set"), "Set %d topic(s) on %s/%s", len(topics), owner, name)
}
