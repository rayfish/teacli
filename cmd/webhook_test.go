package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func TestNumericArgumentsRejectMalformedIDs(t *testing.T) {
	var requests atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests.Add(1)
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Path == "/api/v1/version" {
			fmt.Fprint(w, `{"version":"1.25.0"}`)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()
	configureWebhookTestServer(t, srv.URL)

	commands := []struct {
		name string
		run  func(*cobra.Command, []string) error
	}{
		{"webhook get", runWebhookGet},
		{"webhook update", runWebhookUpdate},
		{"webhook delete", runWebhookDelete},
		{"webhook test", runWebhookTest},
		{"label get", runLabelGet},
		{"label update", runLabelUpdate},
		{"label delete", runLabelDelete},
		{"milestone get", runMilestoneGet},
		{"milestone update", runMilestoneUpdate},
		{"milestone close", runMilestoneClose},
		{"milestone reopen", runMilestoneReopen},
		{"milestone delete", runMilestoneDelete},
		{"pr comment delete", runPRCommentDelete},
	}
	for _, command := range commands {
		for _, id := range []string{"12oops", "oops", "9223372036854775808"} {
			t.Run(command.name+"/"+id, func(t *testing.T) {
				requests.Store(0)
				cmd := &cobra.Command{}
				cmd.Flags().Bool("dry-run", false, "")
				cmd.Flags().String("format", "text", "")
				var runErr error
				output := captureWebhookOutput(t, func() {
					runErr = command.run(cmd, []string{"owner/repo", id})
				})
				if code := errors.ExitCode(runErr); code != errors.ExitValidationError {
					t.Errorf("exit code = %d, want 2 (error: %v)", code, runErr)
				}
				if output != "" {
					t.Errorf("unexpected success output: %q", output)
				}
				if got := requests.Load(); got != 0 {
					t.Errorf("sent %d HTTP requests for an invalid id", got)
				}
			})
		}
	}
}

func TestWebhookMutationsHonorDryRun(t *testing.T) {
	var mutations atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == http.MethodGet && r.URL.Path == "/api/v1/version" {
			fmt.Fprint(w, `{"version":"1.25.0"}`)
			return
		}
		mutations.Add(1)
		fmt.Fprint(w, `{"id":12}`)
	}))
	defer srv.Close()
	configureWebhookTestServer(t, srv.URL)

	commands := []struct {
		name string
		run  func(*cobra.Command, []string) error
		args []string
	}{
		{"create", runWebhookCreate, []string{"owner/repo"}},
		{"update", runWebhookUpdate, []string{"owner/repo", "12"}},
		{"delete", runWebhookDelete, []string{"owner/repo", "12"}},
		{"test", runWebhookTest, []string{"owner/repo", "12"}},
	}
	for _, command := range commands {
		t.Run(command.name, func(t *testing.T) {
			mutations.Store(0)
			cmd := &cobra.Command{}
			cmd.Flags().Bool("dry-run", true, "")
			cmd.Flags().String("format", "text", "")
			cmd.Flags().String("url", "", "")
			cmd.Flags().String("events", "push", "")
			cmd.Flags().String("secret", "", "")
			cmd.Flags().String("branch-filter", "", "")
			cmd.Flags().Bool("active", true, "")
			cmd.Flags().Bool("inactive", false, "")
			if err := cmd.Flags().Set("url", "https://example.com/hook"); err != nil {
				t.Fatal(err)
			}
			var runErr error
			output := captureWebhookOutput(t, func() {
				runErr = command.run(cmd, command.args)
			})
			if runErr != nil {
				t.Fatal(runErr)
			}
			if !strings.HasPrefix(output, "DRY-RUN: Would ") {
				t.Errorf("output = %q, want dry-run preview", output)
			}
			if got := mutations.Load(); got != 0 {
				t.Errorf("dry run sent %d mutation requests", got)
			}
		})
	}
}

func configureWebhookTestServer(t *testing.T, serverURL string) {
	t.Helper()
	previousConfig, previousServer := configPath, serverName
	configPath = filepath.Join(t.TempDir(), "config.ini")
	serverName = "example"
	t.Cleanup(func() { configPath, serverName = previousConfig, previousServer })
	if err := os.WriteFile(configPath, []byte("[example]\nurl = "+serverURL+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func captureWebhookOutput(t *testing.T, run func()) string {
	t.Helper()
	output, err := os.CreateTemp(t.TempDir(), "stdout")
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	previous := os.Stdout
	os.Stdout = output
	defer func() { os.Stdout = previous }()
	run()
	os.Stdout = previous
	if _, err := output.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	data, err := io.ReadAll(output)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
