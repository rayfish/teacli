package gitremote

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseURL(t *testing.T) {
	tests := []struct {
		raw      string
		wantHost string
		wantPath string
	}{
		{"https://gitea.example.com/owner/repo.git", "gitea.example.com", "owner/repo"},
		{"https://gitea.example.com/owner/repo", "gitea.example.com", "owner/repo"},
		{"http://gitea.example.com:3000/owner/repo.git", "gitea.example.com", "owner/repo"},
		{"https://user:pass@gitea.example.com/owner/repo.git", "gitea.example.com", "owner/repo"},
		{"https://gitea.example.com/gitea/owner/repo.git", "gitea.example.com", "gitea/owner/repo"},
		{"ssh://git@gitea.example.com:2222/owner/repo.git", "gitea.example.com", "owner/repo"},
		{"ssh://git@gitea.example.com/owner/repo", "gitea.example.com", "owner/repo"},
		{"git@gitea.example.com:owner/repo.git", "gitea.example.com", "owner/repo"},
		{"git@code-host:owner/repo.git", "code-host", "owner/repo"},
		{"code-host:owner/repo", "code-host", "owner/repo"},
		{"git@gitea.example.com:gitea/owner/repo.git", "gitea.example.com", "gitea/owner/repo"},
		{"git@gitea.example.com:/owner/repo.git", "gitea.example.com", "owner/repo"},
		{"git@gitea.example.com:/gitea/owner/repo.git", "gitea.example.com", "gitea/owner/repo"},
		{"GIT@Gitea.Example.COM:Owner/Repo.git", "gitea.example.com", "Owner/Repo"},
	}
	for _, tt := range tests {
		got, err := ParseURL(tt.raw)
		if err != nil {
			t.Errorf("ParseURL(%q): %v", tt.raw, err)
			continue
		}
		if got.Host != tt.wantHost || got.Path != tt.wantPath {
			t.Errorf("ParseURL(%q) = host %q path %q, want host %q path %q",
				tt.raw, got.Host, got.Path, tt.wantHost, tt.wantPath)
		}
	}
}

func TestParseURLRejectsUnusable(t *testing.T) {
	for _, raw := range []string{"", "not a url", "/srv/local/repo.git", "https://gitea.example.com/", "file:///srv/repo.git"} {
		if got, err := ParseURL(raw); err == nil {
			t.Errorf("ParseURL(%q) = %+v, want an error", raw, got)
		}
	}
}

func TestSplitOwnerRepo(t *testing.T) {
	tests := []struct {
		path, prefix string
		wantOwner    string
		wantRepo     string
	}{
		{"owner/repo", "", "owner", "repo"},
		{"owner/repo", "/", "owner", "repo"},
		{"gitea/owner/repo", "/gitea", "owner", "repo"},
		{"gitea/owner/repo", "gitea/", "owner", "repo"},
	}
	for _, tt := range tests {
		owner, repo, ok := SplitOwnerRepo(tt.path, tt.prefix)
		if !ok || owner != tt.wantOwner || repo != tt.wantRepo {
			t.Errorf("SplitOwnerRepo(%q, %q) = (%q, %q, %v), want (%q, %q, true)",
				tt.path, tt.prefix, owner, repo, ok, tt.wantOwner, tt.wantRepo)
		}
	}

	bad := []struct{ path, prefix string }{
		{"repo", ""},
		{"owner/repo/extra", ""},
		{"gitea/owner/repo", "/other"},
	}
	for _, tt := range bad {
		if owner, repo, ok := SplitOwnerRepo(tt.path, tt.prefix); ok {
			t.Errorf("SplitOwnerRepo(%q, %q) = (%q, %q, true), want ok=false",
				tt.path, tt.prefix, owner, repo)
		}
	}
}

// gitRepo makes a throwaway git repository with the given remotes, as
// name/url pairs, and returns its path.
func gitRepo(t *testing.T, remotes ...string) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init", "-q")
	for i := 0; i+1 < len(remotes); i += 2 {
		run("remote", "add", remotes[i], remotes[i+1])
	}
	return dir
}

var exampleServers = []Server{{Name: "example", URL: "https://gitea.example.com"}}

func TestDetectResolvesOriginAgainstMatchingServer(t *testing.T) {
	dir := gitRepo(t, "origin", "git@gitea.example.com:owner/repo.git")

	got, err := Detect(dir, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	want := Detected{Remote: "origin", Server: "example", Owner: "owner", Repo: "repo"}
	if *got != want {
		t.Errorf("Detect = %+v, want %+v", *got, want)
	}
}

func TestDetectFindsRepoFromSubdirectory(t *testing.T) {
	dir := gitRepo(t, "origin", "https://gitea.example.com/owner/repo.git")
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := Detect(sub, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Errorf("Detect = %+v, want owner/repo", *got)
	}
}

func TestDetectPrefersOriginOverUpstream(t *testing.T) {
	dir := gitRepo(t,
		"upstream", "https://gitea.example.com/upstream-owner/repo.git",
		"origin", "https://gitea.example.com/my-owner/repo.git")

	got, err := Detect(dir, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Remote != "origin" || got.Owner != "my-owner" {
		t.Errorf("Detect = %+v, want origin my-owner/repo", *got)
	}
}

func TestDetectFallsBackToUpstreamThenSoleRemote(t *testing.T) {
	upstream := gitRepo(t, "upstream", "https://gitea.example.com/owner/repo.git")
	got, err := Detect(upstream, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Remote != "upstream" {
		t.Errorf("Detect remote = %q, want upstream", got.Remote)
	}

	sole := gitRepo(t, "mirror", "https://gitea.example.com/owner/repo.git")
	got, err = Detect(sole, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Remote != "mirror" {
		t.Errorf("Detect remote = %q, want mirror", got.Remote)
	}
}

func TestDetectPrefersTheRemoteThatMatchesAServer(t *testing.T) {
	dir := gitRepo(t,
		"origin", "git@github.com:owner/fork.git",
		"gitea", "https://gitea.example.com/owner/repo.git")

	got, err := Detect(dir, exampleServers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Remote != "gitea" || got.Repo != "repo" {
		t.Errorf("Detect = %+v, want the gitea remote", *got)
	}
}

func TestDetectMatchesHostIgnoringPortAndScheme(t *testing.T) {
	dir := gitRepo(t, "origin", "ssh://git@gitea.example.com:2222/owner/repo.git")
	servers := []Server{{Name: "example", URL: "http://gitea.example.com:3000"}}

	got, err := Detect(dir, servers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Server != "example" {
		t.Errorf("Detect server = %q, want example", got.Server)
	}
}

func TestDetectStripsServerSubpath(t *testing.T) {
	dir := gitRepo(t, "origin", "https://host.example.com/gitea/owner/repo.git")
	servers := []Server{{Name: "sub", URL: "https://host.example.com/gitea"}}

	got, err := Detect(dir, servers)
	if err != nil {
		t.Fatalf("Detect: %v", err)
	}
	if got.Owner != "owner" || got.Repo != "repo" {
		t.Errorf("Detect = %+v, want owner/repo", *got)
	}
}

func TestDetectOutsideAGitRepository(t *testing.T) {
	dir := t.TempDir()

	_, err := Detect(dir, exampleServers)
	if !errors.Is(err, ErrNoRepo) {
		t.Errorf("Detect outside a repo = %v, want ErrNoRepo", err)
	}
}

func TestDetectWithNoRemotes(t *testing.T) {
	dir := gitRepo(t)

	_, err := Detect(dir, exampleServers)
	if !errors.Is(err, ErrNoRemote) {
		t.Errorf("Detect with no remotes = %v, want ErrNoRemote", err)
	}
}

func TestDetectWhenNoRemoteMatchesAConfiguredServer(t *testing.T) {
	dir := gitRepo(t, "origin", "git@github.com:owner/repo.git")

	_, err := Detect(dir, exampleServers)
	var mismatch *HostMismatchError
	if !errors.As(err, &mismatch) {
		t.Fatalf("Detect = %v, want a HostMismatchError", err)
	}
	if mismatch.Host != "github.com" {
		t.Errorf("HostMismatchError.Host = %q, want github.com", mismatch.Host)
	}
}
