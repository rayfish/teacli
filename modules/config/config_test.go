package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetConfigPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	want := filepath.Join(home, ".config", "teacli", "teacli.ini")
	if got := getConfigPath(); got != want {
		t.Errorf("getConfigPath() = %q, want %q", got, want)
	}

	// The directory has to exist for the first Save to land.
	if _, err := os.Stat(filepath.Dir(want)); err != nil {
		t.Errorf("config directory was not created: %v", err)
	}
}

func TestLoadFromHonoursAnExplicitPath(t *testing.T) {
	// A path given on the command line wins over the default location, and the
	// default location is not touched.
	home := t.TempDir()
	t.Setenv("HOME", home)

	explicit := filepath.Join(t.TempDir(), "elsewhere.ini")
	body := "[global]\ndefault_server = other\n\n[other]\nurl = https://forge.example.com\ntoken = t\n"
	if err := os.WriteFile(explicit, []byte(body), 0600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFrom(explicit)
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}

	server, err := cfg.GetServer("other")
	if err != nil {
		t.Fatalf("GetServer: %v", err)
	}
	if server.URL != "https://forge.example.com" {
		t.Errorf("url = %q, want the one from the explicit file", server.URL)
	}
	if cfg.path != explicit {
		t.Errorf("path = %q, want %q", cfg.path, explicit)
	}
}

func TestLoadFromEmptyPathUsesTheDefault(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	cfg, err := LoadFrom("")
	if err != nil {
		t.Fatalf("LoadFrom: %v", err)
	}
	if want := filepath.Join(home, ".config", "teacli", "teacli.ini"); cfg.path != want {
		t.Errorf("path = %q, want %q", cfg.path, want)
	}
}
