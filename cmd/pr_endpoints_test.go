package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/rayfish/teacli/modules/gitremote"
	"github.com/spf13/cobra"
)

func endpointCommand(t *testing.T, handler http.HandlerFunc) *cobra.Command {
	t.Helper()
	safetyCommand(t, handler)
	previous := inlineComments
	inlineComments = nil
	t.Cleanup(func() { inlineComments = previous })
	cmd := &cobra.Command{}
	cmd.Flags().String("format", "json", "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().String("style", "merge", "")
	cmd.Flags().String("body", "example body", "")
	cmd.Flags().String("body-file", "", "")
	cmd.Flags().String("file", "example.go", "")
	cmd.Flags().Int64("line", 5, "")
	cmd.Flags().Bool("old", false, "")
	cmd.Flags().Int64("extra-lines", 0, "")
	cmd.Flags().String("state", "comment", "")
	cmd.Flags().Int64("review-id", 0, "")
	cmd.Flags().String("commit", "", "")
	cmd.Flags().Int("comment-id", 0, "")
	return cmd
}

type prEndpointCase struct {
	name     string
	run      func(*cobra.Command, []string) error
	args     []string
	method   string
	suffix   string
	flags    map[string]string
	payload  map[string]any
	response string
	status   int
}

var prEndpointCases = []prEndpointCase{
	{name: "pinned", run: runPRPinned, args: nil, method: "GET", suffix: "/pinned", response: `[{"number":1,"title":"Pinned"}]`, status: 200},
	{name: "branch pair", run: runPRFind, args: []string{"main", "feature/topic"}, method: "GET", suffix: "/main/feature%2Ftopic", response: `{"number":1}`, status: 200},
	{name: "update branch", run: runPRUpdateBranch, args: []string{"1"}, method: "POST", suffix: "/1/update?style=rebase", flags: map[string]string{"style": "rebase"}, status: 200},
	{name: "cancel auto merge", run: runPRCancelAutoMerge, args: []string{"1"}, method: "DELETE", suffix: "/1/merge", status: 204},
	{name: "submit pending", run: runPRReviewSubmit, args: []string{"1"}, method: "POST", suffix: "/1/reviews/2", flags: map[string]string{"review-id": "2", "state": "approve"}, payload: map[string]any{"event": "APPROVED", "body": "example body"}, response: `{"id":2,"state":"APPROVED"}`, status: 200},
	{name: "get review comment", run: runPRReviewCommentGet, args: []string{"1", "2", "3"}, method: "GET", suffix: "/1/reviews/2/comments/3", response: `{"id":3,"body":"example body"}`, status: 200},
	{name: "create review comment", run: runPRReviewCommentCreate, args: []string{"1", "2"}, method: "POST", suffix: "/1/reviews/2/comments", flags: map[string]string{"old": "true", "extra-lines": "2"}, payload: map[string]any{"body": "example body", "path": "example.go", "new_position": float64(0), "old_position": float64(5), "extra_lines_count": float64(2)}, response: `{"id":3,"body":"example body"}`, status: 200},
	{name: "delete review comment", run: runPRReviewCommentDelete, args: []string{"1", "2", "3"}, method: "DELETE", suffix: "/1/reviews/2/comments/3", status: 204},
	{name: "reply to conversation", run: runPRReply, args: []string{"1"}, method: "POST", suffix: "/1/comments/3/replies", flags: map[string]string{"comment-id": "3", "file": "", "line": "0"}, payload: map[string]any{"body": "example body"}, response: `{"id":4,"body":"example body"}`, status: 201},
}

func TestMissingPREndpoints(t *testing.T) {
	for _, tc := range prEndpointCases {
		for _, inferred := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/inferred=%t", tc.name, inferred), func(t *testing.T) {
				stubDetect(t, &gitremote.Detected{Server: "example", Owner: "owner", Repo: "repo"}, nil)
				requests := 0
				cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) {
					requests++
					want := "/api/v1/repos/owner/repo/pulls" + tc.suffix
					if r.Method != tc.method || r.URL.RequestURI() != want {
						t.Errorf("request = %s %s, want %s %s", r.Method, r.URL.RequestURI(), tc.method, want)
					}
					if tc.payload != nil {
						var body map[string]any
						if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
							t.Fatal(err)
						}
						for k, v := range tc.payload {
							if body[k] != v {
								t.Errorf("body[%s] = %v, want %v", k, body[k], v)
							}
						}
					}
					w.WriteHeader(tc.status)
					fmt.Fprint(w, tc.response)
				})
				for name, value := range tc.flags {
					if err := cmd.Flags().Set(name, value); err != nil {
						t.Fatal(err)
					}
				}
				args := append([]string(nil), tc.args...)
				if !inferred {
					args = append([]string{"owner/repo"}, args...)
				}
				out, err := safetyOutput(t, func() error { return tc.run(cmd, args) })
				if err != nil {
					t.Fatal(err)
				}
				if requests != 1 {
					t.Fatalf("sent %d requests, want 1", requests)
				}
				if !json.Valid([]byte(out)) {
					t.Fatalf("invalid JSON output: %s", out)
				}
			})
		}
	}
}

func TestNewPREndpointErrors(t *testing.T) {
	for _, tc := range prEndpointCases {
		for _, status := range []int{403, 404, 405, 409, 422, 500} {
			t.Run(fmt.Sprintf("%s/%d", tc.name, status), func(t *testing.T) {
				cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(status)
					fmt.Fprint(w, `{"message":"request refused"}`)
				})
				for name, value := range tc.flags {
					if err := cmd.Flags().Set(name, value); err != nil {
						t.Fatal(err)
					}
				}
				args := append([]string{"owner/repo"}, tc.args...)
				out, err := safetyOutput(t, func() error { return tc.run(cmd, args) })
				wantCode := errors.ExitGeneralError
				switch status {
				case 403:
					wantCode = errors.ExitPermissionDenied
				case 404:
					wantCode = errors.ExitNotFound
				case 422:
					wantCode = errors.ExitValidationError
				}
				if errors.ExitCode(err) != wantCode {
					t.Fatalf("error = %v, want exit %d", err, wantCode)
				}
				if out != "" {
					t.Fatalf("failure printed success: %s", out)
				}
			})
		}
	}
}

func TestNewPRMutationsDryRun(t *testing.T) {
	for _, tc := range prEndpointCases {
		if tc.method == http.MethodGet {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) { t.Error("dry run sent a resource request") })
			for name, value := range tc.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			if err := cmd.Flags().Set("dry-run", "true"); err != nil {
				t.Fatal(err)
			}
			args := append([]string{"owner/repo"}, tc.args...)
			out, err := safetyOutput(t, func() error { return tc.run(cmd, args) })
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(out, "DRY-RUN") {
				t.Fatalf("missing dry run preview: %s", out)
			}
		})
	}
}

func TestInlineReplyUsesConversationID(t *testing.T) {
	cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/owner/repo/pulls/1/comments/7/replies" {
			t.Fatalf("created a separate review instead of replying: %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":8,"body":"reply"}`)
	})
	if err := cmd.Flags().Set("comment-id", "7"); err != nil {
		t.Fatal(err)
	}
	if _, err := safetyOutput(t, func() error { return runPRReply(cmd, []string{"owner/repo", "1"}) }); err != nil {
		t.Fatal(err)
	}
}

func TestCreatePendingReview(t *testing.T) {
	cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/owner/repo/pulls/1/reviews" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if body["event"] != "PENDING" || body["commit_id"] != "abc123" {
			t.Fatalf("unexpected payload: %v", body)
		}
		fmt.Fprint(w, `{"id":2,"state":"PENDING"}`)
	})
	for name, value := range map[string]string{"state": "pending", "commit": "abc123"} {
		if err := cmd.Flags().Set(name, value); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := safetyOutput(t, func() error { return runPRReviewSubmit(cmd, []string{"owner/repo", "1"}) }); err != nil {
		t.Fatal(err)
	}
}

func TestPREndpointInputValidation(t *testing.T) {
	tests := []struct {
		name   string
		run    func(*cobra.Command, []string) error
		args   []string
		flags  map[string]string
		inline bool
	}{
		{name: "bad PR number", run: runPRUpdateBranch, args: []string{"1x"}},
		{name: "bad style", run: runPRUpdateBranch, args: []string{"1"}, flags: map[string]string{"style": "squash"}},
		{name: "bad cancellation PR", run: runPRCancelAutoMerge, args: []string{"1x"}},
		{name: "bad review ID", run: runPRReviewCommentGet, args: []string{"1", "2x", "3"}},
		{name: "bad comment ID", run: runPRReviewCommentDelete, args: []string{"1", "2", "3x"}},
		{name: "zero comment ID", run: runPRReviewCommentGet, args: []string{"1", "2", "0"}},
		{name: "zero line", run: runPRReviewCommentCreate, args: []string{"1", "2"}, flags: map[string]string{"line": "0"}},
		{name: "negative extra lines", run: runPRReviewCommentCreate, args: []string{"1", "2"}, flags: map[string]string{"extra-lines": "-1"}},
		{name: "empty file", run: runPRReviewCommentCreate, args: []string{"1", "2"}, flags: map[string]string{"file": ""}},
		{name: "pending submission state", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"review-id": "2", "state": "pending"}},
		{name: "pending submission commit", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"review-id": "2", "commit": "abc123"}},
		{name: "pending submission comments", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"review-id": "2"}, inline: true},
		{name: "bad pending review ID", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"review-id": "-2"}},
		{name: "zero pending review ID", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"review-id": "0"}},
		{name: "invalid review state", run: runPRReviewSubmit, args: []string{"1"}, flags: map[string]string{"state": "invalid"}},
		{name: "negative inline reply ID", run: runPRReply, args: []string{"1"}, flags: map[string]string{"comment-id": "-1"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) { t.Error("invalid input sent a request") })
			for name, value := range tc.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			if tc.inline {
				inlineComments = []string{"example.go:5:example"}
			}
			args := append([]string{"owner/repo"}, tc.args...)
			if err := tc.run(cmd, args); errors.ExitCode(err) != errors.ExitValidationError {
				t.Fatalf("error = %v, want validation error", err)
			}
		})
	}
}

func TestPRMutationsRejectUnexpectedSuccessStatus(t *testing.T) {
	for _, tc := range prEndpointCases {
		if tc.method == http.MethodGet {
			continue
		}
		t.Run(tc.name, func(t *testing.T) {
			cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusAccepted); fmt.Fprint(w, `{}`) })
			for name, value := range tc.flags {
				if err := cmd.Flags().Set(name, value); err != nil {
					t.Fatal(err)
				}
			}
			args := append([]string{"owner/repo"}, tc.args...)
			out, err := safetyOutput(t, func() error { return tc.run(cmd, args) })
			if err == nil || out != "" {
				t.Fatalf("unconfirmed mutation reported success: %q, %v", out, err)
			}
		})
	}
}

func TestReviewCommentListShowsIDs(t *testing.T) {
	cmd := endpointCommand(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/repos/owner/repo/pulls/1/reviews/2/comments" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `[{"id":101,"body":"example comment","path":"example.go","position":5}]`)
	})
	if err := cmd.Flags().Set("format", "text"); err != nil {
		t.Fatal(err)
	}
	out, err := safetyOutput(t, func() error { return runPRReviewComments(cmd, []string{"owner/repo", "1", "2"}) })
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "ID") || !strings.Contains(out, "101") {
		t.Fatalf("comment IDs missing from output: %s", out)
	}
}
