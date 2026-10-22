package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rayfish/teacli/modules/config"
	"github.com/spf13/cobra"
)

func TestAuthMutationsDryRunPreservesConfig(t *testing.T) {
	tests := []struct {
		name string
		run  func(*cobra.Command, []string) error
		args []string
	}{
		{"login", runAuthLogin, []string{"https://new.example.com"}},
		{"logout", runAuthLogout, []string{"custom"}},
		{"logout default", runAuthLogout, nil},
		{"set token", runConfigSet, []string{"custom", "token", "replacement-token"}},
		{"set URL", runConfigSet, []string{"custom", "url", "https://new.example.com"}},
		{"unset", runConfigUnset, []string{"custom"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldPath := configPath
			configPath = filepath.Join(t.TempDir(), "config.ini")
			t.Cleanup(func() { configPath = oldPath })
			original := []byte("[global]\ndefault_server = custom\n[custom]\nurl = https://forge.example.com\ntoken = fake-token\n")
			if err := os.WriteFile(configPath, original, 0600); err != nil {
				t.Fatal(err)
			}
			cmd := authTestCommand(true)
			if err := tt.run(cmd, tt.args); err != nil {
				t.Fatal(err)
			}
			got, err := os.ReadFile(configPath)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != string(original) {
				t.Error("dry run changed config")
			}
		})
	}
}

func TestAuthLoginDryRunDoesNotCreateConfig(t *testing.T) {
	oldPath := configPath
	configPath = filepath.Join(t.TempDir(), "nested", "config.ini")
	t.Cleanup(func() { configPath = oldPath })
	if err := runAuthLogin(authTestCommand(true), []string{"https://forge.example.com"}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("dry run created config: %v", err)
	}
}

func TestAuthLogoutDefaultUsesConfiguredName(t *testing.T) {
	oldPath := configPath
	configPath = filepath.Join(t.TempDir(), "config.ini")
	t.Cleanup(func() { configPath = oldPath })
	cfg, err := config.LoadFrom(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddServer("custom", "https://forge.example.com", "fake-token"); err != nil {
		t.Fatal(err)
	}
	if err := cfg.SetDefaultServer("custom"); err != nil {
		t.Fatal(err)
	}
	if err := runAuthLogout(authTestCommand(false), nil); err != nil {
		t.Fatal(err)
	}
	cfg, err = config.LoadFrom(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.ListServers()) != 0 {
		t.Error("default server remains after logout")
	}
}

func authTestCommand(dryRun bool) *cobra.Command {
	cmd := &cobra.Command{}
	cmd.Flags().Bool("dry-run", dryRun, "")
	cmd.Flags().String("format", "json", "")
	cmd.Flags().String("token", "fake-token", "")
	cmd.Flags().String("name", "new", "")
	cmd.Flags().Bool("default", true, "")
	return cmd
}
