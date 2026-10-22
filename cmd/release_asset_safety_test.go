package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func TestReleaseDownloadRejectsUnsafeDefaultFilename(t *testing.T) {
	dir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	work := filepath.Join(dir, "work")
	if err := os.Mkdir(work, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(work); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Error(err)
		}
	})
	target := filepath.Join(dir, "sentinel")
	for _, name := range []string{"../sentinel", target, `..\sentinel`, ".", "..", ""} {
		t.Run(name, func(t *testing.T) {
			if err := os.WriteFile(target, []byte("prior content"), 0600); err != nil {
				t.Fatal(err)
			}
			downloads := 0
			var srv *httptest.Server
			srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				switch r.URL.Path {
				case "/api/v1/version":
					fmt.Fprint(w, `{"version":"1.25.0"}`)
				case "/api/v1/repos/owner/repo/releases/tags/v1":
					fmt.Fprint(w, `{"id":1}`)
				case "/api/v1/repos/owner/repo/releases/1/assets/2":
					json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": name, "browser_download_url": srv.URL + "/download"})
				case "/download":
					downloads++
					fmt.Fprint(w, "replacement")
				default:
					t.Errorf("unexpected path: %s", r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()
			configureSafetyServer(t, srv.URL)
			cmd := &cobra.Command{}
			cmd.Flags().String("out", "", "")
			cmd.Flags().Bool("dry-run", false, "")
			err := runReleaseAssetDownload(cmd, []string{"owner/repo", "v1", "2"})
			if errors.ExitCode(err) != errors.ExitValidationError {
				t.Errorf("unsafe name should fail validation, got %v", err)
			}
			data, err := os.ReadFile(target)
			if err != nil || string(data) != "prior content" {
				t.Errorf("parent file changed: %q, %v", data, err)
			}
			if downloads != 0 {
				t.Errorf("unsafe name downloaded %d times", downloads)
			}
		})
	}
}

func TestReleaseDownloadForeignURLDoesNotReceiveToken(t *testing.T) {
	var foreignAuth string
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		foreignAuth = r.Header.Get("Authorization")
		fmt.Fprint(w, "release bytes")
	}))
	defer foreign.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.25.0"}`)
		case "/api/v1/repos/owner/repo/releases/tags/v1":
			fmt.Fprint(w, `{"id":1}`)
		case "/api/v1/repos/owner/repo/releases/1/assets/2":
			json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "asset", "browser_download_url": foreign.URL})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	configureSafetyServer(t, srv.URL)
	out := filepath.Join(t.TempDir(), "asset")
	cmd := &cobra.Command{}
	cmd.Flags().String("out", out, "")
	cmd.Flags().Bool("dry-run", false, "")
	if err := runReleaseAssetDownload(cmd, []string{"owner/repo", "v1", "2"}); err != nil {
		t.Fatal(err)
	}
	if foreignAuth != "" {
		t.Error("foreign download received configured credentials")
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "release bytes" {
		t.Errorf("download = %q, %v", data, err)
	}
}

func TestReleaseDownloadPreservesDestinationOnInterruptedTransfer(t *testing.T) {
	foreign := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "100")
		fmt.Fprint(w, "partial")
	}))
	defer foreign.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/v1/version":
			fmt.Fprint(w, `{"version":"1.25.0"}`)
		case "/api/v1/repos/owner/repo/releases/tags/v1":
			fmt.Fprint(w, `{"id":1}`)
		case "/api/v1/repos/owner/repo/releases/1/assets/2":
			json.NewEncoder(w).Encode(map[string]any{"id": 2, "name": "asset", "browser_download_url": foreign.URL})
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()
	configureSafetyServer(t, srv.URL)
	dir := t.TempDir()
	out := filepath.Join(dir, "asset")
	if err := os.WriteFile(out, []byte("prior content"), 0600); err != nil {
		t.Fatal(err)
	}
	cmd := &cobra.Command{}
	cmd.Flags().String("out", out, "")
	cmd.Flags().Bool("dry-run", false, "")
	output, err := safetyOutput(t, func() error {
		return runReleaseAssetDownload(cmd, []string{"owner/repo", "v1", "2"})
	})
	if errors.ExitCode(err) != errors.ExitGeneralError || output != "" {
		t.Errorf("incomplete download returned output %q, error %v", output, err)
	}
	data, err := os.ReadFile(out)
	if err != nil || string(data) != "prior content" {
		t.Errorf("prior destination lost: %q, %v", data, err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil || len(entries) != 1 {
		t.Errorf("temporary files remain: %v, %v", entries, err)
	}
}
