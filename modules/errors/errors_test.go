package errors

import (
	"fmt"
	"net/http"
	"testing"

	"code.gitea.io/sdk/gitea"
)

func resp(status int) *gitea.Response {
	return &gitea.Response{Response: &http.Response{StatusCode: status}}
}

func TestFromGiteaMapsStatusToExitCode(t *testing.T) {
	cases := []struct {
		status int
		want   int
	}{
		{http.StatusNotFound, ExitNotFound},
		{http.StatusForbidden, ExitPermissionDenied},
		{http.StatusUnauthorized, ExitPermissionDenied},
		{http.StatusUnprocessableEntity, ExitValidationError},
		{http.StatusInternalServerError, ExitGeneralError},
	}
	for _, c := range cases {
		got := FromGitea(resp(c.status), fmt.Errorf("boom"))
		if got.Code != c.want {
			t.Errorf("status %d -> code %d, want %d", c.status, got.Code, c.want)
		}
	}
}

func TestFromGiteaNilResponseIsGeneral(t *testing.T) {
	got := FromGitea(nil, fmt.Errorf("network down"))
	if got.Code != ExitGeneralError {
		t.Errorf("nil response -> code %d, want %d", got.Code, ExitGeneralError)
	}
}

func TestMessageIncludesDetails(t *testing.T) {
	err := NewValidationError("invalid --state", map[string]interface{}{
		"value": "bogus",
		"valid": []string{"open", "closed", "all"},
	})

	got := Message(err)
	want := "invalid --state (valid: open, closed, all, value: bogus)"
	if got != want {
		t.Errorf("Message() = %q, want %q", got, want)
	}
}

func TestMessageWithoutDetails(t *testing.T) {
	if got := Message(NewGeneralError("network down")); got != "network down" {
		t.Errorf("Message() = %q, want \"network down\"", got)
	}
}

func TestMessageOnPlainError(t *testing.T) {
	if got := Message(fmt.Errorf("unknown command \"nope\"")); got != `unknown command "nope"` {
		t.Errorf("Message() = %q, want the raw error text", got)
	}
}

func TestMessageIsDeterministic(t *testing.T) {
	err := NewValidationError("boom", map[string]interface{}{
		"a": "1", "b": "2", "c": "3", "d": "4", "e": "5",
	})

	first := Message(err)
	for i := 0; i < 20; i++ {
		if got := Message(err); got != first {
			t.Fatalf("Message() varies between calls: %q then %q", first, got)
		}
	}
}

func TestExitCode(t *testing.T) {
	if got := ExitCode(NewNotFoundError("release", "v9")); got != ExitNotFound {
		t.Errorf("ExitCode(CLIError) = %d, want %d", got, ExitNotFound)
	}
	if got := ExitCode(fmt.Errorf("plain")); got != ExitGeneralError {
		t.Errorf("ExitCode(plain) = %d, want %d", got, ExitGeneralError)
	}
	if got := ExitCode(nil); got != ExitSuccess {
		t.Errorf("ExitCode(nil) = %d, want %d", got, ExitSuccess)
	}
}
