// Package gitremote resolves the repository of the current directory from its
// git remotes, so commands can infer owner/repo instead of being given it.
package gitremote

import (
	"errors"
	"fmt"
	"net/url"
	"os/exec"
	"sort"
	"strings"
)

var (
	// ErrNoRepo reports that the directory is not inside a git repository.
	ErrNoRepo = errors.New("not in a git repository")
	// ErrNoRemote reports that the repository has no remotes to read.
	ErrNoRemote = errors.New("git repository has no remotes")
)

// HostMismatchError reports that a remote was found but its host belongs to no
// configured server, which is the case in a clone of some other forge.
type HostMismatchError struct {
	Remote string
	Host   string
}

func (e *HostMismatchError) Error() string {
	return fmt.Sprintf("remote %s points at %s, which matches no configured server", e.Remote, e.Host)
}

// Server is a configured Gitea server: the name it is known by and the base
// URL its API lives at.
type Server struct {
	Name string
	URL  string
}

// Remote is a git remote URL split into the parts that identify a repository.
type Remote struct {
	Name string
	Host string
	Path string
}

// Detected is a repository resolved from the current directory.
type Detected struct {
	Remote string
	Server string
	Owner  string
	Repo   string
}

// ParseURL splits a git remote URL into host and path. It accepts the three
// forms git writes: https/http URLs, ssh:// URLs, and the scp-like
// "[user@]host:owner/repo" shorthand.
func ParseURL(raw string) (*Remote, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, fmt.Errorf("empty remote URL")
	}

	var host, path string
	if strings.Contains(raw, "://") {
		u, err := url.Parse(raw)
		if err != nil {
			return nil, fmt.Errorf("unparseable remote URL %q: %w", raw, err)
		}
		host, path = u.Hostname(), u.Path
	} else {
		// scp-like: everything before the first colon is [user@]host. A path
		// starting with a slash is an absolute local path, not a repository
		// coordinate we can use.
		colon := strings.Index(raw, ":")
		if colon < 0 {
			return nil, fmt.Errorf("remote URL %q is a local path, not a server URL", raw)
		}
		host, path = raw[:colon], raw[colon+1:]
		if at := strings.LastIndex(host, "@"); at >= 0 {
			host = host[at+1:]
		}
		// A leading slash (git@host:/owner/repo.git) is an absolute path on
		// the host, not a repository coordinate. Strip it so the trailing
		// Trim handles it the same way the https:// branch does.
	}

	host = strings.ToLower(host)
	path = strings.Trim(path, "/")
	path = strings.TrimSuffix(path, ".git")
	if host == "" || strings.ContainsAny(host, " \t") {
		return nil, fmt.Errorf("remote URL %q has no host", raw)
	}
	if path == "" {
		return nil, fmt.Errorf("remote URL %q has no repository path", raw)
	}

	return &Remote{Host: host, Path: path}, nil
}

// SplitOwnerRepo takes the owner and repo out of a remote path, dropping the
// server's own path prefix first so subpath installations resolve. It reports
// false when what is left is not exactly "owner/repo".
func SplitOwnerRepo(path, prefix string) (owner, repo string, ok bool) {
	path = strings.Trim(path, "/")
	prefix = strings.Trim(prefix, "/")
	if prefix != "" {
		if !strings.HasPrefix(path, prefix+"/") {
			return "", "", false
		}
		path = strings.TrimPrefix(path, prefix+"/")
	}

	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// Detect resolves the repository for dir against the configured servers. It
// prefers a remote whose host matches a server, breaking ties by remote name
// in the usual order: origin, then upstream, then whatever else is defined.
func Detect(dir string, servers []Server) (*Detected, error) {
	if _, err := git(dir, "rev-parse", "--git-dir"); err != nil {
		return nil, ErrNoRepo
	}

	out, err := git(dir, "remote")
	if err != nil {
		return nil, ErrNoRemote
	}
	names := orderRemotes(strings.Fields(out))
	if len(names) == 0 {
		return nil, ErrNoRemote
	}

	var first *Remote
	for _, name := range names {
		raw, err := git(dir, "config", "--get", "remote."+name+".url")
		if err != nil {
			continue
		}
		remote, err := ParseURL(raw)
		if err != nil {
			continue
		}
		remote.Name = name
		if first == nil {
			first = remote
		}
		if d, ok := match(remote, servers); ok {
			return d, nil
		}
	}

	if first == nil {
		return nil, ErrNoRemote
	}
	return nil, &HostMismatchError{Remote: first.Name, Host: first.Host}
}

// match finds the server the remote belongs to. Where several share a host,
// the one with the longest matching path prefix wins, so a subpath install
// beats a server mounted at the root of the same host.
func match(remote *Remote, servers []Server) (*Detected, bool) {
	best := -1
	var found *Detected
	for _, server := range servers {
		u, err := url.Parse(server.URL)
		if err != nil || !strings.EqualFold(u.Hostname(), remote.Host) {
			continue
		}
		prefix := strings.Trim(u.Path, "/")
		owner, repo, ok := SplitOwnerRepo(remote.Path, prefix)
		if !ok || len(prefix) <= best {
			continue
		}
		best = len(prefix)
		found = &Detected{Remote: remote.Name, Server: server.Name, Owner: owner, Repo: repo}
	}
	return found, found != nil
}

// orderRemotes puts origin first and upstream second, leaving the rest in the
// order git listed them.
func orderRemotes(names []string) []string {
	rank := func(name string) int {
		switch name {
		case "origin":
			return 0
		case "upstream":
			return 1
		}
		return 2
	}
	ordered := append([]string(nil), names...)
	sort.SliceStable(ordered, func(i, j int) bool {
		return rank(ordered[i]) < rank(ordered[j])
	})
	return ordered
}

// git runs a read-only git command in dir and returns its trimmed output.
func git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
