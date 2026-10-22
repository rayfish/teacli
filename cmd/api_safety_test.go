package cmd

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func configureSafetyServer(t *testing.T, serverURL string) {
	t.Helper()
	previousConfig, previousServer := configPath, serverName
	configPath = filepath.Join(t.TempDir(), "config.ini")
	serverName = "example"
	t.Cleanup(func() { configPath, serverName = previousConfig, previousServer })
	if err := os.WriteFile(configPath, []byte("[example]\nurl = "+serverURL+"\ntoken = placeholder-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
}

func apiSafetyCommand(out string, paginate bool) *cobra.Command {
	cmd := &cobra.Command{}
	for _, flag := range []string{"method", "input", "out", "format"} {
		cmd.Flags().String(flag, "", "")
	}
	for _, flag := range []string{"field", "raw-field", "header"} {
		cmd.Flags().StringArray(flag, nil, "")
	}
	cmd.Flags().Bool("include", false, "")
	cmd.Flags().Bool("paginate", paginate, "")
	cmd.Flags().Bool("dry-run", false, "")
	cmd.Flags().Int("per-page", 0, "")
	cmd.Flags().Set("out", out)
	return cmd
}

func TestAPIOutputPreservesExistingFileOnFailure(t *testing.T) {
	for _, paginate := range []bool{false, true} {
		t.Run(fmt.Sprint(paginate), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusForbidden)
				fmt.Fprint(w, `{"message":"forbidden"}`)
			}))
			defer srv.Close()
			configureSafetyServer(t, srv.URL)
			out := filepath.Join(t.TempDir(), "output.json")
			if err := os.WriteFile(out, []byte("prior content"), 0600); err != nil {
				t.Fatal(err)
			}
			err := runAPI(apiSafetyCommand(out, paginate), []string{"items"})
			if errors.ExitCode(err) != errors.ExitPermissionDenied {
				t.Fatalf("expected permission error, got %v", err)
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != "prior content" {
				t.Fatalf("prior output lost: %q, %v", data, err)
			}
		})
	}
}

func TestAPIOutputSuccessfulReplacement(t *testing.T) {
	for _, body := range []string{"new content", ""} {
		t.Run(body, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, body)
			}))
			defer srv.Close()
			configureSafetyServer(t, srv.URL)
			out := filepath.Join(t.TempDir(), "output.json")
			if err := os.WriteFile(out, []byte("prior content"), 0600); err != nil {
				t.Fatal(err)
			}
			if err := runAPI(apiSafetyCommand(out, false), []string{"items"}); err != nil {
				t.Fatal(err)
			}
			want := body
			if body != "" {
				want += "\n"
			}
			data, err := os.ReadFile(out)
			if err != nil || string(data) != want {
				t.Fatalf("output = %q, %v; want %q", data, err, want)
			}
		})
	}
}

func TestAPIAuthStaysOnConfiguredOrigin(t *testing.T) {
	var foreignAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "download")
	}))
	defer foreign.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "token placeholder-token" {
			t.Errorf("configured origin auth = %q", got)
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, foreign.URL, http.StatusFound)
			return
		}
		fmt.Fprint(w, "same origin")
	}))
	defer srv.Close()
	configureSafetyServer(t, srv.URL)
	for _, endpoint := range []string{srv.URL + "/download", foreign.URL, srv.URL + "/redirect"} {
		t.Run(endpoint, func(t *testing.T) {
			foreignAuth = ""
			resp, err := apiDo(&cobra.Command{}, http.MethodGet, endpoint, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()
			if _, err := io.Copy(io.Discard, resp.Body); err != nil {
				t.Fatal(err)
			}
			if foreignAuth != "" {
				t.Errorf("foreign origin received credentials: %q", foreignAuth)
			}
		})
	}
}

func TestAPIOutputPreservesSpecialDestination(t *testing.T) {
	out := filepath.Join(t.TempDir(), "sink")
	if err := os.Symlink(os.DevNull, out); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if err := writeAPIOutput(out, []byte("discarded")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(out)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("special destination replaced: %v, %v", info, err)
	}
}

func TestSameOriginIncludesDefaultPorts(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"https://forge.example:443", "https://forge.example/download", true},
		{"http://forge.example:80", "http://FORGE.example/download", true},
		{"https://[::1]:443", "https://[::1]/download", true},
		{"https://forge.example", "https://forge.example:444", false},
		{"https://forge.example", "https://other.example", false},
		{"https://forge.example", "https://sub.forge.example", false},
		{"https://forge.example", "http://forge.example", false},
	}
	for _, tt := range tests {
		a, err := url.Parse(tt.a)
		if err != nil {
			t.Fatal(err)
		}
		b, err := url.Parse(tt.b)
		if err != nil {
			t.Fatal(err)
		}
		if got := sameOrigin(a, b); got != tt.want {
			t.Errorf("sameOrigin(%q, %q) = %t, want %t", tt.a, tt.b, got, tt.want)
		}
	}
}
