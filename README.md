# teacli

A non-interactive command-line tool for **Gitea** and **Forgejo**, built so that
scripts, CI jobs, and AI agents can drive a forge without a human at the
keyboard.

Nothing prompts. Nothing waits for a TTY. Every command takes explicit flags,
every error maps onto a documented exit code, and any output can be requested as
JSON.

Forgejo works because it serves the same `/api/v1` surface as Gitea; teacli talks
to either one with the same commands.

## Why agents get along with it

- **No prompts or confirmations.** Actions run immediately from their flags, so
  nothing blocks on input that never comes.
- **Schema discovery.** `teacli schema --format json` emits the whole command
  tree, with every path, argument, and flag. An agent can learn the CLI without
  scraping `--help`.
- **Stable exit codes.** `2` is a bad argument, `3` is not found, `4` is
  forbidden. Control flow does not depend on parsing prose.
- **Structured errors.** `--format json` puts errors on stderr as
  `{"error":true,"code":3,"message":"..."}`.
- **Dry-run everywhere.** Every mutating command takes `--dry-run` and reports
  what it would have done.
- **Repository inference.** Inside a clone, `owner/repo` is read from the git
  remote, so an agent working in a checkout does not have to track it.
- **A skill file in the repo.** [SKILL.md](SKILL.md) is the full command
  reference, written to be loaded as an agent skill.

Driving this from an agent? Point it at [SKILL.md](SKILL.md) for the command
reference and [AGENTS.md](AGENTS.md) for the usage rules.

## Installation

### Quick install (curl)

```bash
curl -fsSL https://raw.githubusercontent.com/rayfish/teacli/main/install.sh | sh
```

This downloads the binary for your OS and architecture, verifies it against the
release checksums, and installs it to `/usr/local/bin`. Override any of that
with environment variables:

```bash
# Pin a version and install somewhere on your PATH without sudo
TEACLI_VERSION=v1.0.0 TEACLI_INSTALL_DIR="$HOME/.local/bin" \
  curl -fsSL https://raw.githubusercontent.com/rayfish/teacli/main/install.sh | sh
```

Supported variables: `TEACLI_RELEASE_HOST` (default `https://github.com`),
`TEACLI_RELEASE_REPO` (default `rayfish/teacli`), `TEACLI_VERSION`
(default `latest`), `TEACLI_INSTALL_DIR` (default `/usr/local/bin`). The release
host is only where binaries are downloaded from, so pointing it at your own
Gitea or Forgejo instance installs from a mirror. Linux and macOS only; on
Windows, download `teacli-windows-amd64.exe` from the releases page.

### With the Go toolchain

```bash
go install github.com/rayfish/teacli@latest
```

### From source

```bash
just install                    # builds and installs to ~/.local/bin
just install /usr/local/bin     # or somewhere else (may need sudo)
```

### Updating

```bash
teacli update           # install the latest release
teacli update --check   # report what is available without installing
```

Downloads are verified against the release's `checksums.txt` and the new binary
is smoke-tested before it replaces the old one. Add `sudo` when teacli lives in
a root-owned directory.

## Quick Start

### 1. Configure authentication

```bash
# Login to a Gitea or Forgejo server
teacli auth login https://gitea.example.com --token YOUR_API_TOKEN

# Or name the server and make it the default
teacli auth login https://gitea.example.com --token YOUR_TOKEN --name production --default
```

### 2. List configured servers

```bash
teacli auth config list
```

## Repository inference

Inside a clone, leave the `owner/repo` argument out and teacli reads it from the
git remote:

```bash
cd ~/work/teacli
teacli issue list                 # rayfish/teacli
teacli action get 16057           # the run id shifts into first place
teacli repo current               # show what is inferred, and from where
teacli issue list other/repo      # naming a repository always wins
```

The remote's host has to match one of your configured servers, so a clone of
another forge never queries your server by mistake; you get a validation error
instead. The matched server is also the one teacli talks to, so a clone of a
non-default server needs no `--server`.

`TEACLI_TARGET_REPO=owner/repo` overrides detection where there is no clone, in
CI or a container. A few commands opt out on purpose: `repo delete`, because it
is permanent and nothing prompts; `notification list`/`read`, where omitting the
repository already means every repository; and `admin adopt`/`drop-unadopted`,
which act on repositories that have no clone to stand in.

## Common Commands

### Pull Requests

```bash
# List open PRs (text by default; -s all for every state)
teacli pr list owner/repo

# List as machine-readable JSON
teacli pr list owner/repo --format json

# Create a PR
teacli pr create owner/repo --head feature-branch --base main --title "Fix bug" --body "Details here"

# Merge a PR
teacli pr merge owner/repo 42 --method squash --delete-branch

# Get PR diff
teacli pr diff owner/repo 42
```

### Issues

```bash
# List issues, with filters
teacli issue list owner/repo --state all --label bug --assignee alice

# Create an issue
teacli issue create owner/repo --title "Bug report" --body "Steps to reproduce..." --assignee alice --label bug

# Close an issue
teacli issue close owner/repo 42
```

### CI/CD Actions

```bash
# List action runs
teacli action list owner/repo

# Inspect jobs and logs
teacli action jobs owner/repo 123
teacli action logs owner/repo 456

# Trigger a workflow (--ref defaults to the repo's default branch)
teacli action run owner/repo --workflow build.yml --inputs '{"env":"staging"}'

# Block until a run finishes
teacli action watch owner/repo 123
```

There is no `action cancel`: Gitea has no cancel-run endpoint.

### Webhooks

```bash
# List webhooks
teacli webhook list owner/repo

# Create a webhook
teacli webhook create owner/repo --url https://example.com/hook --events push,pull_request --secret mysecret

# Test a webhook
teacli webhook test owner/repo 1
```

### Repositories

```bash
# List repositories
teacli repo list

# Create a repository (always initialised with a first commit)
teacli repo create my-repo --description "My project" --private

# Rename a repository
teacli repo rename owner/old-name new-name
```

### Anything else

`teacli api` reaches any endpoint that has no typed command, carrying the same
auth, error mapping, and exit codes:

```bash
teacli api repos/owner/repo/topics
teacli api repos/owner/repo/issues --field title="from the raw API" --field body="details"
teacli api repos/owner/repo/issues/1 -X PATCH --raw-field state='"closed"'
teacli api user/repos --paginate
```

## Schema Discovery

`teacli schema` (alias `teacli commands`) emits the full command tree as JSON:
every command's path, description, and flags. This is what an agent should read
instead of guessing at flags or parsing `--help`.

```bash
teacli schema --format json | jq '.commands[].path'
```

## Output Formats

Output is text by default for readability. Two formats exist: `text` and `json`.

Format is resolved in this order: the `--format` flag, then the `TEACLI_FORMAT`
environment variable, then `text`.

### Text (default, human-readable)

```bash
teacli pr list owner/repo
```

```
ID    TITLE                              STATE    CREATED          AUTHOR
123   Fix memory leak                    open     2026-03-21 10:00 alice
124   Update documentation               closed   2026-03-20 15:30 bob
```

### JSON (machine-readable)

```bash
teacli pr list owner/repo --format json
```

```json
[
  {
    "id": 123,
    "number": 42,
    "title": "Fix memory leak",
    "state": "open"
  }
]
```

You can also set a default for a whole shell session or CI job:

```bash
export TEACLI_FORMAT=json
```

## Dry-Run Mode

Preview changes without making them:

```bash
teacli pr merge owner/repo 42 --method squash --dry-run
```

Output:

```
DRY-RUN: Would merge PR #42 using squash method
  Delete source branch: false
```

## Exit Codes

| Code | Meaning |
| ---- | ------- |
| `0` | Success |
| `1` | General error (network, server, authentication) |
| `2` | Validation error (missing arguments, invalid values) |
| `3` | Resource not found |
| `4` | Permission denied |

Errors go to stderr and follow `--format`. Plain text by default:

```
teacli: pull request not found: 999
```

and JSON when you ask for it:

```bash
teacli pr get owner/repo 999 --format json
```

```json
{"error":true,"code":3,"message":"pull request not found: 999"}
```

## CI/CD Integration Example

```bash
#!/bin/bash
set -e

# Create a PR after a successful build
PR_OUTPUT=$(teacli pr create myorg/myrepo \
  --head "feature/build-$BUILD_NUMBER" \
  --base "main" \
  --title "Auto-merge build $BUILD_NUMBER" \
  --body "Automated build from CI" \
  --format json)

PR_NUM=$(echo "$PR_OUTPUT" | jq -r .number)

# Wait for status checks
while true; do
  STATUS=$(teacli pr get myorg/myrepo $PR_NUM --format json | jq -r '.statuses.state')
  if [ "$STATUS" = "success" ]; then
    break
  elif [ "$STATUS" = "failure" ]; then
    echo "Build failed" >&2
    exit 1
  fi
  sleep 30
done

teacli pr merge myorg/myrepo $PR_NUM --method rebase --delete-branch
```

## Global Flags

These persistent flags are available on every subcommand. They have no shorthand
form, to avoid colliding with subcommand-specific flags.

- `--server <name>` Use a specific server from the config
- `--format <format>` Output format: `text` (default) or `json`
- `--dry-run` Show what would be done without making changes
- `--config <path>` Config file path
- `--verbose` Enable verbose output
- `--all` Auto-paginate list commands and return the full set

Environment: `TEACLI_FORMAT` sets the default output format and
`TEACLI_TARGET_REPO` sets the repository to act on when none is given.

## Pagination

List commands return the server's first page (30 items) by default, with nothing
in the output to say more exists. Pass `--all` to walk every page, or
`--page`/`--per-page` to select one specific page.

```bash
teacli issue list owner/repo --all
teacli pr list owner/repo --state all --all
```

## Configuration

Configuration is stored in `~/.config/teacli/teacli.ini`, or wherever `--config`
points:

```ini
[global]
default_server = production

[production]
url = https://gitea.example.com
token = your_api_token

[staging]
url = https://staging.gitea.example.com
token = your_staging_token
```

## Available Commands

```
auth       Authentication and configuration management
schema     Emit the full command tree as JSON for agent discovery
repo       Manage repositories
pr         Manage pull requests
issue      Manage issues
webhook    Manage webhooks
action     Manage CI/CD actions
org        Manage organizations
user       Manage users
branch     Manage branches
release    Manage releases
label      Manage labels
milestone  Manage milestones
api        Call any endpoint that has no typed command
update     Update teacli to the latest release
```

`teacli schema --format json` is the complete list.

## Development

```bash
just build   # build ./teacli
just test    # go test ./...
just release # cross-compile every release target into dist/
```

Contributions are welcome. When you add or change a command, update
[SKILL.md](SKILL.md) in the same commit; it is the reference agents load, and a
stale one is worse than a missing one.

## License

Apache License 2.0. See [LICENSE](LICENSE).
