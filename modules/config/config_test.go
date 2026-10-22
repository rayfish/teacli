package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func TestRemoveServerPersistsAndKeepsDefaultUsable(t *testing.T) {
	for _, remaining := range []bool{false, true} {
		t.Run(fmt.Sprint(remaining), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.ini")
			cfg, err := LoadFrom(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.AddServer("custom", "https://forge.example.com", "removed-token"); err != nil {
				t.Fatal(err)
			}
			if err := cfg.SetDefaultServer("custom"); err != nil {
				t.Fatal(err)
			}
			if remaining {
				if err := cfg.AddServer("other", "https://other.example.com", "remaining-token"); err != nil {
					t.Fatal(err)
				}
			}
			if err := cfg.RemoveServer("custom"); err != nil {
				t.Fatal(err)
			}
			cfg, err = LoadFrom(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := cfg.GetServer("custom"); err == nil {
				t.Error("removed server survived reload")
			}
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(data), "removed-token") {
				t.Error("removed credential remains on disk")
			}
			server, err := cfg.GetDefaultServer()
			if remaining {
				if err != nil || server.URL != "https://other.example.com" {
					t.Fatalf("remaining default = %v, %v", server, err)
				}
			} else if err == nil {
				t.Error("empty config has a default server")
			}
		})
	}
}

func TestSaveRestrictsCredentialPermissions(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprint(existing), func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "nested", "config.ini")
			if existing {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, nil, 0666); err != nil {
					t.Fatal(err)
				}
				if err := os.Chmod(path, 0666); err != nil {
					t.Fatal(err)
				}
			}
			cfg, err := LoadFrom(path)
			if err != nil {
				t.Fatal(err)
			}
			if err := cfg.AddServer("example", "https://forge.example.com", "fake-token"); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			if got := info.Mode().Perm(); got != 0600 {
				t.Errorf("config mode = %o, want 600", got)
			}
		})
	}
}

func TestSaveReportsWriteFailuresAndCleansTemporaryFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.ini")
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	// A directory at the destination makes replacement fail on every user,
	// including when tests run with permission to write otherwise protected files.
	if err := os.Mkdir(path, 0700); err != nil {
		t.Fatal(err)
	}
	if err := cfg.AddServer("example", "https://forge.example.com", "fake-token"); err == nil {
		t.Fatal("saving to a directory succeeded")
	}
	leftovers, err := filepath.Glob(filepath.Join(filepath.Dir(path), ".teacli-config-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(leftovers) != 0 {
		t.Errorf("temporary credential files remain: %d", len(leftovers))
	}
}

func TestSavePreservesConfigSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "credentials.ini")
	if err := os.WriteFile(target, []byte("[example]\nurl = https://forge.example.com\ntoken = removed-token\n"), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.ini")
	if err := os.Symlink(target, path); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	cfg, err := LoadFrom(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := cfg.RemoveServer("example"); err != nil {
		t.Fatal(err)
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("config symlink replaced: %v, %v", info, err)
	}
	data, err := os.ReadFile(target)
	if err != nil || strings.Contains(string(data), "removed-token") {
		t.Fatalf("credential remains in symlink target: %v", err)
	}
}
