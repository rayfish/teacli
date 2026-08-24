// Package cmd contains Actions secret, variable and workflow commands.
package cmd

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

// Secrets and variables live on a repo or on an org. Every command in this file
// takes a repo argument by default and switches to the org endpoints when --org
// is given, so `--org myorg` replaces the `owner/repo` argument rather than
// adding to it.

var actionSecretCmd = &cobra.Command{
	Use:     "secret",
	Aliases: []string{"secrets"},
	Short:   "Manage Actions secrets on a repository or organization",
}

var actionSecretListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List Actions secrets (values are never returned by the server)",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runActionSecretList,
}

var actionSecretSetCmd = &cobra.Command{
	Use:     "set [<owner>/<repo>] <name> <value>",
	Aliases: []string{"create", "update"},
	Short:   "Create or update an Actions secret",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runActionSecretSet,
}

var actionSecretDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <name>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete an Actions secret",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runActionSecretDelete,
}

var actionVariableCmd = &cobra.Command{
	Use:     "variable",
	Aliases: []string{"variables", "var"},
	Short:   "Manage Actions variables on a repository or organization",
}

var actionVariableListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List Actions variables",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runActionVariableList,
}

var actionVariableGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <name>",
	Short: "Get one Actions variable",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionVariableGet,
}

var actionVariableSetCmd = &cobra.Command{
	Use:     "set [<owner>/<repo>] <name> <value>",
	Aliases: []string{"create", "update"},
	Short:   "Create or update an Actions variable",
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runActionVariableSet,
}

var actionVariableDeleteCmd = &cobra.Command{
	Use:     "delete [<owner>/<repo>] <name>",
	Aliases: []string{"remove", "rm"},
	Short:   "Delete an Actions variable",
	Args:    cobra.RangeArgs(1, 2),
	RunE:    runActionVariableDelete,
}

var actionWorkflowCmd = &cobra.Command{
	Use:     "workflow",
	Aliases: []string{"workflows"},
	Short:   "Inspect and enable or disable repository workflows",
}

var actionWorkflowListCmd = &cobra.Command{
	Use:   "list [<owner>/<repo>]",
	Short: "List the workflows defined in a repository",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runActionWorkflowList,
}

var actionWorkflowGetCmd = &cobra.Command{
	Use:   "get [<owner>/<repo>] <workflow-file>",
	Short: "Get one workflow by its file name",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionWorkflowGet,
}

var actionWorkflowEnableCmd = &cobra.Command{
	Use:   "enable [<owner>/<repo>] <workflow-file>",
	Short: "Enable a workflow",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionWorkflowEnable,
}

var actionWorkflowDisableCmd = &cobra.Command{
	Use:   "disable [<owner>/<repo>] <workflow-file>",
	Short: "Disable a workflow",
	Args:  cobra.RangeArgs(1, 2),
	RunE:  runActionWorkflowDisable,
}

func init() {
	actionCmd.AddCommand(actionSecretCmd, actionVariableCmd, actionWorkflowCmd)
	actionSecretCmd.AddCommand(actionSecretListCmd, actionSecretSetCmd, actionSecretDeleteCmd)
	actionVariableCmd.AddCommand(actionVariableListCmd, actionVariableGetCmd, actionVariableSetCmd, actionVariableDeleteCmd)
	actionWorkflowCmd.AddCommand(actionWorkflowListCmd, actionWorkflowGetCmd, actionWorkflowEnableCmd, actionWorkflowDisableCmd)

	for _, c := range []*cobra.Command{
		actionSecretListCmd, actionSecretSetCmd, actionSecretDeleteCmd,
		actionVariableListCmd, actionVariableGetCmd, actionVariableSetCmd, actionVariableDeleteCmd,
	} {
		c.Flags().String("org", "", "Operate on this organization instead of a repository")
	}

	addPageFlags(actionSecretListCmd, actionVariableListCmd)
	actionSecretSetCmd.Flags().StringP("description", "d", "", "Secret description")
	actionVariableSetCmd.Flags().StringP("description", "d", "", "Variable description (organizations only)")
}

// actionScope resolves whether a command targets an org or a repo, and splits
// the remaining positional arguments. With --org the repo argument is absent, so
// every positional shifts left by one.
type actionScope struct {
	org   string
	owner string
	repo  string
	rest  []string
}

func (s actionScope) isOrg() bool { return s.org != "" }

func (s actionScope) label() string {
	if s.isOrg() {
		return "organization " + s.org
	}
	return s.owner + "/" + s.repo
}

func resolveActionScope(cmd *cobra.Command, args []string, wantRest int) (actionScope, error) {
	org, _ := cmd.Flags().GetString("org")

	if org != "" {
		if len(args) != wantRest {
			return actionScope{}, errors.NewValidationError(
				fmt.Sprintf("expected %d argument(s) after --org, got %d", wantRest, len(args)),
				map[string]interface{}{"hint": "--org replaces the owner/repo argument"})
		}
		return actionScope{org: org, rest: args}, nil
	}

	if len(args) < wantRest || len(args) > wantRest+1 {
		return actionScope{}, errors.NewValidationError(
			fmt.Sprintf("expected %d argument(s), with or without a leading <owner>/<repo>", wantRest),
			map[string]interface{}{"hint": "or pass --org <name> to target an organization"})
	}
	owner, repo, args, err := repoTarget(cmd, args, wantRest)
	if err != nil {
		return actionScope{}, err
	}
	return actionScope{owner: owner, repo: repo, rest: args[1:]}, nil
}

func runActionSecretList(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	secrets, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.Secret, *gitea.Response, error) {
		if scope.isOrg() {
			return client.ListOrgActionSecret(scope.org, gitea.ListOrgActionSecretOption{ListOptions: lo})
		}
		return client.ListRepoActionSecret(scope.owner, scope.repo, gitea.ListRepoActionSecretOption{ListOptions: lo})
	})
	if err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(secrets, func() error {
		if len(secrets) == 0 {
			printer.Println("No secrets found.")
			return nil
		}
		rows := make([][]string, 0, len(secrets))
		for _, s := range secrets {
			rows = append(rows, []string{s.Name, s.Description, renderValue(s.Created)})
		}
		return printer.PrintTable([]string{"Name", "Description", "Created"}, rows)
	})
}

func runActionSecretSet(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 2)
	if err != nil {
		return err
	}
	name, value := scope.rest[0], scope.rest[1]
	description, _ := cmd.Flags().GetString("description")

	if dryRunf(cmd, "Would set secret %s on %s", name, scope.label()) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	opts := gitea.CreateSecretOption{Name: name, Data: value, Description: description}
	var resp *gitea.Response
	if scope.isOrg() {
		resp, err = client.CreateOrgActionSecret(scope.org, opts)
	} else {
		resp, err = client.CreateRepoActionSecret(scope.owner, scope.repo, opts)
	}
	if err != nil {
		return errors.FromGitea(resp, err)
	}

	return emitMessage(cmd, okMessage("secret set"), "Set secret %s on %s", name, scope.label())
}

func runActionSecretDelete(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 1)
	if err != nil {
		return err
	}
	name := scope.rest[0]

	if dryRunf(cmd, "Would delete secret %s from %s", name, scope.label()) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if scope.isOrg() {
		// The SDK has no org secret delete, so call the endpoint directly.
		path := fmt.Sprintf("orgs/%s/actions/secrets/%s", url.PathEscape(scope.org), url.PathEscape(name))
		if _, err := apiRequest(cmd, http.MethodDelete, path, nil); err != nil {
			return err
		}
	} else {
		resp, err := client.DeleteRepoActionSecret(scope.owner, scope.repo, name)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "secret", name)
		}
	}

	return emitMessage(cmd, okMessage("secret deleted"), "Deleted secret %s from %s", name, scope.label())
}

// actionVariable is the shape both the repo and the org variable endpoints are
// rendered as, so one printer serves both.
type actionVariable struct {
	Name        string `json:"name"`
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

func runActionVariableList(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 0)
	if err != nil {
		return err
	}
	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var vars []actionVariable
	if scope.isOrg() {
		items, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.OrgActionVariable, *gitea.Response, error) {
			return client.ListOrgActionVariable(scope.org, gitea.ListOrgActionVariableOption{ListOptions: lo})
		})
		if err != nil {
			return err
		}
		for _, v := range items {
			vars = append(vars, actionVariable{Name: v.Name, Value: v.Data, Description: v.Description})
		}
	} else {
		items, err := fetchList(cmd, func(lo gitea.ListOptions) ([]*gitea.RepoActionVariable, *gitea.Response, error) {
			return client.ListRepoActionVariable(scope.owner, scope.repo, gitea.ListRepoActionVariableOption{ListOptions: lo})
		})
		if err != nil {
			return err
		}
		for _, v := range items {
			vars = append(vars, actionVariable{Name: v.Name, Value: v.Value})
		}
	}

	printer := getPrinter(cmd)
	return printer.Emit(vars, func() error {
		if len(vars) == 0 {
			printer.Println("No variables found.")
			return nil
		}
		rows := make([][]string, 0, len(vars))
		for _, v := range vars {
			rows = append(rows, []string{v.Name, v.Value, v.Description})
		}
		return printer.PrintTable([]string{"Name", "Value", "Description"}, rows)
	})
}

func runActionVariableGet(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 1)
	if err != nil {
		return err
	}
	name := scope.rest[0]

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	var v actionVariable
	if scope.isOrg() {
		got, resp, err := client.GetOrgActionVariable(scope.org, name)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "variable", name)
		}
		v = actionVariable{Name: got.Name, Value: got.Data, Description: got.Description}
	} else {
		got, resp, err := client.GetRepoActionVariable(scope.owner, scope.repo, name)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "variable", name)
		}
		v = actionVariable{Name: got.Name, Value: got.Value}
	}

	printer := getPrinter(cmd)
	return printer.Emit(v, func() error {
		var d detail
		d.add("Name", v.Name)
		d.always("Value", v.Value)
		d.add("Description", v.Description)
		return d.print(printer)
	})
}

func runActionVariableSet(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 2)
	if err != nil {
		return err
	}
	name, value := scope.rest[0], scope.rest[1]
	description, _ := cmd.Flags().GetString("description")

	if dryRunf(cmd, "Would set variable %s on %s", name, scope.label()) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	// Gitea has no upsert for variables: creating an existing one is a conflict,
	// so fall back to the update endpoint when the create is rejected.
	if scope.isOrg() {
		resp, err := client.CreateOrgActionVariable(scope.org, gitea.CreateOrgActionVariableOption{
			Name: name, Value: value, Description: description,
		})
		if err != nil {
			if !isConflict(resp) {
				return errors.FromGitea(resp, err)
			}
			resp, err = client.UpdateOrgActionVariable(scope.org, name, gitea.UpdateOrgActionVariableOption{
				Value: value, Description: description,
			})
			if err != nil {
				return errors.FromGitea(resp, err)
			}
		}
	} else {
		resp, err := client.CreateRepoActionVariable(scope.owner, scope.repo, name, value)
		if err != nil {
			if !isConflict(resp) {
				return errors.FromGitea(resp, err)
			}
			resp, err = client.UpdateRepoActionVariable(scope.owner, scope.repo, name, value)
			if err != nil {
				return errors.FromGitea(resp, err)
			}
		}
	}

	return emitMessage(cmd, okMessage("variable set"), "Set variable %s on %s", name, scope.label())
}

// isConflict reports whether the server rejected a create because the resource
// already exists.
func isConflict(resp *gitea.Response) bool {
	return resp != nil && resp.Response != nil &&
		(resp.StatusCode == http.StatusConflict || resp.StatusCode == http.StatusBadRequest)
}

func runActionVariableDelete(cmd *cobra.Command, args []string) error {
	scope, err := resolveActionScope(cmd, args, 1)
	if err != nil {
		return err
	}
	name := scope.rest[0]

	if dryRunf(cmd, "Would delete variable %s from %s", name, scope.label()) {
		return nil
	}

	client, err := getClient(cmd)
	if err != nil {
		return err
	}

	if scope.isOrg() {
		// The SDK has no org variable delete, so call the endpoint directly.
		path := fmt.Sprintf("orgs/%s/actions/variables/%s", url.PathEscape(scope.org), url.PathEscape(name))
		if _, err := apiRequest(cmd, http.MethodDelete, path, nil); err != nil {
			return err
		}
	} else {
		resp, err := client.DeleteRepoActionVariable(scope.owner, scope.repo, name)
		if err != nil {
			return errors.FromGiteaNotFound(resp, err, "variable", name)
		}
	}

	return emitMessage(cmd, okMessage("variable deleted"), "Deleted variable %s from %s", name, scope.label())
}

// actionWorkflow mirrors the workflow object Gitea returns. The SDK does not
// wrap the workflow endpoints, so these commands go through apiRequest.
type actionWorkflow struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Path    string `json:"path"`
	State   string `json:"state"`
	HTMLURL string `json:"html_url"`
	URL     string `json:"url"`
}

type actionWorkflowList struct {
	TotalCount int64             `json:"total_count"`
	Workflows  []*actionWorkflow `json:"workflows"`
}

func workflowPath(owner, repo, workflow string) string {
	return fmt.Sprintf("repos/%s/%s/actions/workflows/%s",
		url.PathEscape(owner), url.PathEscape(repo), url.PathEscape(workflow))
}

func runActionWorkflowList(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 0)
	if err != nil {
		return err
	}

	var list actionWorkflowList
	path := fmt.Sprintf("repos/%s/%s/actions/workflows", url.PathEscape(owner), url.PathEscape(repo))
	if err := apiGetInto(cmd, path, &list); err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(list.Workflows, func() error {
		if len(list.Workflows) == 0 {
			printer.Println("No workflows found.")
			return nil
		}
		rows := make([][]string, 0, len(list.Workflows))
		for _, w := range list.Workflows {
			rows = append(rows, []string{w.ID, w.Name, w.State})
		}
		return printer.PrintTable([]string{"File", "Name", "State"}, rows)
	})
}

func runActionWorkflowGet(cmd *cobra.Command, args []string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}

	var w actionWorkflow
	if err := apiGetInto(cmd, workflowPath(owner, repo, args[1]), &w); err != nil {
		return err
	}

	printer := getPrinter(cmd)
	return printer.Emit(w, func() error {
		var d detail
		d.add("File", w.ID)
		d.add("Name", w.Name)
		d.add("Path", w.Path)
		d.add("State", w.State)
		d.add("URL", w.HTMLURL)
		return d.print(printer)
	})
}

func runActionWorkflowEnable(cmd *cobra.Command, args []string) error {
	return setWorkflowState(cmd, args, "enable")
}

func runActionWorkflowDisable(cmd *cobra.Command, args []string) error {
	return setWorkflowState(cmd, args, "disable")
}

func setWorkflowState(cmd *cobra.Command, args []string, action string) error {
	owner, repo, args, err := repoTarget(cmd, args, 1)
	if err != nil {
		return err
	}
	workflow := args[1]

	if dryRunf(cmd, "Would %s workflow %s in %s/%s", action, workflow, owner, repo) {
		return nil
	}

	path := workflowPath(owner, repo, workflow) + "/" + action
	if _, err := apiRequest(cmd, http.MethodPut, path, nil); err != nil {
		return err
	}

	return emitMessage(cmd, okMessage("workflow "+action+"d"),
		"%sd workflow %s in %s/%s", strings.ToUpper(action[:1])+action[1:], workflow, owner, repo)
}
