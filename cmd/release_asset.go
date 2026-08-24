// Package cmd contains release edit and release asset commands.
package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var releaseUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <tag>",
	Aliases: []string{"edit"},
	Short:   "Edit a release",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runReleaseUpdate,
}

var releaseAssetCmd = &cobra.Command{
	Use:     "asset",
	Aliases: []string{"assets"},
	Short:   "Manage the files attached to a release",
}

var releaseAssetListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <tag>",
	Short: "List the assets of a release",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runReleaseAssetList,
}

var releaseAssetGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <tag> <asset-id>",
	Short: "Get one release asset",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReleaseAssetGet,
}

var releaseAssetUploadCmd = &cobra.Command{
	Use:     "upload [<owner>/<repo>] <tag> <file-path>",
	Aliases: []string{"create", "add"},
	Short:   "Upload a file to a release",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runReleaseAssetUpload,
}

var releaseAssetDownloadCmd = &cobra.Command{
	Use:   "download [<owner>/<repo>] <tag> <asset-id>",
	Short: "Download a release asset",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReleaseAssetDownload,
}

var releaseAssetRenameCmd = &cobra.Command{
	Use:     "rename [<owner>/<repo>] <tag> <asset-id> <new-name>",
	Aliases: []string{"update", "edit"},
	Short:   "Rename a release asset",
	Args:    cobra.RangeArgs(3, 4),
	RunE:    runReleaseAssetRename,
}

var releaseAssetDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <tag> <asset-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a release asset",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runReleaseAssetDelete,
}

func init() {
	releaseCmd.AddCommand(releaseUpdateCmd, releaseAssetCmd)
	releaseAssetCmd.AddCommand(releaseAssetListCmd, releaseAssetGetCmd, releaseAssetUploadCmd,
		releaseAssetDownloadCmd, releaseAssetRenameCmd, releaseAssetDeleteCmd)

	addPageFlags(releaseAssetListCmd)

	releaseUpdateCmd.Flags().String("tag", "", "New git tag")
	releaseUpdateCmd.Flags().StringP("name", "n", "", "New release title")
	releaseUpdateCmd.Flags().StringP("body", "b", "", "New release notes")
	releaseUpdateCmd.Flags().String("body-file", "", "Read the release notes from this file, or - for stdin")
	releaseUpdateCmd.Flags().String("target", "", "New target commitish")
	releaseUpdateCmd.Flags().Bool("draft", false, "Mark as a draft")
	releaseUpdateCmd.Flags().Bool("no-draft", false, "Publish a draft")
	releaseUpdateCmd.Flags().Bool("prerelease", false, "Mark as a prerelease")
	releaseUpdateCmd.Flags().Bool("no-prerelease", false, "Unmark as a prerelease")

	releaseAssetUploadCmd.Flags().String("name", "", "Store the file under this name instead of its basename")
	releaseAssetDownloadCmd.Flags().StringP("out", "o", "", "Write to this path (default: the asset's name)")
}

// releaseByTag resolves a tag to the release object, which the asset endpoints
// address by numeric ID rather than by tag.
func releaseByTag(cmd *cobra.Command, client *gitea.Client, owner, repo, tag string) (*gitea.Release, error) {
	rel, resp, err := client.GetReleaseByTag(owner, repo, tag)
	if err != nil {
		return nil, errors.FromGiteaNotFound(resp, err, "release", tag)
	}
	return rel, nil
}

func runReleaseUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	tag := args[1]

	note, err := bodyFromFlags(cmd, "body", "body-file")
	if err != nil {
		return err
	}

	opts := gitea.EditReleaseOption{
		TagName:      valueOrEmpty(cmd, "tag"),
		Title:        valueOrEmpty(cmd, "name"),
		Target:       valueOrEmpty(cmd, "target"),
		Note:         note,
		IsDraft:      boolPairFlag(cmd, "draft", "no-draft"),
		IsPrerelease: boolPairFlag(cmd, "prerelease", "no-prerelease"),
	}

	if opts.TagName == "" && opts.Title == "" && opts.Target == "" && opts.Note == "" &&
		opts.IsDraft == nil && opts.IsPrerelease == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --name or --body"})
	}

	if dryRunf(cmd, "Would edit release %s in %s/%s", tag, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, tag)
	if err != nil {
		return err
	}

	updated, resp, err := client.EditRelease(owner, repo, rel.ID, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(updated, func() error {
		return printReleaseText(printer, updated)
	})
}

// valueOrEmpty reads a string flag, returning "" when it was not given. The
// release edit endpoint treats empty as "leave unchanged".
func valueOrEmpty(cmd *cobra.Command, name string) string {
	v, _ := cmd.Flags().GetString(name)
	return v
}

func emitAssets(cmd *cobra.Command, assets []*gitea.Attachment) error {
	printer := getPrinter(cmd)
	return printer.Emit(assets, func() error {
		if len(assets) == 0 {
			printer.Println("No assets found.")
			return nil
		}
		rows := make([][]string, 0, len(assets))
		for _, a := range assets {
			rows = append(rows, []string{
				fmt.Sprintf("%d", a.ID), a.Name, fmt.Sprintf("%d", a.Size),
				fmt.Sprintf("%d", a.DownloadCount), a.DownloadURL,
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Size", "Downloads", "URL"}, rows)
	})
}

func runReleaseAssetList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	assets, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Attachment, *gitea.Response, error) {
		return client.ListReleaseAttachments(owner, repo, rel.ID, gitea.ListReleaseAttachmentsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitAssets(cmd, assets)
}

func runReleaseAssetGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	assetID, err := int64Arg(args[2], "asset id")
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	asset, resp, err := client.GetReleaseAttachment(owner, repo, rel.ID, assetID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release asset", args[2])
	}

	printer := getPrinter(cmd)
	return printer.Emit(asset, func() error {
		var d detail
		d.always("ID", asset.ID)
		d.add("Name", asset.Name)
		d.always("Size", asset.Size)
		d.always("Downloads", asset.DownloadCount)
		d.add("Created", asset.Created)
		d.add("URL", asset.DownloadURL)
		return d.print(printer)
	})
}

func runReleaseAssetUpload(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	path := args[2]

	name, _ := cmd.Flags().GetString("name")
	if name == "" {
		name = filepath.Base(path)
	}

	if dryRunf(cmd, "Would upload %s as %s to release %s", path, name, args[1]) {
		return nil
	}

	file, err := os.Open(path)
	if err != nil {
		return errors.NewValidationError(fmt.Sprintf("failed to open %s: %v", path, err), nil)
	}
	defer file.Close()

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	asset, resp, err := client.CreateReleaseAttachment(owner, repo, rel.ID, file, name)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, asset, "Uploaded %s (id %d) to release %s", asset.Name, asset.ID, args[1])
}

func runReleaseAssetDownload(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	assetID, err := int64Arg(args[2], "asset id")
	if err != nil {
		return err
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	asset, resp, err := client.GetReleaseAttachment(owner, repo, rel.ID, assetID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release asset", args[2])
	}

	out, _ := cmd.Flags().GetString("out")
	if out == "" {
		out = asset.Name
	}

	if dryRunf(cmd, "Would download %s to %s", asset.Name, out) {
		return nil
	}

	// The SDK exposes no asset download, and the browser download URL is a web
	// route rather than an API one, so fetch it with the authenticated client.
	httpResp, err := apiDo(cmd, http.MethodGet, asset.DownloadURL, nil, nil)
	if err != nil {
		return err
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode >= 400 {
		body, _ := io.ReadAll(httpResp.Body)
		return apiError(httpResp.StatusCode, body)
	}

	file, err := os.Create(out)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to create %s: %v", out, err))
	}
	defer file.Close()

	written, err := io.Copy(file, httpResp.Body)
	if err != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to write %s: %v", out, err))
	}

	return emitMessage(cmd, map[string]any{"path": out, "bytes": written},
		"Wrote %s (%d bytes)", out, written)
}

func runReleaseAssetRename(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 3)
	if err != nil {
		return err
	}
	assetID, err := int64Arg(args[2], "asset id")
	if err != nil {
		return err
	}
	newName := args[3]

	if dryRunf(cmd, "Would rename asset %d to %s", assetID, newName) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	asset, resp, err := client.EditReleaseAttachment(owner, repo, rel.ID, assetID,
		gitea.EditAttachmentOptions{Name: newName})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release asset", args[2])
	}

	return emitMessage(cmd, asset, "Renamed asset %d to %s", assetID, asset.Name)
}

func runReleaseAssetDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	assetID, err := int64Arg(args[2], "asset id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete asset %d from release %s", assetID, args[1]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	rel, err := releaseByTag(cmd, client, owner, repo, args[1])
	if err != nil {
		return err
	}

	resp, err := client.DeleteReleaseAttachment(owner, repo, rel.ID, assetID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "release asset", args[2])
	}

	return emitMessage(cmd, okMessage("asset deleted"), "Deleted asset %d from release %s", assetID, args[1])
}
