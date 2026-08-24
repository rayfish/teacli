// Package cmd contains branch and tag protection commands.
package cmd

import (
	"fmt"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var protectionCmd = &cobra.Command{
	Use:     "protection",
	Aliases: []string{"protect"},
	Short:   "Manage branch and tag protection rules",
}

var branchProtectionCmd = &cobra.Command{
	Use:     "branch",
	Aliases: []string{"branches"},
	Short:   "Manage branch protection rules",
}

var branchProtectionListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List branch protection rules",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runBranchProtectionList,
}

var branchProtectionGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <rule-name>",
	Short: "Get one branch protection rule",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranchProtectionGet,
}

var branchProtectionCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <branch-or-pattern>",
	Short: "Create a branch protection rule",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runBranchProtectionCreate,
}

var branchProtectionUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <rule-name>",
	Aliases: []string{"edit"},
	Short:   "Edit a branch protection rule",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runBranchProtectionUpdate,
}

var branchProtectionDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <rule-name>",
	Aliases: []string{"rm"},
	Short:   "Delete a branch protection rule",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runBranchProtectionDelete,
}

var tagProtectionCmd = &cobra.Command{
	Use:     "tag",
	Aliases: []string{"tags"},
	Short:   "Manage tag protection rules",
}

var tagProtectionListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List tag protection rules",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTagProtectionList,
}

var tagProtectionGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <id>",
	Short: "Get one tag protection rule",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runTagProtectionGet,
}

var tagProtectionCreateCmd = &cobra.Command{
	Use:   "create [<owner>/<repo>] <name-pattern>",
	Short: "Create a tag protection rule",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runTagProtectionCreate,
}

var tagProtectionUpdateCmd = &cobra.Command{
	Use:     "update [<owner>/<repo>] <id>",
	Aliases: []string{"edit"},
	Short:   "Edit a tag protection rule",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runTagProtectionUpdate,
}

var tagProtectionDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <id>",
	Aliases: []string{"rm"},
	Short:   "Delete a tag protection rule",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runTagProtectionDelete,
}

func init() {
	RootCmd.AddCommand(protectionCmd)
	protectionCmd.AddCommand(branchProtectionCmd, tagProtectionCmd)
	branchProtectionCmd.AddCommand(branchProtectionListCmd, branchProtectionGetCmd,
		branchProtectionCreateCmd, branchProtectionUpdateCmd, branchProtectionDeleteCmd)
	tagProtectionCmd.AddCommand(tagProtectionListCmd, tagProtectionGetCmd,
		tagProtectionCreateCmd, tagProtectionUpdateCmd, tagProtectionDeleteCmd)

	addPageFlags(branchProtectionListCmd, tagProtectionListCmd)

	for _, c := range []*cobra.Command{branchProtectionCreateCmd, branchProtectionUpdateCmd} {
		c.Flags().Int64("required-approvals", 0, "Approvals a pull request needs before it can merge")
		c.Flags().StringSlice("push-whitelist-users", nil, "Users allowed to push directly")
		c.Flags().StringSlice("push-whitelist-teams", nil, "Teams allowed to push directly")
		c.Flags().StringSlice("merge-whitelist-users", nil, "Users allowed to merge")
		c.Flags().StringSlice("merge-whitelist-teams", nil, "Teams allowed to merge")
		c.Flags().StringSlice("approvals-whitelist-users", nil, "Users whose approvals count")
		c.Flags().StringSlice("approvals-whitelist-teams", nil, "Teams whose approvals count")
		c.Flags().StringSlice("status-checks", nil, "Status check contexts that must pass")
		c.Flags().Bool("enable-push", false, "Allow direct pushes to the branch")
		c.Flags().Bool("no-push", false, "Block direct pushes to the branch")
		c.Flags().Bool("block-rejected-reviews", false, "Block merging while a review requests changes")
		c.Flags().Bool("no-block-rejected-reviews", false, "Allow merging despite a rejected review")
		c.Flags().Bool("block-outdated", false, "Block merging when the branch is behind its base")
		c.Flags().Bool("no-block-outdated", false, "Allow merging an outdated branch")
		c.Flags().Bool("dismiss-stale-approvals", false, "Dismiss approvals when new commits arrive")
		c.Flags().Bool("no-dismiss-stale-approvals", false, "Keep approvals when new commits arrive")
		c.Flags().Bool("require-signed-commits", false, "Require signed commits")
		c.Flags().Bool("no-require-signed-commits", false, "Do not require signed commits")
		c.Flags().String("protected-files", "", "Glob patterns of files only whitelisted users may change")
		c.Flags().String("unprotected-files", "", "Glob patterns exempt from the file protection")
		c.Flags().String("rule-name", "", "Rule name, when it differs from the branch pattern")
	}

	for _, c := range []*cobra.Command{tagProtectionCreateCmd, tagProtectionUpdateCmd} {
		c.Flags().StringSlice("whitelist-users", nil, "Users allowed to create matching tags")
		c.Flags().StringSlice("whitelist-teams", nil, "Teams allowed to create matching tags")
	}
	tagProtectionUpdateCmd.Flags().String("pattern", "", "New name pattern")
}

func printBranchProtection(cmd *cobra.Command, bp *gitea.BranchProtection) error {
	printer := getPrinter(cmd)
	return printer.Emit(bp, func() error {
		var d detail
		d.add("Rule", bp.RuleName)
		d.add("Branch", bp.BranchName)
		d.always("Direct push", bp.EnablePush)
		d.add("Push whitelist users", bp.PushWhitelistUsernames)
		d.add("Push whitelist teams", bp.PushWhitelistTeams)
		d.always("Merge whitelist enabled", bp.EnableMergeWhitelist)
		d.add("Merge whitelist users", bp.MergeWhitelistUsernames)
		d.add("Merge whitelist teams", bp.MergeWhitelistTeams)
		d.always("Required approvals", bp.RequiredApprovals)
		d.add("Approvals whitelist users", bp.ApprovalsWhitelistUsernames)
		d.add("Approvals whitelist teams", bp.ApprovalsWhitelistTeams)
		d.always("Status checks", bp.EnableStatusCheck)
		d.add("Status contexts", bp.StatusCheckContexts)
		d.always("Block rejected reviews", bp.BlockOnRejectedReviews)
		d.always("Block official review requests", bp.BlockOnOfficialReviewRequests)
		d.always("Block outdated branch", bp.BlockOnOutdatedBranch)
		d.always("Dismiss stale approvals", bp.DismissStaleApprovals)
		d.always("Require signed commits", bp.RequireSignedCommits)
		d.add("Protected files", bp.ProtectedFilePatterns)
		d.add("Unprotected files", bp.UnprotectedFilePatterns)
		d.add("Created", bp.Created)
		d.add("Updated", bp.Updated)
		return d.print(printer)
	})
}

func runBranchProtectionList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	rules, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.BranchProtection, *gitea.Response, error) {
		return client.ListBranchProtections(owner, repo, gitea.ListBranchProtectionsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(rules, func() error {
		if len(rules) == 0 {
			printer.Println("No branch protection rules found.")
			return nil
		}
		rows := make([][]string, 0, len(rules))
		for _, r := range rules {
			rows = append(rows, []string{
				r.RuleName, r.BranchName, fmt.Sprintf("%d", r.RequiredApprovals),
				yesNo(r.EnablePush), yesNo(r.EnableStatusCheck), yesNo(r.RequireSignedCommits),
			})
		}
		return printer.PrintTable(
			[]string{"Rule", "Branch", "Approvals", "Push", "Status checks", "Signed"}, rows)
	})
}

func runBranchProtectionGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	bp, resp, err := client.GetBranchProtection(owner, repo, args[1])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "branch protection rule", args[1])
	}
	return printBranchProtection(cmd, bp)
}

func runBranchProtectionCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	pattern := args[1]

	ruleName, _ := cmd.Flags().GetString("rule-name")
	if ruleName == "" {
		ruleName = pattern
	}

	slice := func(n string) []string { v, _ := cmd.Flags().GetStringSlice(n); return v }
	approvals, _ := cmd.Flags().GetInt64("required-approvals")
	protectedFiles, _ := cmd.Flags().GetString("protected-files")
	unprotectedFiles, _ := cmd.Flags().GetString("unprotected-files")

	opts := gitea.CreateBranchProtectionOption{
		BranchName:                    pattern,
		RuleName:                      ruleName,
		EnablePush:                    boolPairValue(cmd, "enable-push", "no-push", false),
		PushWhitelistUsernames:        slice("push-whitelist-users"),
		PushWhitelistTeams:            slice("push-whitelist-teams"),
		MergeWhitelistUsernames:       slice("merge-whitelist-users"),
		MergeWhitelistTeams:           slice("merge-whitelist-teams"),
		StatusCheckContexts:           slice("status-checks"),
		RequiredApprovals:             approvals,
		ApprovalsWhitelistUsernames:   slice("approvals-whitelist-users"),
		ApprovalsWhitelistTeams:       slice("approvals-whitelist-teams"),
		BlockOnRejectedReviews:        boolPairValue(cmd, "block-rejected-reviews", "no-block-rejected-reviews", false),
		BlockOnOfficialReviewRequests: false,
		BlockOnOutdatedBranch:         boolPairValue(cmd, "block-outdated", "no-block-outdated", false),
		DismissStaleApprovals:         boolPairValue(cmd, "dismiss-stale-approvals", "no-dismiss-stale-approvals", false),
		RequireSignedCommits:          boolPairValue(cmd, "require-signed-commits", "no-require-signed-commits", false),
		ProtectedFilePatterns:         protectedFiles,
		UnprotectedFilePatterns:       unprotectedFiles,
	}
	// The whitelists only take effect when their enable flag is set too, which
	// the API models separately.
	opts.EnablePushWhitelist = len(opts.PushWhitelistUsernames) > 0 || len(opts.PushWhitelistTeams) > 0
	opts.EnableMergeWhitelist = len(opts.MergeWhitelistUsernames) > 0 || len(opts.MergeWhitelistTeams) > 0
	opts.EnableApprovalsWhitelist = len(opts.ApprovalsWhitelistUsernames) > 0 || len(opts.ApprovalsWhitelistTeams) > 0
	opts.EnableStatusCheck = len(opts.StatusCheckContexts) > 0

	if dryRunf(cmd, "Would protect %s in %s/%s", pattern, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	bp, resp, err := client.CreateBranchProtection(owner, repo, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return printBranchProtection(cmd, bp)
}

// boolPairValue reads a --x/--no-x pair into a plain bool, falling back to def
// when neither was given.
func boolPairValue(cmd *cobra.Command, on, off string, def bool) bool {
	if v := boolPairFlag(cmd, on, off); v != nil {
		return *v
	}
	return def
}

// sliceFlagPtr returns the values of a repeatable flag only when it was given,
// so an untouched whitelist is left alone rather than cleared.
func sliceFlagPtr(cmd *cobra.Command, name string) []string {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	v, _ := cmd.Flags().GetStringSlice(name)
	return v
}

func int64FlagPtr(cmd *cobra.Command, name string) *int64 {
	if !cmd.Flags().Changed(name) {
		return nil
	}
	v, _ := cmd.Flags().GetInt64(name)
	return &v
}

func runBranchProtectionUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	rule := args[1]

	opts := gitea.EditBranchProtectionOption{
		EnablePush:                  boolPairFlag(cmd, "enable-push", "no-push"),
		PushWhitelistUsernames:      sliceFlagPtr(cmd, "push-whitelist-users"),
		PushWhitelistTeams:          sliceFlagPtr(cmd, "push-whitelist-teams"),
		MergeWhitelistUsernames:     sliceFlagPtr(cmd, "merge-whitelist-users"),
		MergeWhitelistTeams:         sliceFlagPtr(cmd, "merge-whitelist-teams"),
		StatusCheckContexts:         sliceFlagPtr(cmd, "status-checks"),
		RequiredApprovals:           int64FlagPtr(cmd, "required-approvals"),
		ApprovalsWhitelistUsernames: sliceFlagPtr(cmd, "approvals-whitelist-users"),
		ApprovalsWhitelistTeams:     sliceFlagPtr(cmd, "approvals-whitelist-teams"),
		BlockOnRejectedReviews:      boolPairFlag(cmd, "block-rejected-reviews", "no-block-rejected-reviews"),
		BlockOnOutdatedBranch:       boolPairFlag(cmd, "block-outdated", "no-block-outdated"),
		DismissStaleApprovals:       boolPairFlag(cmd, "dismiss-stale-approvals", "no-dismiss-stale-approvals"),
		RequireSignedCommits:        boolPairFlag(cmd, "require-signed-commits", "no-require-signed-commits"),
		ProtectedFilePatterns:       stringFlagPtr(cmd, "protected-files"),
		UnprotectedFilePatterns:     stringFlagPtr(cmd, "unprotected-files"),
	}
	if cmd.Flags().Changed("push-whitelist-users") || cmd.Flags().Changed("push-whitelist-teams") {
		enable := len(opts.PushWhitelistUsernames) > 0 || len(opts.PushWhitelistTeams) > 0
		opts.EnablePushWhitelist = &enable
	}
	if cmd.Flags().Changed("merge-whitelist-users") || cmd.Flags().Changed("merge-whitelist-teams") {
		enable := len(opts.MergeWhitelistUsernames) > 0 || len(opts.MergeWhitelistTeams) > 0
		opts.EnableMergeWhitelist = &enable
	}
	if cmd.Flags().Changed("status-checks") {
		enable := len(opts.StatusCheckContexts) > 0
		opts.EnableStatusCheck = &enable
	}

	if dryRunf(cmd, "Would edit branch protection rule %s in %s/%s", rule, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	bp, resp, err := client.EditBranchProtection(owner, repo, rule, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "branch protection rule", rule)
	}
	return printBranchProtection(cmd, bp)
}

func runBranchProtectionDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	rule := args[1]

	if dryRunf(cmd, "Would delete branch protection rule %s from %s/%s", rule, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteBranchProtection(owner, repo, rule)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "branch protection rule", rule)
	}

	return emitMessage(cmd, okMessage("rule deleted"), "Deleted branch protection rule %s", rule)
}

func runTagProtectionList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	rules, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.TagProtection, *gitea.Response, error) {
		return client.ListTagProtection(owner, repo, gitea.ListRepoTagProtectionsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(rules, func() error {
		if len(rules) == 0 {
			printer.Println("No tag protection rules found.")
			return nil
		}
		rows := make([][]string, 0, len(rules))
		for _, r := range rules {
			rows = append(rows, []string{
				fmt.Sprintf("%d", r.Id), r.NamePattern,
				renderValue(r.WhitelistUsernames), renderValue(r.WhitelistTeams),
			})
		}
		return printer.PrintTable([]string{"ID", "Pattern", "Users", "Teams"}, rows)
	})
}

func printTagProtection(cmd *cobra.Command, tp *gitea.TagProtection) error {
	printer := getPrinter(cmd)
	return printer.Emit(tp, func() error {
		var d detail
		d.always("ID", tp.Id)
		d.add("Pattern", tp.NamePattern)
		d.add("Whitelisted users", tp.WhitelistUsernames)
		d.add("Whitelisted teams", tp.WhitelistTeams)
		d.add("Created", tp.Created)
		d.add("Updated", tp.Updated)
		return d.print(printer)
	})
}

func runTagProtectionGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	id, err := int64Arg(args[1], "protection id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tp, resp, err := client.GetTagProtection(owner, repo, id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tag protection rule", args[1])
	}
	return printTagProtection(cmd, tp)
}

func runTagProtectionCreate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	pattern := args[1]

	users, _ := cmd.Flags().GetStringSlice("whitelist-users")
	teams, _ := cmd.Flags().GetStringSlice("whitelist-teams")

	if dryRunf(cmd, "Would protect tags matching %s in %s/%s", pattern, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tp, resp, err := client.CreateTagProtection(owner, repo, gitea.CreateTagProtectionOption{
		NamePattern: pattern, WhitelistUsernames: users, WhitelistTeams: teams,
	})
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	return printTagProtection(cmd, tp)
}

func runTagProtectionUpdate(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	id, err := int64Arg(args[1], "protection id")
	if err != nil {
		return err
	}

	opts := gitea.EditTagProtectionOption{
		NamePattern:        stringFlagPtr(cmd, "pattern"),
		WhitelistUsernames: sliceFlagPtr(cmd, "whitelist-users"),
		WhitelistTeams:     sliceFlagPtr(cmd, "whitelist-teams"),
	}
	if opts.NamePattern == nil && opts.WhitelistUsernames == nil && opts.WhitelistTeams == nil {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass --pattern, --whitelist-users or --whitelist-teams"})
	}

	if dryRunf(cmd, "Would edit tag protection rule %d in %s/%s", id, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	tp, resp, err := client.EditTagProtection(owner, repo, id, opts)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tag protection rule", args[1])
	}
	return printTagProtection(cmd, tp)
}

func runTagProtectionDelete(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	id, err := int64Arg(args[1], "protection id")
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would delete tag protection rule %d from %s/%s", id, owner, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteTagProtection(owner, repo, id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "tag protection rule", args[1])
	}

	return emitMessage(cmd, okMessage("rule deleted"), "Deleted tag protection rule %d", id)
}
