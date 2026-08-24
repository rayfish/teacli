package cmd

import (
	"encoding/json"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
)

func TestAPIURL(t *testing.T) {
	tests := []struct {
		base, path, want string
	}{
		{"http://git.example", "repos/o/r", "http://git.example/api/v1/repos/o/r"},
		{"http://git.example/", "/repos/o/r", "http://git.example/api/v1/repos/o/r"},
		{"http://git.example", "/api/v1/version", "http://git.example/api/v1/version"},
		{"http://git.example", "/api/v2/thing", "http://git.example/api/v2/thing"},
		{"http://git.example", "https://other/x", "https://other/x"},
	}
	for _, tt := range tests {
		if got := apiURL(tt.base, tt.path); got != tt.want {
			t.Errorf("apiURL(%q, %q) = %q, want %q", tt.base, tt.path, got, tt.want)
		}
	}
}

func TestAPIBodyFields(t *testing.T) {
	// --field always sends a string; --raw-field keeps the JSON type. Mixing
	// them up silently sends the wrong type, so pin the distinction down.
	raw, err := apiBody("", []string{"title=5"}, []string{"labels=[1,2]", "draft=true", "ms=null"})
	if err != nil {
		t.Fatalf("apiBody: %v", err)
	}

	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got["title"] != "5" {
		t.Errorf("title = %#v, want the string \"5\"", got["title"])
	}
	if got["draft"] != true {
		t.Errorf("draft = %#v, want true", got["draft"])
	}
	if got["ms"] != nil {
		t.Errorf("ms = %#v, want nil", got["ms"])
	}
	labels, ok := got["labels"].([]any)
	if !ok || len(labels) != 2 {
		t.Errorf("labels = %#v, want a 2-element array", got["labels"])
	}
}

func TestAPIBodyEmpty(t *testing.T) {
	raw, err := apiBody("", nil, nil)
	if err != nil {
		t.Fatalf("apiBody: %v", err)
	}
	if raw != nil {
		t.Errorf("apiBody with no input = %q, want nil so the method stays GET", raw)
	}
}

func TestAPIBodyRejectsInvalidJSON(t *testing.T) {
	_, err := apiBody("", nil, []string{"state=closed"})
	if err == nil {
		t.Fatal("expected an error for an unquoted --raw-field string")
	}
	if code := errors.ExitCode(err); code != errors.ExitValidationError {
		t.Errorf("exit code = %d, want %d", code, errors.ExitValidationError)
	}
}

func TestWithPageQuery(t *testing.T) {
	got, err := withPageQuery("repos/o/r/issues?state=open", 3, 20)
	if err != nil {
		t.Fatalf("withPageQuery: %v", err)
	}
	// The caller's own query string has to survive the added paging params.
	want := "repos/o/r/issues?limit=20&page=3&state=open"
	if got != want {
		t.Errorf("withPageQuery = %q, want %q", got, want)
	}
}

func TestSplitPair(t *testing.T) {
	k, v, err := splitPair("Accept:application/json", ":", "--header")
	if err != nil {
		t.Fatalf("splitPair: %v", err)
	}
	if k != "Accept" || v != "application/json" {
		t.Errorf("splitPair = (%q, %q), want (Accept, application/json)", k, v)
	}

	// A value containing the separator must not be split twice.
	_, v, err = splitPair("url=http://x/y", "=", "--field")
	if err != nil {
		t.Fatalf("splitPair: %v", err)
	}
	if v != "http://x/y" {
		t.Errorf("value = %q, want http://x/y", v)
	}

	if _, _, err := splitPair("nosep", "=", "--field"); err == nil {
		t.Error("expected an error when the separator is missing")
	}
}
