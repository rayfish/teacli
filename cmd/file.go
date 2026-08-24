// Package cmd contains repository file and git object commands.
package cmd

import (
	"encoding/base64"
	"fmt"
	"os"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var fileCmd = &cobra.Command{
	Use:     "file",
	Aliases: []string{"files", "contents"},
	Short:   "Read and write files in a repository",
}

var fileListCmd = &cobra.Command{
	Use:     "list [<owner>/<repo>] [<path>]",
	Aliases: []string{"ls"},
	Short:   "List the contents of a directory",
	Args:    cobra.RangeArgs(0, 2),
	RunE:    runFileList,
}

var fileGetCmd = &cobra.Command{
	Use:     "get [<owner>/<repo>] <path>",
	Aliases: []string{"cat", "read"},
	Short:   "Print a file's contents",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runFileGet,
}

var fileInfoCmd = &cobra.Command{
	Use:   "info [<owner>/<repo>] <path>",
	Short: "Show a file's metadata without its contents",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runFileInfo,
}

var fileCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <path>",
	Short: "Create a file and commit it",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runFileCreate,
}

var fileUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <path>",
	Aliases: []string{"edit"},
	Short:   "Update a file and commit the change",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runFileUpdate,
}

var fileDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <path>",
	Aliases: []string{"rm"},
	Short:   "Delete a file and commit the change",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runFileDelete,
}

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Inspect raw git objects: refs, trees, blobs and notes",
}

var gitRefListCmd = &cobra.Command{
	Use:   "refs [<owner>/<repo>] [<ref-prefix>]",
	Short: "List git references, optionally under a prefix such as heads or tags",
	Args:  cobra.RangeArgs(0, 2),
	RunE:  runGitRefs,
}

var gitTreeCmd = &cobra.Command{
	Use:   "tree [<owner>/<repo>] <ref>",
	Short: "List the tree at a ref",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runGitTree,
}

var gitBlobCmd = &cobra.Command{
	Use:   "blob [<owner>/<repo>] <sha>",
	Short: "Print a blob's contents",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runGitBlob,
}

var gitNoteCmd = &cobra.Command{
	Use:   "note [<owner>/<repo>] <sha>",
	Short: "Show the git note attached to a commit",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runGitNote,
}

func init() {
	RootCmd.AddCommand(fileCmd, gitCmd)
	fileCmd.AddCommand(fileListCmd, fileGetCmd, fileInfoCmd, fileCreateCmd, fileUpdateCmd, fileDeleteCmd)
	gitCmd.AddCommand(gitRefListCmd, gitTreeCmd, gitBlobCmd, gitNoteCmd)

	addPageFlags(gitTreeCmd)

	// The positional path is ambiguous once the repository is inferred, since
	// a path contains slashes too, so these give an unambiguous way to pass one.
	fileListCmd.Flags().String("path", "", "Directory to list, for use with an inferred repository")
	gitRefListCmd.Flags().String("ref-prefix", "", "Reference prefix such as heads or tags, for use with an inferred repository")

	for _, c := range []*cobra.Command{fileListCmd, fileGetCmd, fileInfoCmd} {
		c.Flags().StringP("ref", "r", "", "Branch, tag or commit to read from (defaults to the default branch)")
	}
	fileGetCmd.Flags().StringP("out", "o", "", "Write to this file instead of stdout")

	for _, c := range []*cobra.Command{fileCreateCmd, fileUpdateCmd, fileDeleteCmd} {
		c.Flags().StringP("message", "m", "", "Commit message")
		c.Flags().StringP("branch", "b", "", "Branch to commit on (defaults to the default branch)")
		c.Flags().String("new-branch", "", "Create this branch from --branch and commit there")
		c.Flags().String("author-name", "", "Commit author name")
		c.Flags().String("author-email", "", "Commit author email")
		c.Flags().Bool("signoff", false, "Add a Signed-off-by trailer")
	}
	for _, c := range []*cobra.Command{fileCreateCmd, fileUpdateCmd} {
		c.Flags().StringP("content", "c", "", "File content")
		c.Flags().StringP("file", "f", "", "Read the content from this local file, or - for stdin")
	}
	fileUpdateCmd.Flags().String("sha", "", "Blob SHA of the file being replaced (looked up when omitted)")
	fileUpdateCmd.Flags().String("from-path", "", "Move the file from this path")
	fileDeleteCmd.Flags().String("sha", "", "Blob SHA of the file being deleted (looked up when omitted)")

	gitTreeCmd.Flags().Bool("recursive", false, "Walk into subdirectories")
	gitBlobCmd.Flags().StringP("out", "o", "", "Write to this file instead of stdout")
}

func refFlag(cmd *cobra.Command) string {
	v, _ := cmd.Flags().GetString("ref")
	return v
}

// fileCommitOptions collects the flags every write command shares.
func fileCommitOptions(cmd *cobra.Command) gitea.FileOptions {
	message, _ := cmd.Flags().GetString("message")
	branch, _ := cmd.Flags().GetString("branch")
	newBranch, _ := cmd.Flags().GetString("new-branch")
	authorName, _ := cmd.Flags().GetString("author-name")
	authorEmail, _ := cmd.Flags().GetString("author-email")
	signoff, _ := cmd.Flags().GetBool("signoff")

	return gitea.FileOptions{
		Message:       message,
		BranchName:    branch,
		NewBranchName: newBranch,
		Author:        gitea.Identity{Name: authorName, Email: authorEmail},
		Signoff:       signoff,
	}
}

// resolveFileSHA finds the current blob SHA of a path, which update and delete
// need for optimistic concurrency. Asking the caller for it every time would be
// busywork, so look it up when --sha is absent.
func resolveFileSHA(cmd *cobra.Command, client *gitea.Client, owner, repo, path string) (string, error) {
	if sha, _ := cmd.Flags().GetString("sha"); sha != "" {
		return sha, nil
	}
	branch, _ := cmd.Flags().GetString("branch")
	contents, resp, err := client.GetContents(owner, repo, branch, path)
	if err != nil {
		return "", errors.FromGiteaNotFound(resp, err, "file", path)
	}
	return contents.SHA, nil
}

func runFileList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	path, _ := cmd.Flags().GetString("path")
	if len(args) == 2 {
		path = args[1]
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	entries, resp, err := client.ListContents(owner, repo, refFlag(cmd), path)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "path", path)
	}

	printer := getPrinter(cmd)
	return printer.Emit(entries, func() error {
		if len(entries) == 0 {
			printer.Println("Empty directory.")
			return nil
		}
		rows := make([][]string, 0, len(entries))
		for _, e := range entries {
			rows = append(rows, []string{e.Type, fmt.Sprintf("%d", e.Size), shortSHA(e.SHA), e.Path})
		}
		return printer.PrintTable([]string{"Type", "Size", "SHA", "Path"}, rows)
	})
}

func runFileGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	path := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	data, resp, err := client.GetFile(owner, repo, refFlag(cmd), path)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "file", path)
	}

	return writeBlob(cmd, data, path)
}

// writeBlob sends binary content to --out or stdout. Under --format json it is
// wrapped so the output stays parseable.
func writeBlob(cmd *cobra.Command, data []byte, name string) error {
	out, _ := cmd.Flags().GetString("out")
	if out != "" {
		if dryRunf(cmd, "Would write %d bytes to %s", len(data), out) {
			return nil
		}
		if err := os.WriteFile(out, data, 0o644); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to write %s: %v", out, err))
		}
		return emitMessage(cmd, map[string]any{"path": out, "bytes": len(data)},
			"Wrote %s (%d bytes)", out, len(data))
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]any{"name": name, "content": string(data)}, func() error {
		os.Stdout.Write(data)
		if len(data) > 0 && data[len(data)-1] != '\n' {
			printer.Println()
		}
		return nil
	})
}

func runFileInfo(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	path := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	contents, resp, err := client.GetContents(owner, repo, refFlag(cmd), path)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "file", path)
	}

	printer := getPrinter(cmd)
	return printer.Emit(contents, func() error {
		var d detail
		d.add("Path", contents.Path)
		d.add("Name", contents.Name)
		d.add("Type", contents.Type)
		d.always("Size", contents.Size)
		d.add("SHA", contents.SHA)
		d.add("Last commit", contents.LastCommitSha)
		if contents.DownloadURL != nil {
			d.add("Download", *contents.DownloadURL)
		}
		if contents.HTMLURL != nil {
			d.add("URL", *contents.HTMLURL)
		}
		return d.print(printer)
	})
}

func runFileCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	path := args[1]

	content, err := bodyFromFlags(cmd, "content", "file")
	if err != nil {
		return err
	}
	if content == "" {
		return errors.NewValidationError("file content required",
			map[string]interface{}{"hint": "pass --content <text> or --file <path>"})
	}

	if dryRunf(cmd, "Would create %s in %s/%s", path, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	result, resp, err := client.CreateFile(owner, repo, path, gitea.CreateFileOptions{
		FileOptions: fileCommitOptions(cmd),
		Content:     base64.StdEncoding.EncodeToString([]byte(content)),
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitFileResponse(cmd, result, "Created", path)
}

func runFileUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	path := args[1]

	content, err := bodyFromFlags(cmd, "content", "file")
	if err != nil {
		return err
	}
	if content == "" {
		return errors.NewValidationError("file content required",
			map[string]interface{}{"hint": "pass --content <text> or --file <path>"})
	}
	fromPath, _ := cmd.Flags().GetString("from-path")

	if dryRunf(cmd, "Would update %s in %s/%s", path, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	sha, err := resolveFileSHA(cmd, client, owner, repo, path)
	if err != nil {
		return err
	}

	result, resp, err := client.UpdateFile(owner, repo, path, gitea.UpdateFileOptions{
		FileOptions: fileCommitOptions(cmd),
		SHA:         sha,
		Content:     base64.StdEncoding.EncodeToString([]byte(content)),
		FromPath:    fromPath,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitFileResponse(cmd, result, "Updated", path)
}

func runFileDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	path := args[1]

	if dryRunf(cmd, "Would delete %s from %s/%s", path, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	sha, err := resolveFileSHA(cmd, client, owner, repo, path)
	if err != nil {
		return err
	}

	resp, err := client.DeleteFile(owner, repo, path, gitea.DeleteFileOptions{
		FileOptions: fileCommitOptions(cmd),
		SHA:         sha,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, okMessage("file deleted"), "Deleted %s from %s/%s", path, owner, repo)
}

func emitFileResponse(cmd *cobra.Command, result *gitea.FileResponse, verb, path string) error {
	printer := getPrinter(cmd)
	return printer.Emit(result, func() error {
		sha := ""
		if result != nil && result.Commit != nil {
			sha = shortSHA(result.Commit.SHA)
		}
		printer.Printf("%s %s in commit %s\n", verb, path, sha)
		return nil
	})
}

func runGitRefs(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if prefix, _ := cmd.Flags().GetString("ref-prefix"); prefix != "" && len(args) == 1 {
		args = append(args, prefix)
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		refs []*gitea.Reference
		resp *gitea.Response
	)
	if len(args) == 2 {
		refs, resp, err = client.GetRepoRefs(owner, repo, args[1])
	} else {
		refs, resp, err = client.ListAllGitRefs(owner, repo)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "ref", strings.Join(args[1:], ""))
	}

	printer := getPrinter(cmd)
	return printer.Emit(refs, func() error {
		if len(refs) == 0 {
			printer.Println("No refs found.")
			return nil
		}
		rows := make([][]string, 0, len(refs))
		for _, r := range refs {
			kind, sha := "", ""
			if r.Object != nil {
				kind, sha = r.Object.Type, shortSHA(r.Object.SHA)
			}
			rows = append(rows, []string{r.Ref, kind, sha})
		}
		return printer.PrintTable([]string{"Ref", "Type", "SHA"}, rows)
	})
}

func runGitTree(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	recursive, _ := cmd.Flags().GetBool("recursive")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tree, resp, err := client.GetTrees(owner, repo, gitea.ListTreeOptions{
		ListOptions: listOptions(cmd), Ref: args[1], Recursive: recursive,
	})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tree", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(tree, func() error {
		if len(tree.Entries) == 0 {
			printer.Println("Empty tree.")
			return nil
		}
		rows := make([][]string, 0, len(tree.Entries))
		for _, e := range tree.Entries {
			rows = append(rows, []string{e.Mode, e.Type, fmt.Sprintf("%d", e.Size), shortSHA(e.SHA), e.Path})
		}
		if err := printer.PrintTable([]string{"Mode", "Type", "Size", "SHA", "Path"}, rows); err != nil {
			return err
		}
		if tree.Truncated {
			printer.Printf("\n%d entries total; this page is truncated by the server.\n", tree.TotalCount)
		}
		return nil
	})
}

func runGitBlob(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	blob, resp, err := client.GetBlob(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "blob", args[1])
	}

	// Gitea returns blobs base64-encoded; decode so the caller sees the file.
	data := []byte(blob.Content)
	if blob.Encoding == "base64" {
		decoded, decodeErr := base64.StdEncoding.DecodeString(blob.Content)
		if decodeErr != nil {
			return errors.NewGeneralError(fmt.Sprintf("failed to decode blob: %v", decodeErr))
		}
		data = decoded
	}

	return writeBlob(cmd, data, blob.SHA)
}

func runGitNote(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	note, resp, err := client.GetRepoNote(owner, repo, args[1], gitea.GetRepoNoteOptions{})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "git note", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(note, func() error {
		printer.Println(strings.TrimRight(note.Message, "\n"))
		return nil
	})
}
