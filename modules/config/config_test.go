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
