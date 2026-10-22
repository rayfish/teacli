package cmd

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func TestIssueStateRespectsDryRun(t *testing.T) {
	for name, run := range map[string]func(*cobra.Command, []string) error{"close": runIssueClose, "reopen": runIssueReopen} {
		t.Run(name, func(t *testing.T) {
			requests := 0
			cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) { requests++; fmt.Fprint(w, `{"number":1}`) })
			if err := cmd.Flags().Set("dry-run", "true"); err != nil {
				t.Fatal(err)
			}
			_, err := safetyOutput(t, func() error { return run(cmd, []string{"owner/repo", "1"}) })
			if err != nil {
				t.Fatal(err)
			}
			if requests != 0 {
				t.Errorf("dry run sent %d resource requests", requests)
			}
		})
	}
}

func TestIssueRejectsMalformedNumbers(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
		args []string
	}{
		{"get", runIssueGet, []string{"owner/repo", "12oops"}},
		{"close", runIssueClose, []string{"owner/repo", "12oops"}},
		{"reopen", runIssueReopen, []string{"owner/repo", "12oops"}},
		{"comments", runIssueCommentList, []string{"owner/repo", "12oops"}},
		{"comment create", runIssueCommentCreate, []string{"owner/repo", "12oops"}},
		{"comment delete", runIssueCommentDelete, []string{"owner/repo", "12oops"}},
		{"labels", runIssueLabelList, []string{"owner/repo", "12oops"}},
		{"label add", runIssueLabelAdd, []string{"owner/repo", "12oops", "1"}},
		{"label remove", runIssueLabelRemove, []string{"owner/repo", "12oops", "1"}},
		{"label clear", runIssueLabelClear, []string{"owner/repo", "12oops"}},
		{"assign", runIssueAssign, []string{"owner/repo", "12oops", "example"}},
		{"unassign", runIssueUnassign, []string{"owner/repo", "12oops", "example"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) { requests++; w.WriteHeader(http.StatusNotFound) })
			_, err := safetyOutput(t, func() error { return tt.run(cmd, tt.args) })
			if got := errors.ExitCode(err); got != 2 {
				t.Errorf("exit = %d, want 2 (error: %v)", got, err)
			}
			if requests != 0 {
				t.Errorf("invalid input sent %d requests", requests)
			}
		})
	}
}
