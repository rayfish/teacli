package cmd

import (
	"encoding/json"
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

func TestRunBranchDelete(t *testing.T) {
	tests := []struct {
		name         string
		getStatus    int
		deleteStatus int
		format       string
		dryRun       bool
		wantCode     int
		wantError    string
		wantOutput   string
		wantGet      int
		wantDelete   int
	}{
		{name: "deleted", getStatus: 200, deleteStatus: 204, wantOutput: "Deleted branch 'feature'\n", wantGet: 1, wantDelete: 1},
		{name: "protected", getStatus: 200, deleteStatus: 403, wantCode: 4, wantError: "403 Forbidden", wantGet: 1, wantDelete: 1},
		{name: "missing", getStatus: 404, deleteStatus: 404, wantCode: 3, wantError: "branch not found: feature", wantGet: 1},
		{name: "removed after preflight", getStatus: 200, deleteStatus: 404, wantCode: 3, wantError: "404 Not Found", wantGet: 1, wantDelete: 1},
		{name: "unexpected success status", getStatus: 200, deleteStatus: 200, wantCode: 1, wantError: "200 OK", wantGet: 1, wantDelete: 1},
		{name: "json", getStatus: 200, deleteStatus: 204, format: "json", wantGet: 1, wantDelete: 1},
		{name: "json failure", getStatus: 200, deleteStatus: 403, format: "json", wantCode: 4, wantError: "403 Forbidden", wantGet: 1, wantDelete: 1},
		{name: "dry run", dryRun: true, wantOutput: "DRY-RUN: Would delete branch 'feature'\n"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gets, deletes int
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				if r.URL.Path == "/api/v1/version" && r.Method == http.MethodGet {
					fmt.Fprint(w, `{"version":"1.25.0"}`)
					return
				}
				if r.URL.Path != "/api/v1/repos/owner/repo/branches/feature" {
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
					return
				}
				switch r.Method {
				case http.MethodGet:
					gets++
					w.WriteHeader(tt.getStatus)
					if tt.getStatus == http.StatusOK {
						fmt.Fprint(w, `{"name":"feature"}`)
					} else {
						fmt.Fprint(w, `{"message":"not found"}`)
					}
				case http.MethodDelete:
					deletes++
					w.WriteHeader(tt.deleteStatus)
					if tt.deleteStatus != http.StatusNoContent {
						fmt.Fprint(w, `{"message":"branch protected"}`)
					}
				default:
					t.Errorf("unexpected method: %s", r.Method)
					w.WriteHeader(http.StatusMethodNotAllowed)
				}
			}))
			defer srv.Close()

			previousConfig, previousServer := configPath, serverName
			configPath = filepath.Join(t.TempDir(), "config.ini")
			serverName = "example"
			t.Cleanup(func() { configPath, serverName = previousConfig, previousServer })
			if err := os.WriteFile(configPath, []byte("[example]\nurl = "+srv.URL+"\n"), 0600); err != nil {
				t.Fatal(err)
			}
			cmd := &cobra.Command{}
			cmd.Flags().Bool("dry-run", tt.dryRun, "")
			format := tt.format
			if format == "" {
				format = "text"
			}
			cmd.Flags().String("format", format, "")

			out, err := os.CreateTemp(t.TempDir(), "stdout")
			if err != nil {
				t.Fatal(err)
			}
			defer out.Close()
			previousStdout := os.Stdout
			os.Stdout = out
			defer func() { os.Stdout = previousStdout }()
			err = runBranchDelete(cmd, []string{"owner/repo", "feature"})
			os.Stdout = previousStdout
			if got := errors.ExitCode(err); got != tt.wantCode {
				t.Errorf("exit code = %d, want %d (error: %v)", got, tt.wantCode, err)
			}
			if tt.wantError != "" && (err == nil || !strings.Contains(err.Error(), tt.wantError)) {
				t.Errorf("error = %v, want %q", err, tt.wantError)
			}
			data, readErr := os.ReadFile(out.Name())
			if readErr != nil {
				t.Fatal(readErr)
			}
			if tt.format == "json" && tt.wantCode == 0 {
				var result map[string]string
				if err := json.Unmarshal(data, &result); err != nil {
					t.Fatal(err)
				}
				if len(result) != 2 || result["status"] != "deleted" || result["branch"] != "feature" {
					t.Errorf("unexpected JSON output: %s", data)
				}
			} else if string(data) != tt.wantOutput {
				t.Errorf("stdout = %q, want %q", data, tt.wantOutput)
			}
			if gets != tt.wantGet || deletes != tt.wantDelete {
				t.Errorf("GET/DELETE calls = %d/%d, want %d/%d", gets, deletes, tt.wantGet, tt.wantDelete)
			}
		})
	}
}
