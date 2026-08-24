package cmd

import (
	stderrors "errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/rayfish/teacli/internal/utils"
	"github.com/rayfish/teacli/modules/config"
	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/gitremote"

	"github.com/spf13/cobra"
)

// currentRepo resolves the repository of the working directory, at most once
// per process. Tests replace it.
var currentRepo = sync.OnceValues(detectCurrentRepo)

// inferredServerName records the server the detected repository belongs to, so
// getServer can talk to it rather than to the default one. A process runs one
// command, and it is set only when that command actually infers its repository,
// which keeps an explicit owner/repo argument from being redirected by whatever
// directory it was typed in.
var inferredServerName string

// repoTarget resolves the leading <owner>/<repo> argument, inferring it from
// the git remote of the current directory when it was left out. wantRest is
// the number of arguments the command takes after the repository. It returns
// args with the repository restored at index 0, so callers index the rest of
// their arguments as though it had been given.
func repoTarget(cmd *cobra.Command, args []string, wantRest int) (string, string, []string, error) {
	if len(args) > wantRest {
		owner, repo, err := repoArg(args[0])
		if err != nil {
			return "", "", nil, err
		}
		return owner, repo, args, nil
	}

	detected, err := currentRepo()
	if err != nil {
		return "", "", nil, inferenceError(cmd, err)
	}

	inferredServerName = detected.Server

	full := append([]string{detected.Owner + "/" + detected.Repo}, args...)
	return detected.Owner, detected.Repo, full, nil
}

// inferredServer reports the server the inferred repository belongs to, or ""
// when nothing was inferred.
func inferredServer() string {
	return inferredServerName
}

// inferenceError explains why the repository could not be worked out, and
// always says how to say it explicitly instead.
func inferenceError(cmd *cobra.Command, err error) error {
	hint := map[string]interface{}{"hint": "pass the repository: " + cmd.UseLine()}

	var mismatch *gitremote.HostMismatchError
	switch {
	case stderrors.Is(err, gitremote.ErrNoRepo):
		return errors.NewValidationError("not in a git repository", hint)
	case stderrors.Is(err, gitremote.ErrNoRemote):
		return errors.NewValidationError("cannot infer repository: this git repository has no usable remote", hint)
	case stderrors.As(err, &mismatch):
		return errors.NewValidationError(
			fmt.Sprintf("cannot infer repository: remote %s points at %s, which matches no configured server",
				mismatch.Remote, mismatch.Host), hint)
	}

	var cliErr *errors.CLIError
	if stderrors.As(err, &cliErr) {
		return cliErr
	}
	return errors.NewValidationError(fmt.Sprintf("cannot infer repository: %v", err), hint)
}

// detectCurrentRepo reads TEACLI_TARGET_REPO if it is set, and otherwise
// matches the git remotes of the working directory against the configured
// servers. The variable is named apart from install.sh's TEACLI_REPO, which
// says which repository to download teacli itself from.
func detectCurrentRepo() (*gitremote.Detected, error) {
	if env := os.Getenv("TEACLI_TARGET_REPO"); env != "" {
		owner, repo, err := utils.ParseRepoArg(env)
		if err != nil {
			return nil, errors.NewValidationError("invalid TEACLI_TARGET_REPO: "+err.Error(), nil)
		}
		return &gitremote.Detected{Remote: "TEACLI_TARGET_REPO", Owner: owner, Repo: repo}, nil
	}

	dir, err := os.Getwd()
	if err != nil {
		return nil, err
	}

	cfg, err := config.Load()
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to load config: %v", err))
	}
	return gitremote.Detect(dir, configuredServers(cfg))
}

// configuredServers lists the configured servers by name, sorted so that a
// host configured twice always resolves the same way.
func configuredServers(cfg *config.Config) []gitremote.Server {
	names := cfg.ListServers()
	sort.Strings(names)

	servers := make([]gitremote.Server, 0, len(names))
	for _, name := range names {
		server, err := cfg.GetServer(name)
		if err != nil || server == nil {
			continue
		}
		servers = append(servers, gitremote.Server{Name: name, URL: server.URL})
	}
	return servers
}

// repoTargetByArg resolves the leading [<owner>/<repo>] of a command whose
// argument count cannot say whether the repository was given, because what
// follows it is variadic or optional. It decides on the argument itself: a
// repository always contains a slash, and what can lead these commands
// instead, an issue number or a user, team or topic name, never does.
func repoTargetByArg(cmd *cobra.Command, args []string) (string, string, []string, error) {
	wantRest := len(args)
	if len(args) > 0 && strings.Contains(args[0], "/") {
		wantRest = len(args) - 1
	}
	return repoTarget(cmd, args, wantRest)
}

var repoCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the repository inferred from the current directory",
	Long: `Report which repository commands will act on when no <owner>/<repo> is given,
and which configured server it was matched against.`,
	Args: cobra.NoArgs,
	RunE: runRepoCurrent,
}

func init() {
	repoCmd.AddCommand(repoCurrentCmd)
}

// currentRepoFields describes a detected repository for both output formats.
func currentRepoFields(detected *gitremote.Detected) map[string]any {
	return map[string]any{
		"repository": detected.Owner + "/" + detected.Repo,
		"owner":      detected.Owner,
		"repo":       detected.Repo,
		"remote":     detected.Remote,
		"server":     detected.Server,
	}
}

func runRepoCurrent(cmd *cobra.Command, args []string) error {
	detected, err := currentRepo()
	if err != nil {
		return inferenceError(cmd, err)
	}

	fields := currentRepoFields(detected)
	printer := getPrinter(cmd)
	return printer.Emit(fields, func() error {
		d := &detail{}
		d.set("Repository", detected.Owner+"/"+detected.Repo)
		d.add("Remote", detected.Remote)
		d.add("Server", detected.Server)
		return d.print(printer)
	})
}
