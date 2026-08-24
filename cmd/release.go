// Package cmd contains release commands
package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var releaseCmd = &cobra.Command{
	Use:     "release",
	Aliases: []string{"releases"},
	Short:   "Manage releases",
	Long:    `List, create, delete, and upload assets for releases`,
}

var releaseListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List releases",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReleaseList,
}

var releaseGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <tag>",
	Short: "Get release details",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runReleaseGet,
}

var releaseCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>]",
	Short: "Create a new release",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runReleaseCreate,
}

var releaseDeleteCmd = &cobra.Command{
	Use:   "delete [<owner>/<repo>] <tag>",
	Short: "Delete a release",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runReleaseDelete,
}

var releaseUploadCmd = &cobra.Command{
	Use:   "upload [<owner>/<repo>] <tag> <file-path>",
	Short: "Upload asset to release",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReleaseUpload,
}

func init() {
	RootCmd.AddCommand(releaseCmd)
	releaseCmd.AddCommand(releaseListCmd, releaseGetCmd, releaseCreateCmd, releaseDeleteCmd, releaseUploadCmd)

	releaseListCmd.Flags().Int("page", 0, "Page number")
	releaseListCmd.Flags().Int("per-page", 0, "Results per page")

	releaseCreateCmd.Flags().StringP("tag", "t", "", "Git tag (required)")
	releaseCreateCmd.MarkFlagRequired("tag")
	releaseCreateCmd.Flags().StringP("name", "n", "", "Release name (required)")
	releaseCreateCmd.MarkFlagRequired("name")
	releaseCreateCmd.Flags().StringP("body", "b", "", "Release notes")
	releaseCreateCmd.Flags().Bool("draft", false, "Create as draft")
	releaseCreateCmd.Flags().Bool("prerelease", false, "Mark as pre-release")
	releaseCreateCmd.Flags().String("target", "", "Target commit/branch")

	releaseUploadCmd.Flags().StringP("name", "n", "", "Asset name (defaults to filename)")
}

func runReleaseList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	releases, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Release, *gitea.Response, error) {
		return client.ListReleases(owner, repo, gitea.ListReleasesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)

	return printer.Emit(releases, func() error {
		for _, rel := range releases {
			draft := "draft"
			if !rel.IsDraft {
				draft = "released"
			}
			prerelease := ""
			if rel.IsPrerelease {
				prerelease = " (pre)"
			}
			printer.Printf("%s %s [%s%s]\n", rel.TagName, rel.Title, draft, prerelease)
		}
		return nil
	})
}

func runReleaseGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	tag := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	rel, resp, err := client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release", tag)
	}

	printer := getPrinter(cmd)
	return printer.Emit(rel, func() error {
		return printReleaseText(printer, rel)
	})
}

func runReleaseCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	tag, _ := cmd.Flags().GetString("tag")
	name, _ := cmd.Flags().GetString("name")
	body, _ := cmd.Flags().GetString("body")
	draft, _ := cmd.Flags().GetBool("draft")
	prerelease, _ := cmd.Flags().GetBool("prerelease")
	target, _ := cmd.Flags().GetString("target")

	if tag == "" || name == "" {
		return errors.NewValidationError("--tag and --name are required",
			map[string]interface{}{"missing_flags": []string{"--tag", "--name"}})
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateReleaseOption{
		TagName:      tag,
		Target:       target,
		Title:        name,
		Note:         body,
		IsDraft:      draft,
		IsPrerelease: prerelease,
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer := getPrinter(cmd)
		printer.Printf("DRY-RUN: Would create release\n")
		printer.Printf("  Tag: %s\n", tag)
		printer.Printf("  Name: %s\n", name)
		if body != "" {
			printer.Printf("  Body: %s\n", utils.Truncate(body, 80))
		}
		printer.Printf("  Draft: %v\n", draft)
		printer.Printf("  Pre-release: %v\n", prerelease)
		if target != "" {
			printer.Printf("  Target: %s\n", target)
		}
		return nil
	}

	rel, resp, err := client.CreateRelease(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(rel, func() error {
		printer.Printf("%s %s\n", rel.TagName, rel.HTMLURL)
		return nil
	})
}

func runReleaseDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	tag := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	rel, resp, err := client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release", tag)
	}
	releaseID := rel.ID

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would delete release '%s'\n", tag)
		return nil
	}

	resp, err = client.DeleteRelease(owner, repo, releaseID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release", tag)
	}

	return printer.Emit(map[string]any{"status": "deleted", "tag": tag}, func() error {
		printer.Printf("Deleted release '%s'\n", tag)
		return nil
	})
}

func runReleaseUpload(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}

	tag := args[1]
	filePath := args[2]

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return errors.NewValidationError(fmt.Sprintf("file not found: %s", filePath), nil)
	}

	assetName, _ := cmd.Flags().GetString("name")
	if assetName == "" {
		assetName = filepath.Base(filePath)
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	rel, resp, err := client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release", tag)
	}
	releaseID := rel.ID

	printer := getPrinter(cmd)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would upload '%s' to release '%s'\n", filePath, tag)
		printer.Printf("  Asset name: %s\n", assetName)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to open file: %v", err))
	}
	defer file.Close()

	asset, resp, err := client.CreateReleaseAttachment(owner, repo, releaseID, file, assetName)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release", tag)
	}

	return printer.Emit(asset, func() error {
		printer.Printf("%s %s\n", asset.Name, asset.DownloadURL)
		return nil
	})
}
