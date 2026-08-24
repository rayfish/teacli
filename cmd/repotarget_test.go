package cmd

import (
	stderrors "errors"
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/gitremote"

	"github.com/spf13/cobra"
)

// stubDetect makes currentRepo return the given result for one test.
func stubDetect(t *testing.T, detected *gitremote.Detected, err error) {
	t.Helper()
	previous := currentRepo
	currentRepo = func() (*gitremote.Detected, error) { return detected, err }
	t.Cleanup(func() { currentRepo = previous })
}

var here = &gitremote.Detected{Remote: "origin", Server: "example", Owner: "acme", Repo: "widgets"}

func TestRepoTargetKeepsAnExplicitRepository(t *testing.T) {
	stubDetect(t, here, nil)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	owner, repo, args, err := repoTarget(cmd, []string{"other/thing", "42"}, 1)
	if err != nil {
		t.Fatalf("repoTarget: %v", err)
	}
	if owner != "other" || repo != "thing" {
		t.Errorf("repoTarget = %q/%q, want other/thing", owner, repo)
	}
	if len(args) != 2 || args[1] != "42" {
		t.Errorf("args = %v, want the arguments unchanged", args)
	}
}

func TestRepoTargetInfersWhenTheRepositoryIsOmitted(t *testing.T) {
	stubDetect(t, here, nil)
	cmd := &cobra.Command{Use: "get [<owner>/<repo>] <run-id>"}

	owner, repo, args, err := repoTarget(cmd, []string{"42"}, 1)
	if err != nil {
		t.Fatalf("repoTarget: %v", err)
	}
	if owner != "acme" || repo != "widgets" {
		t.Errorf("repoTarget = %q/%q, want acme/widgets", owner, repo)
	}
	if len(args) != 2 || args[0] != "acme/widgets" || args[1] != "42" {
		t.Errorf("args = %v, want the inferred repository restored at index 0", args)
	}
}

func TestRepoTargetInfersWithNoOtherArguments(t *testing.T) {
	stubDetect(t, here, nil)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	owner, repo, args, err := repoTarget(cmd, nil, 0)
	if err != nil {
		t.Fatalf("repoTarget: %v", err)
	}
	if owner != "acme" || repo != "widgets" || len(args) != 1 {
		t.Errorf("repoTarget = %q/%q args %v, want acme/widgets with one argument", owner, repo, args)
	}
}

func TestRepoTargetReportsAMalformedRepository(t *testing.T) {
	stubDetect(t, here, nil)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	_, _, _, err := repoTarget(cmd, []string{"noslash"}, 0)
	var cliErr *errors.CLIError
	if !asCLIError(err, &cliErr) || cliErr.Code != errors.ExitValidationError {
		t.Fatalf("repoTarget = %v, want a validation error", err)
	}
}

func TestRepoTargetOutsideAGitRepository(t *testing.T) {
	stubDetect(t, nil, gitremote.ErrNoRepo)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	_, _, _, err := repoTarget(cmd, nil, 0)
	var cliErr *errors.CLIError
	if !asCLIError(err, &cliErr) || cliErr.Code != errors.ExitValidationError {
		t.Fatalf("repoTarget = %v, want a validation error", err)
	}
	if !strings.Contains(cliErr.Message, "not in a git repository") {
		t.Errorf("message = %q, want it to say we are not in a git repository", cliErr.Message)
	}
}

func TestRepoTargetWhenTheRemoteMatchesNoServer(t *testing.T) {
	stubDetect(t, nil, &gitremote.HostMismatchError{Remote: "origin", Host: "github.com"})
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	_, _, _, err := repoTarget(cmd, nil, 0)
	var cliErr *errors.CLIError
	if !asCLIError(err, &cliErr) {
		t.Fatalf("repoTarget = %v, want a validation error", err)
	}
	if !strings.Contains(cliErr.Message, "github.com") {
		t.Errorf("message = %q, want it to name the unmatched host", cliErr.Message)
	}
}

func TestInferredRepositorySelectsItsServer(t *testing.T) {
	stubDetect(t, here, nil)
	resetInferredServer(t)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	if _, _, _, err := repoTarget(cmd, nil, 0); err != nil {
		t.Fatalf("repoTarget: %v", err)
	}
	if got := inferredServer(); got != "example" {
		t.Errorf("inferredServer() = %q, want example", got)
	}
}

func TestExplicitRepositoryDoesNotSelectAServer(t *testing.T) {
	stubDetect(t, here, nil)
	resetInferredServer(t)
	cmd := &cobra.Command{Use: "list [<owner>/<repo>]"}

	if _, _, _, err := repoTarget(cmd, []string{"other/thing"}, 0); err != nil {
		t.Fatalf("repoTarget: %v", err)
	}
	if got := inferredServer(); got != "" {
		t.Errorf("inferredServer() = %q, want it left unset", got)
	}
}

// asCLIError unwraps err into the CLI's structured error type.
func asCLIError(err error, target **errors.CLIError) bool {
	return stderrors.As(err, target)
}

// resetInferredServer clears the server an earlier test may have inferred.
func resetInferredServer(t *testing.T) {
	t.Helper()
	inferredServerName = ""
	t.Cleanup(func() { inferredServerName = "" })
}

func TestServerChoicePrefersTheFlagThenTheInferredServer(t *testing.T) {
	tests := []struct {
		flag, inferred, want string
	}{
		{"named", "example", "named"},
		{"named", "", "named"},
		{"", "example", "example"},
		{"", "", ""},
	}
	for _, tt := range tests {
		if got := serverChoice(tt.flag, tt.inferred); got != tt.want {
			t.Errorf("serverChoice(%q, %q) = %q, want %q", tt.flag, tt.inferred, got, tt.want)
		}
	}
}

func TestCurrentRepoFieldsDescribeTheDetectedRepository(t *testing.T) {
	fields := currentRepoFields(here)

	want := map[string]any{
		"repository": "acme/widgets",
		"owner":      "acme",
		"repo":       "widgets",
		"remote":     "origin",
		"server":     "example",
	}
	for key, value := range want {
		if fields[key] != value {
			t.Errorf("currentRepoFields()[%q] = %v, want %v", key, fields[key], value)
		}
	}
}

func TestCurrentRepoFieldsWithoutAMatchedServer(t *testing.T) {
	fields := currentRepoFields(&gitremote.Detected{Remote: "TEACLI_TARGET_REPO", Owner: "acme", Repo: "widgets"})

	if fields["server"] != "" {
		t.Errorf("server = %v, want it empty when nothing matched", fields["server"])
	}
	if fields["repository"] != "acme/widgets" {
		t.Errorf("repository = %v, want acme/widgets", fields["repository"])
	}
}
