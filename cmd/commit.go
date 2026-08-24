// Package cmd contains commit, tag, comparison and commit status commands.
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:     "commit",
	Aliases: []string{"commits"},
	Short:   "Inspect commits, diffs and commit statuses",
}

var commitListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List commits",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCommitList,
}

var commitGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <sha>",
	Short: "Get one commit",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runCommitGet,
}

var commitDiffCmd = &cobra.Command{
	Use:   "diff [<owner>/<repo>] <sha>",
	Short: "Print a commit's diff",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runCommitDiff,
}

var commitPatchCmd = &cobra.Command{
	Use:   "patch [<owner>/<repo>] <sha>",
	Short: "Print a commit as a git patch",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runCommitPatch,
}

var commitCompareCmd = &cobra.Command{
	Use:   "compare [<owner>/<repo>] <base> <head>",
	Short: "List the commits between two refs",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runCommitCompare,
}

var statusCmd = &cobra.Command{
	Use:     "status",
	Aliases: []string{"statuses"},
	Short:   "Read and set commit statuses",
}

var statusListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <ref>",
	Short: "List the statuses posted against a ref",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStatusList,
}

var statusCombinedCmd = &cobra.Command{
	Use:   "combined [<owner>/<repo>] <ref>",
	Short: "Show the combined status of a ref",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runStatusCombined,
}

var statusCreateCmd = &cobra.Command{
	Use:     "create [<owner>/<repo>] <sha>",
	Aliases: []string{"set"},
	Short:   "Post a status against a commit",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runStatusCreate,
}

var tagCmd = &cobra.Command{
	Use:     "tag",
	Aliases: []string{"tags"},
	Short:   "Manage git tags",
}

var tagListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List tags",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTagList,
}

var tagGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <tag>",
	Short: "Get one tag",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runTagGet,
}

var tagCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <tag>",
	Short: "Create a tag",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runTagCreate,
}

var tagDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <tag>",
	Aliases: []string{"rm"},
	Short:   "Delete a tag",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runTagDelete,
}

func init() {
	RootCmd.AddCommand(commitCmd, statusCmd, tagCmd)
	commitCmd.AddCommand(commitListCmd, commitGetCmd, commitDiffCmd, commitPatchCmd, commitCompareCmd)
	statusCmd.AddCommand(statusListCmd, statusCombinedCmd, statusCreateCmd)
	tagCmd.AddCommand(tagListCmd, tagGetCmd, tagCreateCmd, tagDeleteCmd)

	addPageFlags(commitListCmd, statusListCmd, tagListCmd)

	commitListCmd.Flags().String("sha", "", "Start from this branch, tag or commit")
	commitListCmd.Flags().String("path", "", "Only commits touching this path")
	commitListCmd.Flags().String("not", "", "Exclude commits reachable from this ref")
	commitListCmd.Flags().Bool("stat", false, "Include diff stats for each commit")
	commitListCmd.Flags().Bool("files", false, "Include the affected file list for each commit")

	statusCreateCmd.Flags().StringP("state", "s", "",
		"Status state: pending, success, error, failure, warning (required)")
	statusCreateCmd.MarkFlagRequired("state")
	statusCreateCmd.Flags().StringP("context", "c", "", "Status context, e.g. ci/build")
	statusCreateCmd.Flags().StringP("description", "d", "", "Short status description")
	statusCreateCmd.Flags().StringP("target-url", "u", "", "URL with the full details")

	tagCreateCmd.Flags().StringP("message", "m", "", "Annotation message")
	tagCreateCmd.Flags().String("target", "", "Commit, branch or tag to point at (defaults to the default branch)")
}

// commitSubject is the first line of a commit message, which is all a table row
// has room for.
func commitSubject(c *gitea.Commit) string {
	if c == nil || c.RepoCommit == nil {
		return ""
	}
	return strings.SplitN(strings.TrimSpace(c.RepoCommit.Message), "\n", 2)[0]
}

func emitCommitTable(cmd *cobra.Command, commits []*gitea.Commit) error {
	printer := getPrinter(cmd)
	return printer.Emit(commits, func() error {
		if len(commits) == 0 {
			printer.Println("No commits found.")
			return nil
		}
		rows := make([][]string, 0, len(commits))
		for _, c := range commits {
			sha, when := "", ""
			if c.CommitMeta != nil {
				sha = shortSHA(c.SHA)
				when = renderValue(c.Created)
			}
			author := userName(c.Author)
			if author == "" && c.RepoCommit != nil && c.RepoCommit.Author != nil {
				author = c.RepoCommit.Author.Name
			}
			rows = append(rows, []string{sha, author, when, cell(commitSubject(c), 60)})
		}
		return printer.PrintTable([]string{"SHA", "Author", "Date", "Message"}, rows)
	})
}

func runCommitList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	sha, _ := cmd.Flags().GetString("sha")
	path, _ := cmd.Flags().GetString("path")
	not, _ := cmd.Flags().GetString("not")
	stat, _ := cmd.Flags().GetBool("stat")
	files, _ := cmd.Flags().GetBool("files")

	commits, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Commit, *gitea.Response, error) {
		return client.ListRepoCommits(owner, repo, gitea.ListCommitOptions{
			ListOptions: lo, SHA: sha, Path: path, Not: not, Stat: stat, Files: files,
		})
	})
	if err != nil {
		return err
	}

	return emitCommitTable(cmd, commits)
}

func runCommitGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	commit, resp, err := client.GetSingleCommit(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "commit", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(commit, func() error {
		var d detail
		if commit.CommitMeta != nil {
			d.add("SHA", commit.SHA)
			d.add("Date", commit.Created)
		}
		if commit.RepoCommit != nil {
			if a := commit.RepoCommit.Author; a != nil {
				d.add("Author", fmt.Sprintf("%s <%s>", a.Name, a.Email))
			}
			if c := commit.RepoCommit.Committer; c != nil {
				d.add("Committer", fmt.Sprintf("%s <%s>", c.Name, c.Email))
			}
		}
		parents := make([]string, 0, len(commit.Parents))
		for _, p := range commit.Parents {
			if p != nil {
				parents = append(parents, shortSHA(p.SHA))
			}
		}
		d.add("Parents", parents)
		if commit.Stats != nil {
			d.always("Changes", fmt.Sprintf("+%d -%d (%d total)",
				commit.Stats.Additions, commit.Stats.Deletions, commit.Stats.Total))
		}
		if len(commit.Files) > 0 {
			names := make([]string, 0, len(commit.Files))
			for _, f := range commit.Files {
				if f != nil {
					names = append(names, f.Filename)
				}
			}
			d.add("Files", strings.Join(names, "\n"))
		}
		d.add("URL", commit.HTMLURL)
		if commit.RepoCommit != nil {
			d.add("Message", strings.TrimSpace(commit.RepoCommit.Message))
		}
		return d.print(printer)
	})
}

func runCommitDiff(cmd *cobra.Command, args []string) error {
	return printCommitBlob(cmd, args, false)
}

func runCommitPatch(cmd *cobra.Command, args []string) error {
	return printCommitBlob(cmd, args, true)
}

func printCommitBlob(cmd *cobra.Command, args []string, patch bool) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		data []byte
		resp *gitea.Response
	)
	if patch {
		data, resp, err = client.GetCommitPatch(owner, repo, args[1])
	} else {
		data, resp, err = client.GetCommitDiff(owner, repo, args[1])
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "commit", args[1])
	}

	printer := getPrinter(cmd)
	// A diff is already text; under --format json wrap it so the output stays
	// parseable rather than emitting a bare blob.
	return printer.Emit(map[string]string{"sha": args[1], "diff": string(data)}, func() error {
		printer.Println(strings.TrimRight(string(data), "\n"))
		return nil
	})
}

func runCommitCompare(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	compare, resp, err := client.CompareCommits(owner, repo, args[1], args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comparison", args[1]+"..."+args[2])
	}

	return emitCommitTable(cmd, compare.Commits)
}

// parseStatusState validates a --state value. Gitea rejects unknown states with
// a bare 422, so name the accepted set here.
func parseStatusState(s string) (gitea.StatusState, error) {
	switch strings.ToLower(s) {
	case "pending":
		return gitea.StatusPending, nil
	case "success":
		return gitea.StatusSuccess, nil
	case "error":
		return gitea.StatusError, nil
	case "failure":
		return gitea.StatusFailure, nil
	case "warning":
		return gitea.StatusWarning, nil
	default:
		return "", errors.NewValidationError(fmt.Sprintf("invalid state: %s", s),
			map[string]interface{}{"expected": "pending, success, error, failure, or warning"})
	}
}

func emitStatuses(cmd *cobra.Command, statuses []*gitea.Status) error {
	printer := getPrinter(cmd)
	return printer.Emit(statuses, func() error {
		if len(statuses) == 0 {
			printer.Println("No statuses found.")
			return nil
		}
		rows := make([][]string, 0, len(statuses))
		for _, s := range statuses {
			rows = append(rows, []string{
				fmt.Sprintf("%d", s.ID), string(s.State), s.Context,
				cell(s.Description, 40), s.TargetURL,
			})
		}
		return printer.PrintTable([]string{"ID", "State", "Context", "Description", "URL"}, rows)
	})
}

func runStatusList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	statuses, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Status, *gitea.Response, error) {
		return client.ListStatuses(owner, repo, args[1], gitea.ListStatusesOption{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitStatuses(cmd, statuses)
}

func runStatusCombined(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	combined, resp, err := client.GetCombinedStatus(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "ref", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(combined, func() error {
		printer.Printf("State: %s (%d status(es)) at %s\n\n",
			combined.State, combined.TotalCount, shortSHA(combined.SHA))
		if len(combined.Statuses) == 0 {
			printer.Println("No statuses posted.")
			return nil
		}
		rows := make([][]string, 0, len(combined.Statuses))
		for _, s := range combined.Statuses {
			rows = append(rows, []string{string(s.State), s.Context, cell(s.Description, 50)})
		}
		return printer.PrintTable([]string{"State", "Context", "Description"}, rows)
	})
}

func runStatusCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	sha := args[1]

	stateFlag, _ := cmd.Flags().GetString("state")
	state, err := parseStatusState(stateFlag)
	if err != nil {
		return err
	}
	context, _ := cmd.Flags().GetString("context")
	description, _ := cmd.Flags().GetString("description")
	targetURL, _ := cmd.Flags().GetString("target-url")

	if dryRunf(cmd, "Would post a %s status on %s/%s@%s", state, owner, repo, shortSHA(sha)) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	status, resp, err := client.CreateStatus(owner, repo, sha, gitea.CreateStatusOption{
		State: state, Context: context, Description: description, TargetURL: targetURL,
	})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "commit", sha)
	}

	return emitMessage(cmd, status, "Posted %s status %q on %s", status.State, status.Context, shortSHA(sha))
}

func runTagList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tags, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Tag, *gitea.Response, error) {
		return client.ListRepoTags(owner, repo, gitea.ListRepoTagsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(tags, func() error {
		if len(tags) == 0 {
			printer.Println("No tags found.")
			return nil
		}
		rows := make([][]string, 0, len(tags))
		for _, t := range tags {
			sha := ""
			if t.Commit != nil {
				sha = shortSHA(t.Commit.SHA)
			}
			rows = append(rows, []string{t.Name, sha, cell(strings.TrimSpace(t.Message), 50)})
		}
		return printer.PrintTable([]string{"Name", "Commit", "Message"}, rows)
	})
}

func runTagGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tag, resp, err := client.GetTag(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tag", args[1])
	}

	printer := getPrinter(cmd)
	return printer.Emit(tag, func() error {
		var d detail
		d.add("Name", tag.Name)
		d.add("ID", tag.ID)
		if tag.Commit != nil {
			d.add("Commit", tag.Commit.SHA)
			d.add("Commit URL", tag.Commit.URL)
		}
		d.add("Tarball", tag.TarballURL)
		d.add("Zipball", tag.ZipballURL)
		d.add("Message", strings.TrimSpace(tag.Message))
		return d.print(printer)
	})
}

func runTagCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	name := args[1]

	message, _ := cmd.Flags().GetString("message")
	target, _ := cmd.Flags().GetString("target")

	if dryRunf(cmd, "Would create tag %s in %s/%s", name, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tag, resp, err := client.CreateTag(owner, repo, gitea.CreateTagOption{
		TagName: name, Message: message, Target: target,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, tag, "Created tag %s in %s/%s", tag.Name, owner, repo)
}

func runTagDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	name := args[1]

	if dryRunf(cmd, "Would delete tag %s from %s/%s", name, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteTag(owner, repo, name)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tag", name)
	}

	return emitMessage(cmd, okMessage("tag deleted"), "Deleted tag %s from %s/%s", name, owner, repo)
}
