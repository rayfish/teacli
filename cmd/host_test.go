package cmd

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rayfish/teacli/modules/errors"
	"github.com/spf13/cobra"
)

func TestHostSelection(t *testing.T) {
	const one = "[primary]\nurl = https://forge.example.com\n"
	const two = one + "[secondary]\nurl = https://other.example.com\n"
	tests := []struct {
		name    string
		config  string
		host    string
		want    string
		message string
	}{
		{name: "single instance", config: one, want: "https://forge.example.com"},
		{name: "no instances", message: "specify --host"},
		{name: "multiple instances", config: two, message: "specify --host"},
		{name: "default does not disambiguate", config: "[global]\ndefault_server = primary\n" + two, message: "specify --host"},
		{name: "explicit name", config: two, host: "secondary", want: "https://other.example.com"},
		{name: "explicit hostname", config: two, host: "other.example.com", want: "https://other.example.com"},
		{name: "explicit URL", config: two, host: "https://other.example.com/", want: "https://other.example.com"},
		{name: "unknown host", config: two, host: "missing", message: "unknown host"},
		{name: "ambiguous hostname", config: one + "[secondary]\nurl = https://forge.example.com/subpath\n", host: "forge.example.com", message: "matches multiple instances"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setHostConfig(t, tt.config, tt.host)
			server, err := getServer(&cobra.Command{})
			if tt.message != "" {
				if errors.ExitCode(err) != errors.ExitValidationError || !strings.Contains(err.Error(), tt.message) {
					t.Fatalf("getServer error = %v, want validation error containing %q", err, tt.message)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if server.URL != tt.want {
				t.Fatalf("server URL = %q, want %q", server.URL, tt.want)
			}
		})
	}
}

func setHostConfig(t *testing.T, body, host string) {
	t.Helper()
	previousPath, previousHost, previousServer := configPath, hostName, serverName
	t.Cleanup(func() { configPath, hostName, serverName = previousPath, previousHost, previousServer })
	configPath = filepath.Join(t.TempDir(), "config.ini")
	hostName = host
	serverName = ""
	if err := os.WriteFile(configPath, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}
}

func TestRepositoryDetectionRequiresHostWithMultipleInstances(t *testing.T) {
	t.Setenv("TEACLI_TARGET_REPO", "")
	setHostConfig(t, "[global]\ndefault_server = primary\n[primary]\nurl = https://forge.example.com\n[secondary]\nurl = https://other.example.com\n", "")
	dir := t.TempDir()
	for _, args := range [][]string{
		{"init", dir},
		{"-C", dir, "remote", "add", "origin", "https://forge.example.com/acme/widgets.git"},
		{"-C", dir, "remote", "add", "upstream", "https://other.example.com/acme/other.git"},
	} {
		if out, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Fatalf("git: %v: %s", err, out)
		}
	}
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previousDir); err != nil {
			t.Error(err)
		}
	})
	if _, err := detectCurrentRepo(); errors.ExitCode(err) != errors.ExitValidationError || !strings.Contains(err.Error(), "specify --host") {
		t.Fatalf("detectCurrentRepo error = %v, want --host required", err)
	}
	hostName = "secondary"
	detected, err := detectCurrentRepo()
	if err != nil {
		t.Fatal(err)
	}
	if detected.Server != "secondary" || detected.Repo != "other" {
		t.Fatalf("detected = %+v, want repository from selected instance", detected)
	}
}

func TestHostAndLegacyServerSelectors(t *testing.T) {
	setHostConfig(t, "[primary]\nurl = https://forge.example.com\n[secondary]\nurl = https://other.example.com\n", "")
	serverName = "secondary"
	server, err := getServer(&cobra.Command{})
	if err != nil || server.URL != "https://other.example.com" {
		t.Fatalf("legacy selector = %v, %v", server, err)
	}
	hostName = "primary"
	if _, err := getServer(&cobra.Command{}); errors.ExitCode(err) != errors.ExitValidationError {
		t.Fatalf("conflicting selectors: %v", err)
	}
}

func TestEnvironmentRepositoryRequiresHostWithMultipleInstances(t *testing.T) {
	setHostConfig(t, "[primary]\nurl = https://forge.example.com\n[secondary]\nurl = https://other.example.com\n", "")
	t.Setenv("TEACLI_TARGET_REPO", "acme/widgets")
	if _, err := detectCurrentRepo(); errors.ExitCode(err) != errors.ExitValidationError {
		t.Fatalf("missing host: %v", err)
	}
	hostName = "secondary"
	detected, err := detectCurrentRepo()
	if err != nil {
		t.Fatal(err)
	}
	if detected.Server != "secondary" || detected.Owner != "acme" || detected.Repo != "widgets" {
		t.Fatalf("unexpected repository: %+v", detected)
	}
}

func TestLogoutRequiresHostWithMultipleInstances(t *testing.T) {
	setHostConfig(t, "[global]\ndefault_server = primary\n[primary]\nurl = https://forge.example.com\n[secondary]\nurl = https://other.example.com\n", "")
	before, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := runAuthLogout(authTestCommand(false), nil); errors.ExitCode(err) != errors.ExitValidationError {
		t.Fatalf("missing host: %v", err)
	}
	after, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("logout without a host changed config")
	}
}
