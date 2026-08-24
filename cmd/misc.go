// Package cmd contains packages, reactions, access tokens and server-info
// commands.
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var packageCmd = &cobra.Command{
	Use:     "package",
	Aliases: []string{"packages", "pkg"},
	Short:   "Manage packages in a user or organization registry",
}

var packageListCmd = &cobra.Command{
	Use:   "list <owner>",
	Short: "List an owner's packages",
	Args:  cobra.ExactArgs(1),
	RunE:  runPackageList,
}

var packageGetCmd = &cobra.Command{
	Use:   "get <owner> <type> <name> <version>",
	Short: "Get one package version",
	Args:  cobra.ExactArgs(4),
	RunE:  runPackageGet,
}

var packageLatestCmd = &cobra.Command{
	Use:   "latest <owner> <type> <name>",
	Short: "Get the latest version of a package",
	Args:  cobra.ExactArgs(3),
	RunE:  runPackageLatest,
}

var packageFilesCmd = &cobra.Command{
	Use:   "files <owner> <type> <name> <version>",
	Short: "List the files in a package version",
	Args:  cobra.ExactArgs(4),
	RunE:  runPackageFiles,
}

var packageDeleteCmd = &cobra.Command{
	Use:     "delete <owner> <type> <name> <version>",
	Aliases: []string{"rm"},
	Short:   "Delete a package version",
	Args:    cobra.ExactArgs(4),
	RunE:    runPackageDelete,
}

var packageLinkCmd = &cobra.Command{
	Use:   "link <owner> <type> <name> <repo-name>",
	Short: "Link a package to a repository",
	Args:  cobra.ExactArgs(4),
	RunE:  runPackageLink,
}

var packageUnlinkCmd = &cobra.Command{
	Use:   "unlink <owner> <type> <name>",
	Short: "Unlink a package from its repository",
	Args:  cobra.ExactArgs(3),
	RunE:  runPackageUnlink,
}

var reactionCmd = &cobra.Command{
	Use:     "reaction",
	Aliases: []string{"reactions"},
	Short:   "React to issues, pull requests and comments",
}

var reactionListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List the reactions on an issue or pull request",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runReactionList,
}

var reactionAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> <reaction>",
	Short: "React to an issue or pull request",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReactionAdd,
}

var reactionRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <issue-number> <reaction>",
	Aliases: []string{"rm"},
	Short:   "Remove your reaction from an issue or pull request",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runReactionRemove,
}

var reactionCommentListCmd = &cobra.Command{
	Use:   "comment-list [<owner>/<repo>] <comment-id>",
	Short: "List the reactions on a comment",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runReactionCommentList,
}

var reactionCommentAddCmd = &cobra.Command{
	Use:   "comment-add [<owner>/<repo>] <comment-id> <reaction>",
	Short: "React to a comment",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReactionCommentAdd,
}

var reactionCommentRemoveCmd = &cobra.Command{
	Use:   "comment-remove [<owner>/<repo>] <comment-id> <reaction>",
	Short: "Remove your reaction from a comment",
	Args:  cobra.RangeArgs(2, 3),
	RunE:  runReactionCommentRemove,
}

var tokenCmd = &cobra.Command{
	Use:     "token",
	Aliases: []string{"tokens"},
	Short:   "Manage your API access tokens",
}

var tokenListCmd = &cobra.Command{
	Use:   "list",
	Short: "List your access tokens",
	Args:  cobra.NoArgs,
	RunE:  runTokenList,
}

var tokenCreateCmd = &cobra.Command{
	Use:   "create <name>",
	Short: "Create an access token (the value is shown once, and only once)",
	Args:  cobra.ExactArgs(1),
	RunE:  runTokenCreate,
}

var tokenDeleteCmd = &cobra.Command{
	Use:     "delete <name-or-id>",
	Aliases: []string{"rm"},
	Short:   "Delete an access token",
	Args:    cobra.ExactArgs(1),
	RunE:    runTokenDelete,
}

var versionCmd = &cobra.Command{
	Use:   "server-version",
	Short: "Print the Gitea server's version",
	Args:  cobra.NoArgs,
	RunE:  runServerVersion,
}

var templateCmd = &cobra.Command{
	Use:     "template",
	Aliases: []string{"templates"},
	Short:   "List the gitignore, license and label templates the server offers",
}

var templateGitignoreCmd = &cobra.Command{
	Use:   "gitignore [<name>]",
	Short: "List gitignore templates, or print one",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTemplateGitignore,
}

var templateLicenseCmd = &cobra.Command{
	Use:   "license [<name>]",
	Short: "List license templates, or print one",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTemplateLicense,
}

var templateLabelCmd = &cobra.Command{
	Use:   "label [<name>]",
	Short: "List label templates, or print one",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTemplateLabel,
}

var markdownCmd = &cobra.Command{
	Use:   "markdown",
	Short: "Render markdown through the server",
	Args:  cobra.NoArgs,
	RunE:  runMarkdown,
}

var settingsCmd = &cobra.Command{
	Use:   "settings",
	Short: "Show the server's global settings",
	Args:  cobra.NoArgs,
	RunE:  runSettings,
}

func init() {
	RootCmd.AddCommand(packageCmd, reactionCmd, tokenCmd, versionCmd, templateCmd,
		markdownCmd, settingsCmd)
	packageCmd.AddCommand(packageListCmd, packageGetCmd, packageLatestCmd, packageFilesCmd,
		packageDeleteCmd, packageLinkCmd, packageUnlinkCmd)
	reactionCmd.AddCommand(reactionListCmd, reactionAddCmd, reactionRemoveCmd,
		reactionCommentListCmd, reactionCommentAddCmd, reactionCommentRemoveCmd)
	tokenCmd.AddCommand(tokenListCmd, tokenCreateCmd, tokenDeleteCmd)
	templateCmd.AddCommand(templateGitignoreCmd, templateLicenseCmd, templateLabelCmd)

	addPageFlags(packageListCmd, reactionListCmd, tokenListCmd)

	tokenCreateCmd.Flags().StringSliceP("scope", "s", nil,
		"Token scopes, e.g. read:repository,write:issue (repeatable)")

	markdownCmd.Flags().StringP("text", "t", "", "Markdown to render")
	markdownCmd.Flags().StringP("file", "f", "", "Read the markdown from this file, or - for stdin")
	markdownCmd.Flags().StringP("mode", "m", "", "Render mode: markdown, gfm, or comment")
	markdownCmd.Flags().StringP("context", "c", "", "Repository context for relative links, as owner/repo")
}

func runPackageList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	packages, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Package, *gitea.Response, error) {
		return client.ListPackages(args[0], gitea.ListPackagesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(packages, func() error {
		if len(packages) == 0 {
			printer.Println("No packages found.")
			return nil
		}
		rows := make([][]string, 0, len(packages))
		for _, p := range packages {
			repo := ""
			if p.Repository != nil {
				repo = p.Repository.FullName
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", p.ID), p.Type, p.Name, p.Version, repo, renderValue(p.CreatedAt),
			})
		}
		return printer.PrintTable([]string{"ID", "Type", "Name", "Version", "Repo", "Created"}, rows)
	})
}

func printPackage(cmd *cobra.Command, p *gitea.Package) error {
	printer := getPrinter(cmd)
	return printer.Emit(p, func() error {
		var d detail
		d.always("ID", p.ID)
		d.add("Type", p.Type)
		d.add("Name", p.Name)
		d.add("Version", p.Version)
		d.add("Owner", p.Owner.UserName)
		d.add("Creator", p.Creator.UserName)
		if p.Repository != nil {
			d.add("Repository", p.Repository.FullName)
		}
		d.add("Created", p.CreatedAt)
		return d.print(printer)
	})
}

func runPackageGet(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	pkg, resp, err := client.GetPackage(args[0], args[1], args[2], args[3])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2]+"@"+args[3])
	}
	return printPackage(cmd, pkg)
}

func runPackageLatest(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	pkg, resp, err := client.GetLatestPackage(args[0], args[1], args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2])
	}
	return printPackage(cmd, pkg)
}

func runPackageFiles(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	files, resp, err := client.ListPackageFiles(args[0], args[1], args[2], args[3])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2]+"@"+args[3])
	}

	printer := getPrinter(cmd)
	return printer.Emit(files, func() error {
		if len(files) == 0 {
			printer.Println("No files in this package.")
			return nil
		}
		rows := make([][]string, 0, len(files))
		for _, f := range files {
			rows = append(rows, []string{
				fmt.Sprintf("%d", f.ID), f.Name, fmt.Sprintf("%d", f.Size), shortSHA(f.SHA256),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Size", "SHA256"}, rows)
	})
}

func runPackageDelete(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would delete package %s/%s %s@%s", args[0], args[1], args[2], args[3]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeletePackage(args[0], args[1], args[2], args[3])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2]+"@"+args[3])
	}
	return emitMessage(cmd, okMessage("package deleted"), "Deleted %s@%s", args[2], args[3])
}

func runPackageLink(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would link package %s to %s", args[2], args[3]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.LinkPackage(args[0], args[1], args[2], args[3])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2])
	}
	return emitMessage(cmd, okMessage("package linked"), "Linked %s to %s", args[2], args[3])
}

func runPackageUnlink(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would unlink package %s", args[2]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.UnlinkPackage(args[0], args[1], args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "package", args[2])
	}
	return emitMessage(cmd, okMessage("package unlinked"), "Unlinked %s", args[2])
}

func emitReactions(cmd *cobra.Command, reactions []*gitea.Reaction) error {
	printer := getPrinter(cmd)
	return printer.Emit(reactions, func() error {
		if len(reactions) == 0 {
			printer.Println("No reactions.")
			return nil
		}
		// Group by emoji: who reacted with what is the useful summary, not a
		// row per reaction.
		order := []string{}
		byEmoji := map[string][]string{}
		for _, r := range reactions {
			if _, seen := byEmoji[r.Reaction]; !seen {
				order = append(order, r.Reaction)
			}
			byEmoji[r.Reaction] = append(byEmoji[r.Reaction], userName(r.User))
		}
		rows := make([][]string, 0, len(order))
		for _, e := range order {
			rows = append(rows, []string{
				e, fmt.Sprintf("%d", len(byEmoji[e])), strings.Join(byEmoji[e], ", "),
			})
		}
		return printer.PrintTable([]string{"Reaction", "Count", "Users"}, rows)
	})
}

func runReactionList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	reactions, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Reaction, *gitea.Response, error) {
		return client.ListIssueReactions(owner, repo, index, gitea.ListIssueReactionsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitReactions(cmd, reactions)
}

func runReactionAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would react %s to %s/%s#%d", args[2], owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	reaction, resp, err := client.PostIssueReaction(owner, repo, index, args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	return emitMessage(cmd, reaction, "Reacted %s to %s/%s#%d", args[2], owner, repo, index)
}

func runReactionRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would remove the %s reaction from %s/%s#%d", args[2], owner, repo, index) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteIssueReaction(owner, repo, index, args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "reaction", args[2])
	}
	return emitMessage(cmd, okMessage("reaction removed"), "Removed the %s reaction", args[2])
}

func runReactionCommentList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	reactions, resp, err := client.GetIssueCommentReactions(owner, repo, commentID)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comment", args[1])
	}

	return emitReactions(cmd, reactions)
}

func runReactionCommentAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would react %s to comment %d", args[2], commentID) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	reaction, resp, err := client.PostIssueCommentReaction(owner, repo, commentID, args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "comment", args[1])
	}
	return emitMessage(cmd, reaction, "Reacted %s to comment %d", args[2], commentID)
}

func runReactionCommentRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 2)
	if err != nil {
		return err
	}
	commentID, err := int64Arg(args[1], "comment id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would remove the %s reaction from comment %d", args[2], commentID) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteIssueCommentReaction(owner, repo, commentID, args[2])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "reaction", args[2])
	}
	return emitMessage(cmd, okMessage("reaction removed"), "Removed the %s reaction", args[2])
}

func runTokenList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tokens, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.AccessToken, *gitea.Response, error) {
		return client.ListAccessTokens(gitea.ListAccessTokensOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(tokens, func() error {
		if len(tokens) == 0 {
			printer.Println("No access tokens.")
			return nil
		}
		rows := make([][]string, 0, len(tokens))
		for _, t := range tokens {
			scopes := make([]string, 0, len(t.Scopes))
			for _, s := range t.Scopes {
				scopes = append(scopes, string(s))
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", t.ID), t.Name, "..." + t.TokenLastEight,
				cell(strings.Join(scopes, ","), 40), renderValue(t.Created),
			})
		}
		return printer.PrintTable([]string{"ID", "Name", "Token", "Scopes", "Created"}, rows)
	})
}

func runTokenCreate(cmd *cobra.Command, args []string) error {
	scopeValues, _ := cmd.Flags().GetStringSlice("scope")
	scopes := make([]gitea.AccessTokenScope, 0, len(scopeValues))
	for _, s := range scopeValues {
		scopes = append(scopes, gitea.AccessTokenScope(s))
	}

	if dryRunf(cmd, "Would create access token %q", args[0]) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	token, resp, err := client.CreateAccessToken(gitea.CreateAccessTokenOption{
		Name: args[0], Scopes: scopes,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(token, func() error {
		// The server returns the token value only on creation, so print it
		// plainly rather than masking it as the list command does.
		printer.Printf("%s\n", token.Token)
		return nil
	})
}

func runTokenDelete(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would delete access token %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// The endpoint accepts either a numeric id or the token's name.
	var target any = args[0]
	if id, err := int64Arg(args[0], "token id"); err == nil {
		target = id
	}

	resp, err := client.DeleteAccessToken(target)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "access token", args[0])
	}

	return emitMessage(cmd, okMessage("token deleted"), "Deleted access token %s", args[0])
}

func runServerVersion(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	version, resp, err := client.ServerVersion()
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]string{"version": version}, func() error {
		printer.Println(version)
		return nil
	})
}

func runTemplateGitignore(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		info, resp, err := client.GetGitignoreTemplateInfo(args[0])
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "gitignore template", args[0])
		}
		printer := getPrinter(cmd)
		return printer.Emit(info, func() error {
			printer.Println(info.Source)
			return nil
		})
	}

	names, resp, err := client.ListGitignoresTemplates()
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return emitStringList(cmd, names, "No gitignore templates.")
}

func runTemplateLicense(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		info, resp, err := client.GetLicenseTemplateInfo(args[0])
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "license template", args[0])
		}
		printer := getPrinter(cmd)
		return printer.Emit(info, func() error {
			printer.Println(info.Body)
			return nil
		})
	}

	licenses, resp, err := client.ListLicenseTemplates()
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(licenses, func() error {
		if len(licenses) == 0 {
			printer.Println("No license templates.")
			return nil
		}
		rows := make([][]string, 0, len(licenses))
		for _, l := range licenses {
			rows = append(rows, []string{l.Key, l.Name, l.URL})
		}
		return printer.PrintTable([]string{"Key", "Name", "URL"}, rows)
	})
}

func runTemplateLabel(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if len(args) == 1 {
		labels, resp, err := client.GetLabelTemplate(args[0])
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "label template", args[0])
		}
		printer := getPrinter(cmd)
		return printer.Emit(labels, func() error {
			rows := make([][]string, 0, len(labels))
			for _, l := range labels {
				rows = append(rows, []string{l.Name, l.Color, yesNo(l.Exclusive), l.Description})
			}
			return printer.PrintTable([]string{"Name", "Color", "Exclusive", "Description"}, rows)
		})
	}

	names, resp, err := client.ListLabelTemplates()
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return emitStringList(cmd, names, "No label templates.")
}

// emitStringList renders a plain list of names, one per line in text mode.
func emitStringList(cmd *cobra.Command, values []string, empty string) error {
	printer := getPrinter(cmd)
	return printer.Emit(values, func() error {
		if len(values) == 0 {
			printer.Println(empty)
			return nil
		}
		printer.Println(strings.Join(values, "\n"))
		return nil
	})
}

func runMarkdown(cmd *cobra.Command, args []string) error {
	text, err := bodyFromFlags(cmd, "text", "file")
	if err != nil {
		return err
	}
	if text == "" {
		return errors.NewValidationError("markdown required",
			map[string]interface{}{"hint": "pass --text <markdown> or --file <path>"})
	}
	mode, _ := cmd.Flags().GetString("mode")
	context, _ := cmd.Flags().GetString("context")

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	html, resp, err := client.RenderMarkdown(gitea.MarkdownOption{
		Text: text, Mode: mode, Context: context,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	printer := getPrinter(cmd)
	return printer.Emit(map[string]string{"html": string(html)}, func() error {
		printer.Println(strings.TrimRight(string(html), "\n"))
		return nil
	})
}

func runSettings(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	ui, _, err := client.GetGlobalUISettings()
	if err != nil {
		return errors.NewGeneralError(err.Error())
	}
	repoSettings, _, err := client.GetGlobalRepoSettings()
	if err != nil {
		return errors.NewGeneralError(err.Error())
	}
	api, _, err := client.GetGlobalAPISettings()
	if err != nil {
		return errors.NewGeneralError(err.Error())
	}
	attachment, _, err := client.GetGlobalAttachmentSettings()
	if err != nil {
		return errors.NewGeneralError(err.Error())
	}

	payload := map[string]any{
		"ui":         ui,
		"repository": repoSettings,
		"api":        api,
		"attachment": attachment,
	}

	printer := getPrinter(cmd)
	return printer.Emit(payload, func() error {
		var d detail
		d.always("Default theme", ui.DefaultTheme)
		d.always("Max response items", api.MaxResponseItems)
		d.always("Default paging num", api.DefaultPagingNum)
		d.always("Default git trees per page", api.DefaultGitTreesPerPage)
		d.always("Max blob size", api.DefaultMaxBlobSize)
		d.always("Mirrors disabled", repoSettings.MirrorsDisabled)
		d.always("HTTP git disabled", repoSettings.HTTPGitDisabled)
		d.always("Migrations disabled", repoSettings.MigrationsDisabled)
		d.always("Attachments enabled", attachment.Enabled)
		d.always("Attachment max size (MB)", attachment.MaxSize)
		d.add("Allowed attachment types", attachment.AllowedTypes)
		return d.print(printer)
	})
}
