package cmd

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func safetyCommand(t *testing.T, handler http.HandlerFunc) *cobra.Command {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/version" {
			fmt.Fprint(w, `{"version":"1.25.0"}`)
			return
		}
		handler(w, r)
	}))
	t.Cleanup(srv.Close)
	previousConfig, previousServer := configPath, serverName
	configPath = filepath.Join(t.TempDir(), "config.ini")
	serverName = "example"
	t.Cleanup(func() { configPath, serverName = previousConfig, previousServer })
	if err := os.WriteFile(configPath, []byte("[example]\nurl = "+srv.URL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().String("format", "text", "")
	cmd.Flags().String("method", "merge", "")
	cmd.Flags().Bool("delete-branch", false, "")
	cmd.Flags().String("state", "approve", "")
	cmd.Flags().String("body", "example", "")
	cmd.Flags().Int("comment-id", 0, "")
	cmd.Flags().String("file", "example.go", "")
	cmd.Flags().Int("line", 1, "")
	return cmd
}

func safetyOutput(t *testing.T, run func() error) (string, error) {
	t.Helper()
	out, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer out.Close()
	previous := os.Stdout
	os.Stdout = out
	defer func() { os.Stdout = previous }()
	runErr := run()
	os.Stdout = previous
	data, err := os.ReadFile(out.Name())
	if err != nil {
		t.Fatal(err)
	}
	return string(data), runErr
}

func TestPRMutationsRespectDryRun(t *testing.T) {
	tests := []struct {
		name   string
		run    func(*cobra.Command, []string) error
		args   []string
		inline bool
	}{
		{name: "close", run: runPRClose, args: []string{"owner/repo", "1"}},
		{name: "reopen", run: runPRReopen, args: []string{"owner/repo", "1"}},
		{name: "reply", run: runPRReply, args: []string{"owner/repo", "1"}},
		{name: "inline reply", run: runPRReply, args: []string{"owner/repo", "1"}, inline: true},
		{name: "review submit", run: runPRReviewSubmit, args: []string{"owner/repo", "1"}},
		{name: "review delete", run: runPRReviewDelete, args: []string{"owner/repo", "1", "2"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			requests := 0
			cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method == http.MethodDelete {
					w.WriteHeader(http.StatusNoContent)
				} else {
					fmt.Fprint(w, `{"id":2,"number":1,"state":"APPROVED"}`)
				}
			})
			if err := cmd.Flags().Set("dry-run", "true"); err != nil {
				t.Fatal(err)
			}
			if tt.inline {
				if err := cmd.Flags().Set("comment-id", "2"); err != nil {
					t.Fatal(err)
				}
			}
			_, err := safetyOutput(t, func() error { return tt.run(cmd, tt.args) })
			if err != nil {
				t.Fatal(err)
			}
			if requests != 0 {
				t.Fatalf("dry run sent %d resource requests", requests)
			}
		})
	}
}

func TestPRMergeRequiresConfirmation(t *testing.T) {
	for _, status := range []int{200, 403, 404, 405, 409, 500, 204} {
		t.Run(fmt.Sprint(status), func(t *testing.T) {
			requests := 0
			cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) {
				requests++
				if r.Method != http.MethodPost || r.URL.Path != "/api/v1/repos/owner/repo/pulls/1/merge" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
				}
				w.WriteHeader(status)
				if status != http.StatusNoContent {
					fmt.Fprint(w, `{"message":"merge refused"}`)
				}
			})
			out, err := safetyOutput(t, func() error { return runPRMerge(cmd, []string{"owner/repo", "1"}) })
			want := 1
			switch status {
			case 200:
				want = 0
			case 403:
				want = 4
			case 404:
				want = 3
			}
			if got := errors.ExitCode(err); got != want {
				t.Errorf("exit = %d, want %d (error: %v)", got, want, err)
			}
			if requests != 1 {
				t.Errorf("merge requests = %d, want 1", requests)
			}
			if status == 200 {
				if !strings.Contains(out, "Merged PR #1") {
					t.Errorf("missing success: %q", out)
				}
			} else if out != "" {
				t.Errorf("failure wrote stdout: %q", out)
			}
		})
	}
}

func TestPRRejectsMalformedNumbers(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
		args []string
	}{
		{"get", runPRGet, []string{"owner/repo", "12oops"}},
		{"merge", runPRMerge, []string{"owner/repo", "12oops"}},
		{"close", runPRClose, []string{"owner/repo", "12oops"}},
		{"reopen", runPRReopen, []string{"owner/repo", "12oops"}},
		{"comments", runPRComments, []string{"owner/repo", "12oops"}},
		{"reviews", runPRReviews, []string{"owner/repo", "12oops"}},
		{"review comments PR", runPRReviewComments, []string{"owner/repo", "12oops", "2"}},
		{"review comments ID", runPRReviewComments, []string{"owner/repo", "1", "2oops"}},
		{"reply", runPRReply, []string{"owner/repo", "12oops"}},
		{"approve", runPRApprove, []string{"owner/repo", "12oops"}},
		{"reject", runPRReject, []string{"owner/repo", "12oops"}},
		{"dismiss PR", runPRDismiss, []string{"owner/repo", "12oops", "2"}},
		{"dismiss ID", runPRDismiss, []string{"owner/repo", "1", "2oops"}},
		{"diff", runPRDiff, []string{"owner/repo", "12oops"}},
		{"review submit", runPRReviewSubmit, []string{"owner/repo", "12oops"}},
		{"review delete PR", runPRReviewDelete, []string{"owner/repo", "12oops", "2"}},
		{"review delete ID", runPRReviewDelete, []string{"owner/repo", "1", "2oops"}},
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
