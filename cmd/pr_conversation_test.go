package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func TestPRConversationResolution(t *testing.T) {
	actions := []struct {
		name   string
		status string
		run    func(*cobra.Command, []string) error
	}{
		{"resolve", "resolved", runPRResolve},
		{"unresolve", "unresolved", runPRUnresolve},
	}
	for _, action := range actions {
		for _, status := range []int{204, 200, 302, 400, 403, 404, 405, 500} {
			t.Run(fmt.Sprintf("%s/%d", action.name, status), func(t *testing.T) {
				requests := 0
				cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) {
					requests++
					wantPath := "/api/v1/repos/owner/repo/pulls/comments/7/" + action.name
					if r.Method != http.MethodPost || r.URL.Path != wantPath {
						t.Errorf("request = %s %s, want POST %s", r.Method, r.URL.Path, wantPath)
					}
					w.WriteHeader(status)
					if status != http.StatusNoContent {
						fmt.Fprint(w, `{"message":"request refused"}`)
					}
				})
				if err := cmd.Flags().Set("format", "json"); err != nil {
					t.Fatal(err)
				}
				out, err := safetyOutput(t, func() error { return action.run(cmd, []string{"owner/repo", "7"}) })
				if requests != 1 {
					t.Fatalf("sent %d requests, want 1", requests)
				}
				wantCode := errors.ExitGeneralError
				switch status {
				case 204:
					wantCode = errors.ExitSuccess
				case 400:
					wantCode = errors.ExitValidationError
				case 403:
					wantCode = errors.ExitPermissionDenied
				case 404:
					wantCode = errors.ExitNotFound
				}
				if errors.ExitCode(err) != wantCode {
					t.Fatalf("error = %v, want exit %d", err, wantCode)
				}
				if status == 204 {
					var result struct {
						Status     string `json:"status"`
						Comment    int64  `json:"comment"`
						Repository string `json:"repository"`
					}
					if err := json.Unmarshal([]byte(out), &result); err != nil {
						t.Fatal(err)
					}
					if result.Status != action.status || result.Comment != 7 || result.Repository != "owner/repo" {
						t.Fatalf("unexpected result: %+v", result)
					}
				} else if out != "" {
					t.Fatalf("failed request printed success: %s", out)
				}
				if (status == 404 || status == 405) && !strings.Contains(err.Error(), "older servers require the web UI") {
					t.Fatalf("missing compatibility diagnostic: %v", err)
				}
			})
		}
	}
}

func TestPRConversationRejectsInvalidCommentIDs(t *testing.T) {
	for _, id := range []string{"0", "-1", "7x", "99999999999999999999999"} {
		t.Run(id, func(t *testing.T) {
			cmd := safetyCommand(t, func(w http.ResponseWriter, r *http.Request) {
				t.Error("invalid ID sent a request")
			})
			if err := runPRResolve(cmd, []string{"owner/repo", id}); errors.ExitCode(err) != errors.ExitValidationError {
				t.Fatalf("error = %v, want validation error", err)
			}
		})
	}
}
