package cmd

import (
	"testing"

	"code.gitea.io/sdk/gitea"
	"github.com/rayfish/teacli/modules/errors"
)

func TestParseDuration(t *testing.T) {
	tests := []struct {
		in   string
		want int64
	}{
		{"3600", 3600},
		{"90m", 5400},
		{"1h30m", 5400},
		{"2h", 7200},
	}
	for _, tt := range tests {
		got, err := parseDuration(tt.in)
		if err != nil {
			t.Errorf("parseDuration(%q): %v", tt.in, err)
			continue
		}
		if got != tt.want {
			t.Errorf("parseDuration(%q) = %d, want %d", tt.in, got, tt.want)
		}
	}

	for _, bad := range []string{"", "later", "0", "-5m"} {
		if _, err := parseDuration(bad); err == nil {
			t.Errorf("parseDuration(%q) should have failed", bad)
		}
	}
}

func TestFormatSeconds(t *testing.T) {
	tests := map[int64]string{
		30:   "30s",
		5400: "1h30m",
		3600: "1h",
	}
	for in, want := range tests {
		if got := formatSeconds(in); got != want {
			t.Errorf("formatSeconds(%d) = %q, want %q", in, got, want)
		}
	}
}

func TestRepoArgReportsValidationError(t *testing.T) {
	owner, repo, err := repoArg("acme/widgets")
	if err != nil {
		t.Fatalf("repoArg: %v", err)
	}
	if owner != "acme" || repo != "widgets" {
		t.Errorf("repoArg = (%q, %q), want (acme, widgets)", owner, repo)
	}

	_, _, err = repoArg("noslash")
	if err == nil {
		t.Fatal("expected an error for a malformed repo argument")
	}
	if code := errors.ExitCode(err); code != errors.ExitValidationError {
		t.Errorf("exit code = %d, want %d", code, errors.ExitValidationError)
	}
}

func TestInt64ArgReportsValidationError(t *testing.T) {
	if _, err := int64Arg("12", "issue number"); err != nil {
		t.Fatalf("int64Arg: %v", err)
	}
	err := func() error { _, e := int64Arg("abc", "issue number"); return e }()
	if err == nil {
		t.Fatal("expected an error for a non-numeric argument")
	}
	if code := errors.ExitCode(err); code != errors.ExitValidationError {
		t.Errorf("exit code = %d, want %d", code, errors.ExitValidationError)
	}
}

func TestParseState(t *testing.T) {
	for in, want := range map[string]gitea.StateType{
		"":       gitea.StateOpen,
		"open":   gitea.StateOpen,
		"closed": gitea.StateClosed,
		"all":    gitea.StateAll,
	} {
		got, err := parseState(in)
		if err != nil {
			t.Errorf("parseState(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("parseState(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := parseState("nonsense"); err == nil {
		t.Error("expected an error for an unknown state")
	}
}

func TestParseAccessModeRejectsUnknown(t *testing.T) {
	// Gitea silently downgrades an unrecognised mode to "none", so teacli has
	// to reject it up front rather than granting less access than asked for.
	if _, err := parseAccessMode("writeable"); err == nil {
		t.Fatal("expected an error for an unknown permission")
	}
	got, err := parseAccessMode("write")
	if err != nil {
		t.Fatalf("parseAccessMode: %v", err)
	}
	if got != gitea.AccessModeWrite {
		t.Errorf("parseAccessMode = %q, want %q", got, gitea.AccessModeWrite)
	}
}

func TestParseStatusStateRejectsUnknown(t *testing.T) {
	if _, err := parseStatusState("green"); err == nil {
		t.Fatal("expected an error for an unknown status state")
	}
	got, err := parseStatusState("success")
	if err != nil {
		t.Fatalf("parseStatusState: %v", err)
	}
	if got != gitea.StatusSuccess {
		t.Errorf("parseStatusState = %q, want %q", got, gitea.StatusSuccess)
	}
}

func TestParseNotifyStatuses(t *testing.T) {
	got, err := parseNotifyStatuses([]string{"unread", "pinned"})
	if err != nil {
		t.Fatalf("parseNotifyStatuses: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("got %d statuses, want 2", len(got))
	}
	if _, err := parseNotifyStatuses([]string{"seen"}); err == nil {
		t.Error("expected an error for an unknown status")
	}
}
