// Package cmd contains organization team commands.
package cmd

import (
	"fmt"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:     "team",
	Aliases: []string{"teams"},
	Short:   "Manage organization teams",
}

var teamListCmd = &cobra.Command{
	Use:   "list <org>",
	Short: "List the teams of an organization",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamList,
}

var teamMineCmd = &cobra.Command{
	Use:   "mine",
	Short: "List the teams you belong to",
	Args:  cobra.NoArgs,
	RunE:  runTeamMine,
}

var teamGetCmd = &cobra.Command{
	Use:   "get <team-id>",
	Short: "Get one team by its numeric ID",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamGet,
}

var teamSearchCmd = &cobra.Command{
	Use:   "search <org> <query>",
	Short: "Search an organization's teams by name",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamSearch,
}

var teamCreateCmd = &cobra.Command{
	Use:   "create <org> <name>",
	Short: "Create a team",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamCreate,
}

var teamUpdateCmd = &cobra.Command{
	Use:     "update <team-id>",
	Aliases: []string{"edit"},
	Short:   "Edit a team",
	Args:    cobra.ExactArgs(1),
	RunE:    runTeamUpdate,
}

var teamDeleteCmd = &cobra.Command{
	Use:     "delete <team-id>",
	Aliases: []string{"rm"},
	Short:   "Delete a team",
	Args:    cobra.ExactArgs(1),
	RunE:    runTeamDelete,
}

var teamMemberCmd = &cobra.Command{
	Use:     "member",
	Aliases: []string{"members"},
	Short:   "Manage team membership",
}

var teamMemberListCmd = &cobra.Command{
	Use:   "list <team-id>",
	Short: "List the members of a team",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamMemberList,
}

var teamMemberAddCmd = &cobra.Command{
	Use:   "add <team-id> <username>...",
	Short: "Add users to a team",
	Args:  cobra.MinimumNArgs(2),
	RunE:  runTeamMemberAdd,
}

var teamMemberRemoveCmd = &cobra.Command{
	Use:     "remove <team-id> <username>...",
	Aliases: []string{"rm"},
	Short:   "Remove users from a team",
	Args:    cobra.MinimumNArgs(2),
	RunE:    runTeamMemberRemove,
}

var teamMemberCheckCmd = &cobra.Command{
	Use:   "check <team-id> <username>",
	Short: "Report whether a user is on a team (exit 3 if not)",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamMemberCheck,
}

var teamRepoCmd = &cobra.Command{
	Use:     "repo",
	Aliases: []string{"repos"},
	Short:   "Manage the repositories a team can reach",
}

var teamRepoListCmd = &cobra.Command{
	Use:   "list <team-id>",
	Short: "List the repositories a team can reach",
	Args:  cobra.ExactArgs(1),
	RunE:  runTeamRepoList,
}

var teamRepoAddCmd = &cobra.Command{
	Use:   "add <team-id> <owner>/<repo>",
	Short: "Give a team access to a repository",
	Args:  cobra.ExactArgs(2),
	RunE:  runTeamRepoAdd,
}

var teamRepoRemoveCmd = &cobra.Command{
	Use:     "remove <team-id> <owner>/<repo>",
	Aliases: []string{"rm"},
	Short:   "Remove a team's access to a repository",
	Args:    cobra.ExactArgs(2),
	RunE:    runTeamRepoRemove,
}

func init() {
	RootCmd.AddCommand(teamCmd)
	teamCmd.AddCommand(teamListCmd, teamMineCmd, teamGetCmd, teamSearchCmd, teamCreateCmd,
		teamUpdateCmd, teamDeleteCmd, teamMemberCmd, teamRepoCmd)
	teamMemberCmd.AddCommand(teamMemberListCmd, teamMemberAddCmd, teamMemberRemoveCmd, teamMemberCheckCmd)
	teamRepoCmd.AddCommand(teamRepoListCmd, teamRepoAddCmd, teamRepoRemoveCmd)

	addPageFlags(teamListCmd, teamMineCmd, teamSearchCmd, teamMemberListCmd, teamRepoListCmd)

	for _, c := range []*cobra.Command{teamCreateCmd, teamUpdateCmd} {
		c.Flags().StringP("description", "d", "", "Team description")
		c.Flags().StringP("permission", "p", "", "Access level: none, read, write, admin, owner")
		c.Flags().StringSlice("units", nil,
			"Repository units the team can reach, e.g. repo.code,repo.issues,repo.pulls")
		c.Flags().Bool("can-create-repos", false, "Let members create repositories in the organization")
		c.Flags().Bool("no-create-repos", false, "Stop members creating repositories")
		c.Flags().Bool("all-repos", false, "Give the team access to every repository")
		c.Flags().Bool("no-all-repos", false, "Limit the team to explicitly granted repositories")
	}
	teamUpdateCmd.Flags().String("name", "", "New team name")
	teamSearchCmd.Flags().Bool("in-description", false, "Also match the query against descriptions")
}

// parseAccessMode validates a --permission value. Gitea silently downgrades an
// unrecognised mode to "none", so reject it here instead.
func parseAccessMode(s string) (gitea.AccessMode, error) {
	switch strings.ToLower(s) {
	case "":
		return "", nil
	case "none":
		return gitea.AccessModeNone, nil
	case "read":
		return gitea.AccessModeRead, nil
	case "write":
		return gitea.AccessModeWrite, nil
	case "admin":
		return gitea.AccessModeAdmin, nil
	case "owner":
		return gitea.AccessModeOwner, nil
	default:
		return "", errors.NewValidationError(fmt.Sprintf("invalid permission: %s", s),
			map[string]interface{}{"expected": "none, read, write, admin, or owner"})
	}
}

func parseRepoUnits(values []string) []gitea.RepoUnitType {
	units := make([]gitea.RepoUnitType, 0, len(values))
	for _, v := range values {
		units = append(units, gitea.RepoUnitType(v))
	}
	return units
}

func emitTeams(cmd *cobra.Command, teams []*gitea.Team) error {
	printer := getPrinter(cmd)
	return printer.Emit(teams, func() error {
		if len(teams) == 0 {
			printer.Println("No teams found.")
			return nil
		}
		rows := make([][]string, 0, len(teams))
		for _, t := range teams {
			org := ""
			if t.Organization != nil {
				org = t.Organization.Name
			}
			rows = append(rows, []string{
				fmt.Sprintf("%d", t.ID), t.Name, org, string(t.Permission),
				yesNo(t.IncludesAllRepositories), cell(t.Description, 40),
			})
		}
		return printer.PrintTable(
			[]string{"ID", "Name", "Org", "Permission", "All repos", "Description"}, rows)
	})
}

func runTeamList(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	teams, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Team, *gitea.Response, error) {
		return client.ListOrgTeams(args[0], gitea.ListTeamsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}
	return emitTeams(cmd, teams)
}

func runTeamMine(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	teams, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Team, *gitea.Response, error) {
		return client.ListMyTeams(&gitea.ListTeamsOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}
	return emitTeams(cmd, teams)
}

func runTeamGet(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	team, resp, err := client.GetTeam(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team", args[0])
	}

	printer := getPrinter(cmd)
	return printer.Emit(team, func() error {
		var d detail
		d.always("ID", team.ID)
		d.add("Name", team.Name)
		if team.Organization != nil {
			d.add("Organization", team.Organization.Name)
		}
		d.add("Permission", string(team.Permission))
		d.always("Can create repos", team.CanCreateOrgRepo)
		d.always("All repositories", team.IncludesAllRepositories)
		units := make([]string, 0, len(team.Units))
		for _, u := range team.Units {
			units = append(units, string(u))
		}
		d.add("Units", units)
		d.add("Description", team.Description)
		return d.print(printer)
	})
}

func runTeamSearch(cmd *cobra.Command, args []string) error {
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	inDescription, _ := cmd.Flags().GetBool("in-description")

	teams, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Team, *gitea.Response, error) {
		return client.SearchOrgTeams(args[0], &gitea.SearchTeamsOptions{
			ListOptions: lo, Query: args[1], IncludeDescription: inDescription,
		})
	})
	if err != nil {
		return err
	}
	return emitTeams(cmd, teams)
}

func runTeamCreate(cmd *cobra.Command, args []string) error {
	org, name := args[0], args[1]

	permission, _ := cmd.Flags().GetString("permission")
	mode, err := parseAccessMode(permission)
	if err != nil {
		return err
	}
	if mode == "" {
		mode = gitea.AccessModeRead
	}
	description, _ := cmd.Flags().GetString("description")
	unitValues, _ := cmd.Flags().GetStringSlice("units")

	canCreate := boolPairFlag(cmd, "can-create-repos", "no-create-repos")
	allRepos := boolPairFlag(cmd, "all-repos", "no-all-repos")

	opts := gitea.CreateTeamOption{
		Name:        name,
		Description: description,
		Permission:  mode,
		Units:       parseRepoUnits(unitValues),
	}
	if canCreate != nil {
		opts.CanCreateOrgRepo = *canCreate
	}
	if allRepos != nil {
		opts.IncludesAllRepositories = *allRepos
	}

	if dryRunf(cmd, "Would create team %s in %s with %s access", name, org, mode) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	team, resp, err := client.CreateTeam(org, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, team, "Created team %s (id %d) in %s", team.Name, team.ID, org)
}

func runTeamUpdate(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}

	permission, _ := cmd.Flags().GetString("permission")
	mode, err := parseAccessMode(permission)
	if err != nil {
		return err
	}
	name, _ := cmd.Flags().GetString("name")
	unitValues, _ := cmd.Flags().GetStringSlice("units")

	opts := gitea.EditTeamOption{
		Name:                    name,
		Description:             stringFlagPtr(cmd, "description"),
		Permission:              mode,
		CanCreateOrgRepo:        boolPairFlag(cmd, "can-create-repos", "no-create-repos"),
		IncludesAllRepositories: boolPairFlag(cmd, "all-repos", "no-all-repos"),
		Units:                   parseRepoUnits(unitValues),
	}

	if opts.Name == "" && opts.Description == nil && opts.Permission == "" &&
		opts.CanCreateOrgRepo == nil && opts.IncludesAllRepositories == nil && len(opts.Units) == 0 {
		return errors.NewValidationError("no changes given",
			map[string]interface{}{"hint": "pass at least one flag, e.g. --permission write"})
	}

	if dryRunf(cmd, "Would edit team %d", id) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// The edit endpoint replaces the whole team, so a partial update needs the
	// current values for anything the caller did not name.
	current, resp, err := client.GetTeam(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team", args[0])
	}
	if opts.Name == "" {
		opts.Name = current.Name
	}
	if opts.Permission == "" {
		opts.Permission = current.Permission
	}
	if len(opts.Units) == 0 {
		opts.Units = current.Units
	}

	resp, err = client.EditTeam(id, opts)
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, okMessage("team updated"), "Updated team %d", id)
}

func runTeamDelete(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	if dryRunf(cmd, "Would delete team %d", id) {
		return nil
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.DeleteTeam(id)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team", args[0])
	}
	return emitMessage(cmd, okMessage("team deleted"), "Deleted team %d", id)
}

func runTeamMemberList(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	members, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.User, *gitea.Response, error) {
		return client.ListTeamMembers(id, gitea.ListTeamMembersOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitUserTable(cmd, members, "No members found.")
}

func runTeamMemberAdd(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	users := args[1:]

	if dryRunf(cmd, "Would add %s to team %d", strings.Join(users, ", "), id) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	for _, u := range users {
		resp, err := client.AddTeamMember(id, u)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "user", u)
		}
	}

	return emitMessage(cmd, okMessage("members added"), "Added %d user(s) to team %d", len(users), id)
}

func runTeamMemberRemove(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	users := args[1:]

	if dryRunf(cmd, "Would remove %s from team %d", strings.Join(users, ", "), id) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	for _, u := range users {
		resp, err := client.RemoveTeamMember(id, u)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "team member", u)
		}
	}

	return emitMessage(cmd, okMessage("members removed"), "Removed %d user(s) from team %d", len(users), id)
}

func runTeamMemberCheck(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	username := args[1]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	user, resp, err := client.GetTeamMember(id, username)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "team member", username)
	}

	printer := getPrinter(cmd)
	return printer.Emit(user, func() error {
		printer.Printf("%s is a member of team %d\n", user.UserName, id)
		return nil
	})
}

func runTeamRepoList(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	repos, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Repository, *gitea.Response, error) {
		return client.ListTeamRepositories(id, gitea.ListTeamRepositoriesOptions{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	return emitRepoTable(cmd, repos)
}

func runTeamRepoAdd(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	org, repo, err := repoArg(args[1])
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would give team %d access to %s/%s", id, org, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.AddTeamRepository(id, org, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[1])
	}

	return emitMessage(cmd, okMessage("repository added"), "Gave team %d access to %s/%s", id, org, repo)
}

func runTeamRepoRemove(cmd *cobra.Command, args []string) error {
	id, err := int64Arg(args[0], "team id")
	if err != nil {
		return err
	}
	org, repo, err := repoArg(args[1])
	if err != nil {
		return err
	}

	if dryRunf(cmd, "Would remove team %d's access to %s/%s", id, org, repo) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}
	resp, err := client.RemoveTeamRepository(id, org, repo)
	if err != nil {
		return errors.FromGiteaNotFound(resp, err, "repository", args[1])
	}

	return emitMessage(cmd, okMessage("repository removed"),
		"Removed team %d's access to %s/%s", id, org, repo)
}
