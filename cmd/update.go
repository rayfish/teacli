// Package cmd contains the self-update command
package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/rayfish/teacli/modules/errors"

	"github.com/spf13/cobra"
)

// Where releases are published. Both match install.sh. The release host is not
// the Gitea or Forgejo server teacli talks to; it is only where binaries are
// downloaded from, so it can be pointed at a mirror.
const (
	defaultReleaseHost = "https://github.com"
	defaultUpdateRepo  = "rayfish/teacli"
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update teacli to the latest release",
	Long: `Download the latest release binary for this platform and replace the running
executable. The download is verified against the release's checksums.txt.`,
	Args: cobra.NoArgs,
	RunE: runUpdate,
}

func init() {
	RootCmd.AddCommand(updateCmd)

	updateCmd.Flags().Bool("check", false, "Report whether an update exists without installing it")
	updateCmd.Flags().String("tag", "", "Install a specific release tag instead of the latest")
	updateCmd.Flags().Bool("force", false, "Reinstall even when already on the target version")
	updateCmd.Flags().String("host", defaultReleaseHost, "Host publishing the releases (GitHub, or a Gitea/Forgejo mirror)")
	updateCmd.Flags().String("repo", defaultUpdateRepo, "owner/repo publishing the releases")
	updateCmd.Flags().Bool("no-verify", false, "Skip checksum verification (not recommended)")
}

// updater fetches release metadata and assets over HTTP.
type updater struct {
	host   string
	repo   string
	token  string
	client *http.Client
}

// releaseBase is the URL prefix release assets are served from.
func (u *updater) releaseBase(tag string) string {
	return fmt.Sprintf("%s/%s/releases/download/%s", strings.TrimSuffix(u.host, "/"), u.repo, tag)
}

func (u *updater) get(rawURL string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to build request: %v", err))
	}
	if u.token != "" {
		req.Header.Set("Authorization", "token "+u.token)
	}

	resp, err := u.client.Do(req)
	if err != nil {
		return nil, errors.NewGeneralError(err.Error())
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.NewGeneralError(fmt.Sprintf("failed to read %s: %v", rawURL, err))
	}
	if resp.StatusCode >= 400 {
		return nil, apiError(resp.StatusCode, data)
	}
	return data, nil
}

// latestEndpoint is where the newest release's metadata lives. GitHub serves it
// from api.github.com, while Gitea and Forgejo serve it from /api/v1 on the
// release host itself. Both answer with a tag_name.
func (u *updater) latestEndpoint() string {
	host := strings.TrimSuffix(u.host, "/")
	if isGitHub(host) {
		return fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", u.repo)
	}
	return fmt.Sprintf("%s/api/v1/repos/%s/releases/latest", host, u.repo)
}

// isGitHub reports whether a release host is github.com itself.
func isGitHub(host string) bool {
	parsed, err := url.Parse(host)
	if err != nil {
		return false
	}
	return strings.EqualFold(parsed.Host, "github.com")
}

// latestTag resolves the newest published release tag.
func (u *updater) latestTag() (string, error) {
	endpoint := u.latestEndpoint()

	data, err := u.get(endpoint)
	if err != nil {
		return "", err
	}

	var release struct {
		TagName string `json:"tag_name"`
	}
	if err := json.Unmarshal(data, &release); err != nil {
		return "", errors.NewGeneralError(fmt.Sprintf("could not parse the release response from %s: %v", endpoint, err))
	}
	if release.TagName == "" {
		return "", errors.NewGeneralError(fmt.Sprintf("no release tag found at %s; pass --tag to pin a version", endpoint))
	}
	return release.TagName, nil
}

// fetchAsset downloads a release asset and returns its bytes.
func (u *updater) fetchAsset(tag, asset string) ([]byte, error) {
	return u.get(u.releaseBase(tag) + "/" + url.PathEscape(asset))
}

// assetName is the release artifact for a platform, matching the names the
// release workflow builds.
func assetName(goos, goarch string) string {
	name := fmt.Sprintf("teacli-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

// normalizeVersion strips a leading "v" so tags and --version output compare.
func normalizeVersion(v string) string {
	return strings.TrimPrefix(strings.TrimSpace(v), "v")
}

// parseChecksums reads `sha256sum` output into asset name -> hex digest.
func parseChecksums(data []byte) map[string]string {
	sums := map[string]string{}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		// sha256sum writes "digest  name", with binary mode marking name "*name".
		sums[strings.TrimPrefix(fields[1], "*")] = fields[0]
	}
	return sums
}

// verifyChecksum fails when the asset's digest is absent or does not match.
func verifyChecksum(sums map[string]string, asset string, payload []byte) error {
	want, ok := sums[asset]
	if !ok {
		return errors.NewGeneralError(fmt.Sprintf("checksums.txt has no entry for %s; re-run with --no-verify to skip verification", asset))
	}

	digest := sha256.Sum256(payload)
	got := hex.EncodeToString(digest[:])
	if !strings.EqualFold(want, got) {
		return errors.NewGeneralError(fmt.Sprintf("checksum mismatch for %s (want %s, got %s)", asset, want, got))
	}
	return nil
}

// currentExecutable resolves the running binary, following symlinks so an
// update replaces the real file rather than a link to it.
func currentExecutable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", errors.NewGeneralError(fmt.Sprintf("could not locate the running binary: %v", err))
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return exe, nil
	}
	return resolved, nil
}

// replaceExecutable swaps newPath in for target. The staged file must already
// live in target's directory so the rename stays on one filesystem.
func replaceExecutable(target, newPath string) error {
	if runtime.GOOS == "windows" {
		// Windows will not rename over a running image, so move it aside first.
		old := target + ".old"
		os.Remove(old)
		if err := os.Rename(target, old); err != nil {
			return errors.NewGeneralError(fmt.Sprintf("could not move the current binary aside: %v", err))
		}
		if err := os.Rename(newPath, target); err != nil {
			os.Rename(old, target)
			return errors.NewGeneralError(fmt.Sprintf("could not install the new binary: %v", err))
		}
		os.Remove(old)
		return nil
	}

	if err := os.Rename(newPath, target); err != nil {
		if os.IsPermission(err) {
			return errors.NewPermissionError(fmt.Sprintf("cannot write to %s. Re-run with sudo, or install to a directory you own.", filepath.Dir(target)))
		}
		return errors.NewGeneralError(fmt.Sprintf("could not install the new binary: %v", err))
	}
	return nil
}

func runUpdate(cmd *cobra.Command, args []string) error {
	check, _ := cmd.Flags().GetBool("check")
	tag, _ := cmd.Flags().GetString("tag")
	force, _ := cmd.Flags().GetBool("force")
	host, _ := cmd.Flags().GetString("host")
	repo, _ := cmd.Flags().GetString("repo")
	noVerify, _ := cmd.Flags().GetBool("no-verify")

	printer := getPrinter(cmd)

	// Releases are published on GitHub, which needs no token, while the servers
	// teacli talks to are Gitea or Forgejo. Send a configured token only when
	// the release host is that same server, so updating from a private mirror
	// still authenticates and a token never reaches a host it does not belong
	// to.
	if host == "" {
		host = defaultReleaseHost
	}
	token := ""
	if server, err := getServer(cmd); err == nil && sameHost(server.URL, host) {
		token = server.Token
	}

	up := &updater{
		host:   host,
		repo:   repo,
		token:  token,
		client: &http.Client{Timeout: 60 * time.Second},
	}

	if tag == "" {
		latest, err := up.latestTag()
		if err != nil {
			return err
		}
		tag = latest
	}

	current := normalizeVersion(cmd.Root().Version)
	target := normalizeVersion(tag)
	upToDate := current == target

	if check {
		return printer.Emit(map[string]any{
			"current":          current,
			"latest":           target,
			"update_available": !upToDate,
		}, func() error {
			if upToDate {
				printer.Printf("teacli %s is up to date\n", current)
			} else {
				printer.Printf("teacli %s is installed; %s is available\n", current, target)
			}
			return nil
		})
	}

	if upToDate && !force {
		return printer.Emit(map[string]any{"status": "up-to-date", "version": current}, func() error {
			printer.Printf("Already on %s. Pass --force to reinstall.\n", current)
			return nil
		})
	}

	exe, err := currentExecutable()
	if err != nil {
		return err
	}

	asset := assetName(runtime.GOOS, runtime.GOARCH)

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		printer.Printf("DRY-RUN: Would update teacli %s -> %s\n", current, target)
		printer.Printf("  Asset: %s\n", asset)
		printer.Printf("  From: %s\n", up.releaseBase(tag)+"/"+asset)
		printer.Printf("  Replacing: %s\n", exe)
		return nil
	}

	payload, err := up.fetchAsset(tag, asset)
	if err != nil {
		if cliErr, ok := err.(*errors.CLIError); ok && cliErr.Code == errors.ExitNotFound {
			return errors.NewNotFoundError("release asset",
				fmt.Sprintf("%s for %s (%s)", asset, tag, up.releaseBase(tag)))
		}
		return err
	}

	if !noVerify {
		sums, err := up.fetchAsset(tag, "checksums.txt")
		if err != nil {
			return errors.NewGeneralError(fmt.Sprintf("could not fetch checksums.txt for %s: %s. Re-run with --no-verify to skip verification.", tag, err))
		}
		if err := verifyChecksum(parseChecksums(sums), asset, payload); err != nil {
			return err
		}
	}

	// Stage in the target directory so the final rename is atomic.
	staged, err := os.CreateTemp(filepath.Dir(exe), ".teacli-update-*")
	if err != nil {
		if os.IsPermission(err) {
			return errors.NewPermissionError(fmt.Sprintf("cannot write to %s. Re-run with sudo, or install to a directory you own.", filepath.Dir(exe)))
		}
		return errors.NewGeneralError(fmt.Sprintf("could not stage the download: %v", err))
	}
	stagedPath := staged.Name()
	defer os.Remove(stagedPath)

	if _, err := staged.Write(payload); err != nil {
		staged.Close()
		return errors.NewGeneralError(fmt.Sprintf("could not write the download: %v", err))
	}
	if err := staged.Close(); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("could not write the download: %v", err))
	}
	if err := os.Chmod(stagedPath, 0o755); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("could not make the download executable: %v", err))
	}

	// Never swap in a binary that cannot run.
	if out, err := exec.Command(stagedPath, "--version").CombinedOutput(); err != nil {
		return errors.NewGeneralError(fmt.Sprintf("the downloaded binary does not run (%v): %s", err, strings.TrimSpace(string(out))))
	}

	if err := replaceExecutable(exe, stagedPath); err != nil {
		return err
	}

	return printer.Emit(map[string]any{
		"status": "updated",
		"from":   current,
		"to":     target,
		"path":   exe,
	}, func() error {
		printer.Printf("Updated teacli %s -> %s (%s)\n", current, target, exe)
		return nil
	})
}

// sameHost reports whether two URLs point at the same server, so the configured
// token is only sent to the server it belongs to.
func sameHost(a, b string) bool {
	ua, err := url.Parse(a)
	if err != nil {
		return false
	}
	ub, err := url.Parse(b)
	if err != nil {
		return false
	}
	return strings.EqualFold(ua.Host, ub.Host)
}
