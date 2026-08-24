// Package cmd contains repository commands (simplified)
package cmd

import (
	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var repoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repos"},
	Short:   "Manage repositories",
	Long:    `List and create repositories`,
}

var repoListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repositories",
	Args:  cobra.NoArgs,
	RunE:  runRepoList,
}

var repoCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create a new repository",
	Args:  cobra.ExactArgs(1),
	RunE:  runRepoCreate,
}

var repoRenameCmd = &cobra.Command{
	Use:   "rename [<owner>/<old-name>] <new-name>",
	Short: "Rename a repository",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runRepoRename,
}

func init() {
	RootCmd.AddCommand(repoCmd)
	repoCmd.AddCommand(repoListCmd, repoCreateCmd, repoRenameCmd)

	repoCreateCmd.Flags().StringP("description", "d", "", "Repository description")
	repoCreateCmd.Flags().BoolP("private", "p", false, "Create private repository")
}

func runRepoList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	user, _ := cmd.Flags().GetString("user")
	org, _ := cmd.Flags().GetString("org")
	if user != "" && org != "" {
		return errors.NewValidationError("--user and --org cannot be combined", nil)
	}

	repos, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Repository, *gitea.Response, error) {
		switch {
		case user != "":
			return client.ListUserRepos(user, gitea.ListReposOptions{ListOptions: lo})
		case org != "":
			return client.ListOrgRepos(org, gitea.ListOrgReposOptions{ListOptions: lo})
		default:
			return client.ListMyRepos(gitea.ListReposOptions{ListOptions: lo})
		}
	})
	if err != nil {
		return err
	}

	return emitRepoTable(cmd, repos)
}

func runRepoCreate(cmd *cobra.Command, args []string) error {
	name := args[0]
	if name == "" {
		return errors.NewValidationError("repository name required",
			map[string]interface{}{"usage": "teacli repo create <name>"})
	}

	description, _ := cmd.Flags().GetString("description")
	private, _ := cmd.Flags().GetBool("private")
	org, _ := cmd.Flags().GetString("org")
	defaultBranch, _ := cmd.Flags().GetString("default-branch")
	gitignores, _ := cmd.Flags().GetString("gitignores")
	license, _ := cmd.Flags().GetString("license")
	readme, _ := cmd.Flags().GetString("readme")
	issueLabels, _ := cmd.Flags().GetString("issue-labels")
	template, _ := cmd.Flags().GetBool("template")
	noAutoInit, _ := cmd.Flags().GetBool("no-auto-init")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateRepoOption{
		Name:          name,
		Description:   description,
		Private:       private,
		AutoInit:      !noAutoInit,
		Template:      template,
		Gitignores:    gitignores,
		License:       license,
		Readme:        readme,
		IssueLabels:   issueLabels,
		DefaultBranch: defaultBranch,
	}

	if dryRunf(cmd, "Would create repository '%s'", name) {
		return nil
	}

	var (
		repo *gitea.Repository
		resp *gitea.Response
	)
	if org != "" {
		repo, resp, err = client.CreateOrgRepo(org, opts)
	} else {
		repo, resp, err = client.CreateRepo(opts)
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		printer.Printf("%s %s\n", repo.FullName, repo.HTMLURL)
		return nil
	})
}

func runRepoRename(cmd *cobra.Command, args []string) error {
	owner, oldName, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	ownerRepo := args[0]
	newName := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.EditRepoOption{
		Name: &newName,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would rename repository '%s' to '%s'\n", ownerRepo, newName)
		return nil
	}

	repo, resp, err := client.EditRepo(owner, oldName, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(repo, func() error {
		printer.Printf("Renamed repository to %s\n", repo.FullName)
		printer.Printf("New URL: %s\n", repo.HTMLURL)
		return nil
	})
}
