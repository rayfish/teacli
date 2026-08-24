// Package cmd contains star, watch, follow, block and subscription commands.
package cmd

import (
	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var starCmd = &cobra.Command{
	Use:     "star",
	Aliases: []string{"stars"},
	Short:   "Star and unstar repositories",
}

var starListCmd = &cobra.Command{
	Use:   "list [<username>]",
	Short: "List the repositories you, or another user, have starred",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStarList,
}

var starAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>]",
	Short: "Star a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStarAdd,
}

var starRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>]",
	Aliases: []string{"rm"},
	Short:   "Unstar a repository",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runStarRemove,
}

var starCheckCmd = &cobra.Command{
	Use:   "check [<owner>/<repo>]",
	Short: "Report whether you have starred a repository (exit 3 if not)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStarCheck,
}

var stargazerCmd = &cobra.Command{
	Use:   "stargazers [<owner>/<repo>]",
	Short: "List the users who starred a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStargazers,
}

var watchCmd = &cobra.Command{
	Use:     "watch",
	Aliases: []string{"watches"},
	Short:   "Watch and unwatch repositories",
}

var watchListCmd = &cobra.Command{
	Use:   "list [<username>]",
	Short: "List the repositories you, or another user, are watching",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWatchList,
}

var watchAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>]",
	Short: "Watch a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWatchAdd,
}

var watchRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>]",
	Aliases: []string{"rm"},
	Short:   "Stop watching a repository",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runWatchRemove,
}

var watchCheckCmd = &cobra.Command{
	Use:   "check [<owner>/<repo>]",
	Short: "Report whether you are watching a repository (exit 3 if not)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runWatchCheck,
}

var followCmd = &cobra.Command{
	Use:   "follow",
	Short: "Follow and unfollow users",
}

var followListCmd = &cobra.Command{
	Use:   "list [<username>]",
	Short: "List the users you, or another user, follow",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runFollowList,
}

var followerListCmd = &cobra.Command{
	Use:   "followers [<username>]",
	Short: "List your followers, or another user's",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runFollowerList,
}

var followAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Follow a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runFollowAdd,
}

var followRemoveCmd = &cobra.Command{
	Use:     "remove <username>",
	Aliases: []string{"rm", "unfollow"},
	Short:   "Unfollow a user",
	Args:    cobra.ExactArgs(1),
	RunE:    runFollowRemove,
}

var followCheckCmd = &cobra.Command{
	Use:   "check <username> [<target>]",
	Short: "Report whether you, or one user, follow another (exit 3 if not)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runFollowCheck,
}

var blockCmd = &cobra.Command{
	Use:     "block",
	Aliases: []string{"blocks"},
	Short:   "Block and unblock users, for your account or an organization",
}

var blockListCmd = &cobra.Command{
	Use:   "list",
	Short: "List the users you, or an organization, have blocked",
	Args:  cobra.NoArgs,
	RunE:  runBlockList,
}

var blockAddCmd = &cobra.Command{
	Use:   "add <username>",
	Short: "Block a user",
	Args:  cobra.ExactArgs(1),
	RunE:  runBlockAdd,
}

var blockRemoveCmd = &cobra.Command{
	Use:     "remove <username>",
	Aliases: []string{"rm", "unblock"},
	Short:   "Unblock a user",
	Args:    cobra.ExactArgs(1),
	RunE:    runBlockRemove,
}

var blockCheckCmd = &cobra.Command{
	Use:   "check <username>",
	Short: "Report whether a user is blocked (exit 3 if not)",
	Args:  cobra.ExactArgs(1),
	RunE:  runBlockCheck,
}

var subscriptionCmd = &cobra.Command{
	Use:     "subscription",
	Aliases: []string{"subscribe", "subscriptions"},
	Short:   "Manage issue subscriptions",
}

var subscriptionListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>] <issue-number>",
	Short: "List the users subscribed to an issue",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runSubscriptionList,
}

var subscriptionAddCmd = &cobra.Command{
	Use:   "add [<owner>/<repo>] <issue-number> [<username>]",
	Short: "Subscribe yourself, or another user, to an issue",
	Args:  cobra.RangeArgs(1, 3),
	RunE:  runSubscriptionAdd,
}

var subscriptionRemoveCmd = &cobra.Command{
	Use:     "remove [<owner>/<repo>] <issue-number> [<username>]",
	Aliases: []string{"rm", "unsubscribe"},
	Short:   "Unsubscribe yourself, or another user, from an issue",
	Args:    cobra.RangeArgs(1, 3),
	RunE:    runSubscriptionRemove,
}

var subscriptionCheckCmd = &cobra.Command{
	Use:   "check [<owner>/<repo>] <issue-number>",
	Short: "Report whether you are subscribed to an issue (exit 3 if not)",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runSubscriptionCheck,
}

func init() {
	RootCmd.AddCommand(starCmd, watchCmd, followCmd, blockCmd, subscriptionCmd)
	starCmd.AddCommand(starListCmd, starAddCmd, starRemoveCmd, starCheckCmd)
	repoCmd.AddCommand(stargazerCmd)
	watchCmd.AddCommand(watchListCmd, watchAddCmd, watchRemoveCmd, watchCheckCmd)
	followCmd.AddCommand(followListCmd, followerListCmd, followAddCmd, followRemoveCmd, followCheckCmd)
	blockCmd.AddCommand(blockListCmd, blockAddCmd, blockRemoveCmd, blockCheckCmd)
	subscriptionCmd.AddCommand(subscriptionListCmd, subscriptionAddCmd,
		subscriptionRemoveCmd, subscriptionCheckCmd)

	addPageFlags(stargazerCmd, followListCmd, followerListCmd, blockListCmd, subscriptionListCmd)

	for _, c := range []*cobra.Command{blockListCmd, blockAddCmd, blockRemoveCmd, blockCheckCmd} {
		c.Flags().String("org", "", "Act as this organization instead of as yourself")
	}
}

func runStarList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		repos []*gitea.Repository
		resp  *gitea.Response
	)
	if len(args) == 1 {
		repos, resp, err = client.GetStarredRepos(args[0])
	} else {
		repos, resp, err = client.GetMyStarredRepos()
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitRepoTable(cmd, repos)
}

func runStarAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would star %s/%s", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.StarRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("starred"), "Starred %s/%s", owner, repo)
}

func runStarRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would unstar %s/%s", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.UnStarRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("unstarred"), "Unstarred %s/%s", owner, repo)
}

func runStarCheck(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	starred, resp, err := client.IsRepoStarring(owner, repo)
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	if !starred {
		return errors.NewNotFoundError("star", args[0])
	}
	return emitMessage(cmd, map[string]any{"starred": true}, "You have starred %s/%s", owner, repo)
}

func runStargazers(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListRepoStargazers(owner, repo, gitea.ListStargazersOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No stargazers.")
}

func runWatchList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		repos []*gitea.Repository
		resp  *gitea.Response
	)
	if len(args) == 1 {
		repos, resp, err = client.GetWatchedRepos(args[0])
	} else {
		repos, resp, err = client.GetMyWatchedRepos()
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitRepoTable(cmd, repos)
}

func runWatchAdd(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would watch %s/%s", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.WatchRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("watching"), "Watching %s/%s", owner, repo)
}

func runWatchRemove(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would stop watching %s/%s", owner, repo) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.UnWatchRepo(owner, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[0])
	}
	return emitMessage(cmd, okMessage("unwatched"), "Stopped watching %s/%s", owner, repo)
}

func runWatchCheck(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	watch, resp, err := client.CheckRepoWatch(owner, repo)
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	if !watch {
		return errors.NewNotFoundError("watch", args[0])
	}
	return emitMessage(cmd, map[string]any{"watching": true}, "You are watching %s/%s", owner, repo)
}

func runFollowList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		if len(args) == 1 {
			return client.ListFollowing(args[0], gitea.ListFollowingOptions{ListOptions: lo})
		}
		return client.ListMyFollowing(gitea.ListFollowingOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "Not following anyone.")
}

func runFollowerList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		if len(args) == 1 {
			return client.ListFollowers(args[0], gitea.ListFollowersOptions{ListOptions: lo})
		}
		return client.ListMyFollowers(gitea.ListFollowersOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No followers.")
}

func runFollowAdd(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would follow %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.Follow(args[0])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}
	return emitMessage(cmd, okMessage("following"), "Following %s", args[0])
}

func runFollowRemove(cmd *cobra.Command, args []string) error {
	if dryRunf(cmd, "Would unfollow %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.Unfollow(args[0])
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}
	return emitMessage(cmd, okMessage("unfollowed"), "Unfollowed %s", args[0])
}

func runFollowCheck(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// These two SDK calls return no error, only the boolean and the response.
	var (
		following bool
		subject   = "You"
		target    = args[0]
	)
	if len(args) == 2 {
		subject, target = args[0], args[1]
		following, _ = client.IsUserFollowing(args[0], args[1])
	} else {
		following, _ = client.IsFollowing(args[0])
	}
	if !following {
		return errors.NewNotFoundError("following relationship", subject+" -> "+target)
	}

	return emitMessage(cmd, map[string]any{"following": true, "user": subject, "target": target},
		"%s follow %s", subject, target)
}

func runBlockList(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		if org != "" {
			return client.ListOrgBlocks(org, gitea.ListOrgBlocksOptions{ListOptions: lo})
		}
		return client.ListMyBlocks(gitea.ListUserBlocksOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No blocked users.")
}

func runBlockAdd(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	if dryRunf(cmd, "Would block %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var resp *gitea.Response
	if org != "" {
		resp, err = client.BlockOrgUser(org, args[0])
	} else {
		resp, err = client.BlockUser(args[0])
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}

	return emitMessage(cmd, okMessage("blocked"), "Blocked %s", args[0])
}

func runBlockRemove(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	if dryRunf(cmd, "Would unblock %s", args[0]) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var resp *gitea.Response
	if org != "" {
		resp, err = client.UnblockOrgUser(org, args[0])
	} else {
		resp, err = client.UnblockUser(args[0])
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "user", args[0])
	}

	return emitMessage(cmd, okMessage("unblocked"), "Unblocked %s", args[0])
}

func runBlockCheck(cmd *cobra.Command, args []string) error {
	org, _ := cmd.Flags().GetString("org")
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var (
		blocked bool
		resp    *gitea.Response
	)
	if org != "" {
		blocked, resp, err = client.CheckOrgBlock(org, args[0])
	} else {
		blocked, resp, err = client.CheckUserBlock(args[0])
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}
	if !blocked {
		return errors.NewNotFoundError("block", args[0])
	}

	return emitMessage(cmd, map[string]any{"blocked": true, "user": args[0]}, "%s is blocked", args[0])
}

func runSubscriptionList(cmd *cobra.Command, args []string) error {
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

	users, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListIssueSubscribers(owner, repo, index,
			gitea.ListIssueSubscribersOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, users, "No subscribers.")
}

func runSubscriptionAdd(cmd *cobra.Command, args []string) error {
	return changeSubscription(cmd, args, true)
}

func runSubscriptionRemove(cmd *cobra.Command, args []string) error {
	return changeSubscription(cmd, args, false)
}

func changeSubscription(cmd *cobra.Command, args []string, subscribe bool) error {
	owner, repo, args, err := repoTargetByArg(cmd, args)
	if err != nil {
		return err
	}
	index, err := int64Arg(args[1], "issue number")
	if err != nil {
		return err
	}

	who := "you"
	if len(args) == 3 {
		who = args[2]
	}
	verb := "unsubscribe"
	if subscribe {
		verb = "subscribe"
	}
	if dryRunf(cmd, "Would %s %s to %s/%s#%d", verb, who, owner, repo, index) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var resp *gitea.Response
	switch {
	case len(args) == 3 && subscribe:
		resp, err = client.AddIssueSubscription(owner, repo, index, args[2])
	case len(args) == 3:
		resp, err = client.DeleteIssueSubscription(owner, repo, index, args[2])
	case subscribe:
		resp, err = client.IssueSubscribe(owner, repo, index)
	default:
		resp, err = client.IssueUnSubscribe(owner, repo, index)
	}
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}

	past := "Unsubscribed"
	if subscribe {
		past = "Subscribed"
	}
	return emitMessage(cmd, okMessage(verb+"d"), "%s %s on %s/%s#%d", past, who, owner, repo, index)
}

func runSubscriptionCheck(cmd *cobra.Command, args []string) error {
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

	watch, resp, err := client.CheckIssueSubscription(owner, repo, index)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "issue", args[1])
	}
	if watch == nil || !watch.Subscribed {
		return errors.NewNotFoundError("subscription", args[1])
	}

	return emitMessage(cmd, watch, "You are subscribed to %s/%s#%d", owner, repo, index)
}
