// Package cmd contains repository wiki commands.
package cmd

import (
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var wikiCmd = &cobra.Command{
	Use:   "wiki",
	Short: "Manage a repository's wiki pages",
}

var wikiListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List wiki pages",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWikiList,
}

var wikiGetCmd = &cobra.Command{
	Use:     "get [<owner>/<repo>] <page>",
	Aliases: []string{"view", "read"},
	Short:   "Print the content of a wiki page",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runWikiGet,
}

var wikiCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <page>",
	Short: "Create a wiki page",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runWikiCreate,
}

var wikiUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <page>",
	Aliases: []string{"edit"},
	Short:   "Update a wiki page",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runWikiUpdate,
}

var wikiDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <page>",
	Aliases: []string{"rm"},
	Short:   "Delete a wiki page",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runWikiDelete,
}

var wikiRevisionsCmd = &cobra.Command{
	Use:     "revisions [<owner>/<repo>] <page>",
	Aliases: []string{"history"},
	Short:   "List the revisions of a wiki page",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runWikiRevisions,
}

func init() {
	RootCmd.AddCommand(wikiCmd)
	wikiCmd.AddCommand(wikiListCmd, wikiGetCmd, wikiCreateCmd, wikiUpdateCmd, wikiDeleteCmd, wikiRevisionsCmd)

	addPageFlags(wikiListCmd)
	wikiRevisionsCmd.Flags().Int("page", 0, "Page number")

	for _, c := range []*cobra.Command{wikiCreateCmd, wikiUpdateCmd} {
		c.Flags().StringP("content", "c", "", "Page content")
		c.Flags().StringP("file", "f", "", "Read the page content from this file, or - for stdin")
		c.Flags().StringP("message", "m", "", "Commit message")
		c.Flags().StringP("title", "t", "", "New page title (defaults to the page argument)")
	}
}

// wikiContent resolves the page body from --content or --file. Wiki pages are
// carried as base64 over the API, so the caller never handles the encoding.
func wikiContent(cmd *cobra.Command) (string, error) {
	content, err := bodyFromFlags(cmd, "content", "file")
	if err != nil {
		return "", err
	}
	if content == "" {
		return "", errors.NewValidationError("page content required",
			map[string]interface{}{"hint": "pass --content <text> or --file <path>"})
	}
	return base64.StdEncoding.EncodeToString([]byte(content)), nil
}

func runWikiList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	pages, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.WikiPageMetaData, *gitea.Response, error) {
		return client.ListWikiPages(owner, repo, gitea.ListWikiPagesOptions{ListOptions: lo})
	})
	if err != nil {
		// Gitea answers 404 both for a missing repo and for a repo whose wiki
		// is disabled, so say which one the caller is looking at.
		var cliErr *errors.CLIError
		if stderrors.As(err, &cliErr) && cliErr.Code == errors.ExitNotFound {
			return errors.NewNotFoundError("wiki", args[0])
		}
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(pages, func() error {
		if len(pages) == 0 {
			printer.Println("No wiki pages found.")
			return nil
		}
		rows := make([][]string, 0, len(pages))
		for _, p := range pages {
			last := ""
			if p.LastCommit != nil {
				last = strings.SplitN(strings.TrimSpace(p.LastCommit.Message), "\n", 2)[0]
			}
			rows = append(rows, []string{p.Title, p.SubURL, last})
		}
		return printer.PrintTable([]string{"Title", "Slug", "Last commit"}, rows)
	})
}

func runWikiGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	page := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	got, resp, err := client.GetWikiPage(owner, repo, page)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "wiki page", page)
	}

	decoded, decodeErr := base64.StdEncoding.DecodeString(got.ContentBase64)
	if decodeErr != nil {
		return errors.NewGeneralError(fmt.Sprintf("failed to decode page content: %v", decodeErr))
	}

	printer := getPrinter(cmd)
	// JSON callers get the page object with the content decoded in place, so
	// they do not have to base64-decode it themselves.
	payload := struct {
		*gitea.WikiPage
		Content string `json:"content"`
	}{WikiPage: got, Content: string(decoded)}

	return printer.Emit(payload, func() error {
		printer.Println(string(decoded))
		return nil
	})
}

func runWikiCreate(cmd *cobra.Command, args []string) error {
	return writeWikiPage(cmd, args, false)
}

func runWikiUpdate(cmd *cobra.Command, args []string) error {
	return writeWikiPage(cmd, args, true)
}

func writeWikiPage(cmd *cobra.Command, args []string, update bool) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	page := args[1]

	content, err := wikiContent(cmd)
	if err != nil {
		return err
	}
	message, _ := cmd.Flags().GetString("message")
	title, _ := cmd.Flags().GetString("title")
	if title == "" {
		title = page
	}

	verb := "create"
	if update {
		verb = "update"
	}
	if dryRunf(cmd, "Would %s wiki page %s in %s/%s", verb, page, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateWikiPageOptions{Title: title, ContentBase64: content, Message: message}

	var (
		result *gitea.WikiPage
		resp   *gitea.Response
	)
	if update {
		result, resp, err = client.EditWikiPage(owner, repo, page, opts)
	} else {
		result, resp, err = client.CreateWikiPage(owner, repo, opts)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "wiki page", page)
	}

	printer := getPrinter(cmd)
	return printer.Emit(result, func() error {
		printer.Printf("%sd wiki page %s: %s\n", strings.ToUpper(verb[:1])+verb[1:], result.Title, result.HTMLURL)
		return nil
	})
}

func runWikiDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	page := args[1]

	if dryRunf(cmd, "Would delete wiki page %s from %s/%s", page, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	resp, err := client.DeleteWikiPage(owner, repo, page)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "wiki page", page)
	}

	return emitMessage(cmd, okMessage("wiki page deleted"), "Deleted wiki page %s from %s/%s", page, owner, repo)
}

func runWikiRevisions(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	page := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// This endpoint takes a page number only: it has no per-page control, so
	// --per-page and --all do not apply here.
	list, resp, err := client.GetWikiPageRevisions(owner, repo, page, gitea.ListWikiPageRevisionsOptions{
		Page: listOptions(cmd).Page,
	})
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "wiki page", page)
	}

	printer := getPrinter(cmd)
	return printer.Emit(list.WikiCommits, func() error {
		if len(list.WikiCommits) == 0 {
			printer.Println("No revisions found.")
			return nil
		}
		rows := make([][]string, 0, len(list.WikiCommits))
		for _, c := range list.WikiCommits {
			author := ""
			if c.Author != nil {
				author = c.Author.Name
			}
			subject := strings.SplitN(strings.TrimSpace(c.Message), "\n", 2)[0]
			rows = append(rows, []string{shortSHA(c.ID), author, subject})
		}
		return printer.PrintTable([]string{"SHA", "Author", "Message"}, rows)
	})
}

// shortSHA abbreviates a commit hash for table output.
func shortSHA(sha string) string {
	if len(sha) > 10 {
		return sha[:10]
	}
	return sha
}
