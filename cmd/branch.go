// Package cmd contains branch commands
package cmd

import (
	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var branchCmd = &cobra.Command{
	Use:     "branch",
	Aliases: []string{"branches"},
	Short:   "Manage branches",
	Long:    `List, create, and delete branches in repositories.`,
}

var branchListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List branches",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBranchList,
}

var branchGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <branch-name>",
	Short: "Get branch details",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranchGet,
}

var branchCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <branch-name>",
	Short: "Create a new branch",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranchCreate,
}

var branchDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <branch-name>",
	Short: "Delete a branch",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranchDelete,
}

func init() {
	RootCmd.AddCommand(branchCmd)
	branchCmd.AddCommand(branchListCmd, branchGetCmd, branchCreateCmd, branchDeleteCmd)

	branchListCmd.Flags().Int("page", 0, "Page number")
	branchListCmd.Flags().Int("per-page", 0, "Results per page")

	branchCreateCmd.Flags().StringP("from", "f", "", "Source branch to create from")
	branchCreateCmd.MarkFlagRequired("from")
}

func runBranchList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	branches, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Branch, *gitea.Response, error) {
		return client.ListRepoBranches(owner, repo, gitea.ListRepoBranchesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(branches, func() error {
		headers := []string{"Name", "Commit"}
		var rows [][]string

		for _, branch := range branches {
			commitID := branch.Commit.ID
			if len(commitID) > 8 {
				commitID = commitID[:8]
			}
			rows = append(rows, []string{
				branch.Name,
				commitID,
			})
		}

		return printer.PrintTable(headers, rows)
	})
}

func runBranchGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	branchName := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	branch, resp, err := client.GetRepoBranch(owner, repo, branchName)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "branch", branchName)
	}

	printer := getPrinter(cmd)
	return printer.Emit(branch, func() error {
		return printBranchText(printer, branch)
	})
}

func runBranchCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	branchName := args[1]
	fromBranch, _ := cmd.Flags().GetString("from")

	if fromBranch == "" {
		return errors.NewValidationError("--from is required",
			map[string]interface{}{"missing_flags": []string{"--from"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateBranchOption{
		BranchName:    branchName,
		OldBranchName: fromBranch,
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would create branch '%s' from '%s'\n", branchName, fromBranch)
		return nil
	}

	branch, resp, err := client.CreateBranch(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return printer.Emit(branch, func() error {
		printer.Printf("Created branch '%s'\n", branchName)
		return nil
	})
}

func runBranchDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	branchName := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would delete branch '%s'\n", branchName)
		return nil
	}

	_, resp, err := client.DeleteRepoBranch(owner, repo, branchName)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "branch", branchName)
	}

	return printer.Emit(map[string]any{"status": "deleted", "branch": branchName}, func() error {
		printer.Printf("Deleted branch '%s'\n", branchName)
		return nil
	})
}
