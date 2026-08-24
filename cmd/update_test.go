package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestAssetName(t *testing.T) {
	cases := []struct {
		goos, goarch, want string
	}{
		{"linux", "amd64", "teacli-linux-amd64"},
		{"linux", "arm64", "teacli-linux-arm64"},
		{"darwin", "arm64", "teacli-darwin-arm64"},
		{"windows", "amd64", "teacli-windows-amd64.exe"},
	}
	for _, c := range cases {
		if got := assetName(c.goos, c.goarch); got != c.want {
			t.Errorf("assetName(%q, %q) = %q, want %q", c.goos, c.goarch, got, c.want)
		}
	}
}

func TestNormalizeVersion(t *testing.T) {
	for in, want := range map[string]string{
		"v0.3.1":  "0.3.1",
		"0.3.1":   "0.3.1",
		" v1.0.0": "1.0.0",
	} {
		if got := normalizeVersion(in); got != want {
			t.Errorf("normalizeVersion(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestParseChecksums(t *testing.T) {
	// sha256sum output, including the "*name" binary-mode form and a blank line.
	data := []byte("abc123  teacli-linux-amd64\n\ndef456 *teacli-darwin-arm64\ngarbage\n")

	sums := parseChecksums(data)
	if sums["teacli-linux-amd64"] != "abc123" {
		t.Errorf("linux digest = %q, want abc123", sums["teacli-linux-amd64"])
	}
	if sums["teacli-darwin-arm64"] != "def456" {
		t.Errorf("darwin digest = %q, want def456", sums["teacli-darwin-arm64"])
	}
	if len(sums) != 2 {
		t.Errorf("parsed %d entries, want 2: %v", len(sums), sums)
	}
}

func TestVerifyChecksum(t *testing.T) {
	payload := []byte("binary contents")
	digest := sha256.Sum256(payload)
	good := hex.EncodeToString(digest[:])

	if err := verifyChecksum(map[string]string{"teacli-linux-amd64": good}, "teacli-linux-amd64", payload); err != nil {
		t.Errorf("matching digest rejected: %v", err)
	}

	// Uppercase digests must still match.
	upper := map[string]string{"teacli-linux-amd64": "ABCDEF"}
	if err := verifyChecksum(upper, "teacli-linux-amd64", payload); err == nil {
		t.Error("mismatched digest accepted")
	}

	if err := verifyChecksum(map[string]string{}, "teacli-linux-amd64", payload); err == nil {
		t.Error("missing entry accepted; an unlisted asset must not be installed")
	}
}

// releaseServer serves the release layout the workflow publishes.
func releaseServer(t *testing.T, tag string, assets map[string][]byte, wantToken string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if wantToken != "" && r.Header.Get("Authorization") != "token "+wantToken {
			w.WriteHeader(http.StatusUnauthorized)
			fmt.Fprint(w, `{"message":"token required"}`)
			return
		}

		if r.URL.Path == "/api/v1/repos/rayfish/teacli/releases/latest" {
			fmt.Fprintf(w, `{"tag_name":%q}`, tag)
			return
		}

		prefix := "/rayfish/teacli/releases/download/" + tag + "/"
		if name, ok := trimPrefix(r.URL.Path, prefix); ok {
			if body, found := assets[name]; found {
				w.Write(body)
				return
			}
		}

		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"not found"}`)
	}))
}

func trimPrefix(s, prefix string) (string, bool) {
	if len(s) >= len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):], true
	}
	return "", false
}

func TestUpdaterLatestTag(t *testing.T) {
	srv := releaseServer(t, "v9.9.9", nil, "")
	defer srv.Close()

	up := &updater{host: srv.URL, repo: "rayfish/teacli", client: srv.Client()}

	tag, err := up.latestTag()
	if err != nil {
		t.Fatalf("latestTag: %v", err)
	}
	if tag != "v9.9.9" {
		t.Errorf("tag = %q, want v9.9.9", tag)
	}
}

func TestUpdaterLatestTagOnMissingRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"message":"release not found"}`)
	}))
	defer srv.Close()

	up := &updater{host: srv.URL, repo: "rayfish/teacli", client: srv.Client()}

	if _, err := up.latestTag(); err == nil {
		t.Fatal("expected an error when no release exists")
	}
}

func TestUpdaterSendsToken(t *testing.T) {
	srv := releaseServer(t, "v1.0.0", map[string][]byte{"teacli-linux-amd64": []byte("payload")}, "secret-token")
	defer srv.Close()

	up := &updater{host: srv.URL, repo: "rayfish/teacli", token: "secret-token", client: srv.Client()}

	got, err := up.fetchAsset("v1.0.0", "teacli-linux-amd64")
	if err != nil {
		t.Fatalf("fetchAsset: %v", err)
	}
	if string(got) != "payload" {
		t.Errorf("asset = %q, want \"payload\"", got)
	}

	// Without the token the same server rejects the request.
	anon := &updater{host: srv.URL, repo: "rayfish/teacli", client: srv.Client()}
	if _, err := anon.fetchAsset("v1.0.0", "teacli-linux-amd64"); err == nil {
		t.Error("expected an auth failure without the token")
	}
}

func TestSameHost(t *testing.T) {
	if !sameHost("https://gitea.example.com", "https://gitea.example.com/") {
		t.Error("identical hosts reported as different")
	}
	if sameHost("https://gitea.example.com", "https://evil.example.com") {
		t.Error("different hosts reported as the same; the token would leak")
	}
}

func TestLatestEndpoint(t *testing.T) {
	// GitHub serves release metadata from api.github.com, not from the host the
	// assets are downloaded from.
	gh := &updater{host: defaultReleaseHost, repo: "rayfish/teacli"}
	want := "https://api.github.com/repos/rayfish/teacli/releases/latest"
	if got := gh.latestEndpoint(); got != want {
		t.Errorf("github endpoint = %q, want %q", got, want)
	}

	// A Gitea or Forgejo mirror answers on its own /api/v1, trailing slash or not.
	mirror := &updater{host: "https://gitea.example.com/", repo: "tools/teacli"}
	want = "https://gitea.example.com/api/v1/repos/tools/teacli/releases/latest"
	if got := mirror.latestEndpoint(); got != want {
		t.Errorf("mirror endpoint = %q, want %q", got, want)
	}
}

func TestIsGitHub(t *testing.T) {
	for host, want := range map[string]bool{
		"https://github.com":            true,
		"https://github.com/":           true,
		"https://GitHub.com":            true,
		"https://gitea.example.com":     false,
		"https://github.com.evil.test":  false,
		"https://raw.githubusercontent": false,
	} {
		if got := isGitHub(host); got != want {
			t.Errorf("isGitHub(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestReplaceExecutable(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the rename-aside path needs a real running image")
	}

	dir := t.TempDir()
	target := filepath.Join(dir, "teacli")
	if err := os.WriteFile(target, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}

	staged := filepath.Join(dir, ".teacli-update-123")
	if err := os.WriteFile(staged, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceExecutable(target, staged); err != nil {
		t.Fatalf("replaceExecutable: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new" {
		t.Errorf("target contents = %q, want \"new\"", got)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o111 == 0 {
		t.Errorf("target mode = %v, want the executable bit set", info.Mode().Perm())
	}
}

func TestUpdaterTimeoutIsSet(t *testing.T) {
	// A hung release server must not hang the CLI forever.
	up := &updater{client: &http.Client{Timeout: 60 * time.Second}}
	if up.client.Timeout == 0 {
		t.Error("update client has no timeout")
	}
}
