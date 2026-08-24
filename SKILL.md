---
name: teacli
description: Use when working with Gitea or Forgejo from the command line or from an agent - pull requests, issues, reviews, comments, repos, files, commits, branches, tags, releases, webhooks, labels, milestones, orgs, teams, users, keys, packages, wikis, notifications, time tracking, protections and Actions on a Gitea or Forgejo server. Also use when a script or CI job needs non-interactive Gitea API access, or when `tea`/`gh`-style commands are needed against Gitea or Forgejo.
---

# teacli: non-interactive CLI for Gitea and Forgejo

A fully non-interactive command-line tool for Gitea and Forgejo, designed for scripts, CI/CD pipelines, and AI agents.

## Overview

**Repository:** `https://github.com/rayfish/teacli`

**Purpose:** Gitea/Forgejo CLI built on cobra + pflag. No prompts, no confirmations, no TTY needed. Forgejo is covered because it serves the same `/api/v1` surface.

**Key facts for agents:**

- Inside a clone of a Gitea or Forgejo repository, the leading `<owner>/<repo>` can be left
  out and it is read from the git remote (`teacli repo current` shows what it
  resolves to)
- Flags work in ANY order (`teacli pr list owner/repo --format json` == `teacli pr list --format json owner/repo`)
- Text is the default output; pass `--format json` for machine-readable output
- `--dry-run` is supported on every mutating command
- `teacli schema` emits the full command tree as JSON: prefer it over guessing or parsing `--help`
- Errors go to stderr as plain text, or as JSON under `--format json`; exit codes are stable (see below)
- `teacli api` reaches any endpoint that has no typed command yet

**Start here:** if a command or flag is not listed below, run `teacli schema --format json`
instead of assuming. The schema is generated from the binary and is always current.

## Installation

```bash
# Quick install (Linux/macOS)
curl -fsSL https://raw.githubusercontent.com/rayfish/teacli/main/install.sh | sh

# Or with the Go toolchain
go install github.com/rayfish/teacli@latest

# Or build and install locally (defaults to ~/.local/bin)
just install
```

## Staying current

```bash
teacli update           # install the latest release
teacli update --check   # report the available version, change nothing
teacli update --tag v1.0.0   # pin a specific release
```

The download is checksum-verified against the release's `checksums.txt`, and the
new binary must run before it replaces the old one. Releases come from GitHub;
pass `--host` and `--repo` to install from a Gitea or Forgejo mirror instead. If
teacli lives in a root-owned directory such as `/usr/local/bin`, run it with
sudo.

## Configuration

### Login to a Gitea or Forgejo server

```bash
# Flags work in any order!
teacli auth login --token <api-token> <server-url>
teacli auth login <server-url> --token <api-token>  # Also works

# Set as default server
teacli auth login --token abc123 https://gitea.example.com --default
```

`teacli login` and `teacli logout` are top-level aliases for `teacli auth login`
and `teacli auth logout`.

### View and Edit Configuration

```bash
teacli auth config list
teacli auth config set <server> <key> <value> [--default]
teacli auth config unset <server>
teacli auth logout [server-name]
```

## Repository inference

Every command that starts with `<owner>/<repo>` can be run without it from
inside a clone of that repository. `teacli` reads the git remotes of the current
directory, matches their host against the servers in your config, and uses the
repository the matching remote points at.

```bash
cd ~/work/teacli
teacli issue list                 # rayfish/teacli
teacli action get 16057           # the run id shifts into first place
teacli pr merge 42 --method squash

teacli repo current               # show what would be inferred, and from where
teacli issue list other/repo      # naming a repository always wins
```

Matching is strict: the remote host has to match a configured server, so a clone
of some other forge never queries your Gitea or Forgejo server by mistake.

```
teacli: cannot infer repository: remote origin points at github.com, which matches no configured server
teacli: not in a git repository
```

Both exit 2, like any other bad argument.

Details worth knowing:

- Remotes are tried in the order `origin`, `upstream`, then the rest, and the
  first one that matches a configured server wins. A fork with `origin` on Gitea
  and `upstream` on GitHub resolves to the Gitea one either way.
- The matched server also selects the server to talk to, so a clone of a
  non-default server works without `--server`. Naming a repository explicitly
  never does this: the directory you type it in does not redirect it.
- `TEACLI_TARGET_REPO=owner/repo` overrides detection, for CI and containers
  with no clone. (`TEACLI_RELEASE_REPO` is a different thing: `install.sh` reads
  it to find teacli's own repository.)
- `repo delete` keeps its repository required. It is permanent and nothing
  prompts, so it does not act on whatever directory you happen to be in.
- `notification list` and `notification read` also keep their behaviour:
  omitting the repository there already means every repository.
- `file list` and `git refs` take their optional path positionally only after an
  explicit repository, since a path contains slashes too. With the repository
  inferred, pass `--path` or `--ref-prefix`.

## Commands

### Pull Requests

```bash
# List PRs
teacli pr list <owner>/<repo> [-s open|closed|all]

# Get PR details and diff
teacli pr get <owner>/<repo> <pr-number>
teacli pr diff <owner>/<repo> <pr-number>

# Create PR
teacli pr create <owner>/<repo> --head <branch> --base <branch> --title <title>

# Merge PR
teacli pr merge <owner>/<repo> <pr-number> [--method merge|rebase|rebase-merge|squash] [--delete-branch]

# Close/Reopen PR
teacli pr close <owner>/<repo> <pr-number>
teacli pr reopen <owner>/<repo> <pr-number>

# Approve/Reject PR
teacli pr approve <owner>/<repo> <pr-number> [-b "message"]
teacli pr reject <owner>/<repo> <pr-number> [-b "message"]
teacli pr dismiss <owner>/<repo> <pr-number> <review-id> [-m "message"]

# Comments and Reviews
teacli pr comments <owner>/<repo> <pr-number>           # List issue comments
teacli pr comments delete <owner>/<repo> <comment-id>   # Delete a comment
teacli pr reviews <owner>/<repo> <pr-number>            # List reviews
teacli pr review-comments <owner>/<repo> <pr-number> <review-id>  # Inline comments
teacli pr reply <owner>/<repo> <pr-number> -b "text"    # General comment
teacli pr reply <owner>/<repo> <pr-number> -b "text" --file path --line N  # Inline review comment
teacli pr delete <owner>/<repo> <pr-number> <review-id> # Delete a review (admin only)

# Submit a full review with inline comments in one call
teacli pr review submit <owner>/<repo> <pr-number> \
  --state approve|request-changes|comment \
  -b "summary" \
  --comment "path/to/file.go:42:this leaks the handle" \
  --comment "other.go:7:nit: rename"
```

`--comment` uses `file:line:body` and can be repeated.

```bash
# Edit, inspect and check a PR
teacli pr update <owner>/<repo> <pr-number> [-t title] [-b body] [--base branch]
                                            [-a assignee] [-l label-id] [-s open|closed]
teacli pr commits <owner>/<repo> <pr-number>
teacli pr files <owner>/<repo> <pr-number>
teacli pr patch <owner>/<repo> <pr-number>
teacli pr merged <owner>/<repo> <pr-number>      # exit 3 when not merged

# Review requests
teacli pr request-review <owner>/<repo> <pr-number> <reviewer>... [--team]
teacli pr unrequest-review <owner>/<repo> <pr-number> <reviewer>... [--team]
teacli pr review-get <owner>/<repo> <pr-number> <review-id>
teacli pr undismiss <owner>/<repo> <pr-number> <review-id>
```

### Issues

```bash
# List, with filters. Default state is open; use --all for every page.
teacli issue list <owner>/<repo> [-s open|closed|all] [--label <name>]
                                 [--milestone <title>] [--assignee <user>]
                                 [--creator <user>] [-q <keyword>]

teacli issue get <owner>/<repo> <issue-number>
teacli issue create <owner>/<repo> --title <title> [--body <body>]
                                   [--label <name-or-id>] [--assignee <user>]
teacli issue close <owner>/<repo> <issue-number>
teacli issue reopen <owner>/<repo> <issue-number>

# Labels on an issue (names are matched case-insensitively, IDs also work)
teacli issue label list <owner>/<repo> <issue-number>
teacli issue label add <owner>/<repo> <issue-number> bug urgent
teacli issue label remove <owner>/<repo> <issue-number> urgent
teacli issue label clear <owner>/<repo> <issue-number>

# Assignees (add/remove, the rest of the set is preserved)
teacli issue assign <owner>/<repo> <issue-number> <username>...
teacli issue unassign <owner>/<repo> <issue-number> <username>...

# Issue Comments
teacli issue comment list <owner>/<repo> <issue-number>
teacli issue comment create <owner>/<repo> <issue-number> --body "text"
teacli issue comment delete <owner>/<repo> <comment-id>
```

`--label` and `--assignee` are repeatable on `issue create`.

```bash
# Edit and delete
teacli issue update <owner>/<repo> <issue-number> [-t title] [-b body] [-s open|closed]
                                                  [-m milestone-id] [-D due] [--no-due]
teacli issue delete <owner>/<repo> <issue-number>          # admin only, permanent

# Pins, locks and templates
teacli issue pin <owner>/<repo> <issue-number>
teacli issue unpin <owner>/<repo> <issue-number>
teacli issue pinned <owner>/<repo>
teacli issue lock <owner>/<repo> <issue-number> [-r reason]
teacli issue unlock <owner>/<repo> <issue-number>
teacli issue templates <owner>/<repo>
teacli issue timeline <owner>/<repo> <issue-number>

# Dependencies and blocking
teacli issue dependency list|add|remove <owner>/<repo> <issue-number> [<other-number>]
teacli issue blocking list|add|remove <owner>/<repo> <issue-number> [<other-number>]

# More comment operations
teacli issue comment get <owner>/<repo> <comment-id>
teacli issue comment update <owner>/<repo> <comment-id> -b "text"
```

### Repositories

```bash
teacli repo list [-u user] [-o org]
teacli repo get <owner>/<repo>
teacli repo current                              # the repository inferred from the current directory
teacli repo search [<keyword>] [-o owner] [--type fork|source|mirror] [--sort alpha|created|updated]
teacli repo create <name> [-d description] [-p private] [--org <org>]
                          [--default-branch <b>] [--license <l>] [--gitignores <g>]
teacli repo from-template <template-owner>/<template-repo> <name> [--owner <o>]
teacli repo edit <owner>/<repo> [--name n] [-d desc] [--private|--public]
                                [--issues|--no-issues] [--wiki|--no-wiki] [--archived|--unarchived]
teacli repo rename <owner>/<old-name> <new-name>
teacli repo delete <owner>/<repo>                # permanent, no confirmation prompt
teacli repo fork <owner>/<repo> [--org <org>] [--name <n>]
teacli repo forks <owner>/<repo>
teacli repo migrate <clone-url> --name <name> [--mirror] [--auth-token <t>] [--wiki] [--issues]
teacli repo transfer <owner>/<repo> <new-owner>
teacli repo transfer-accept|transfer-reject <owner>/<repo>
teacli repo mirror-sync <owner>/<repo>
teacli repo languages <owner>/<repo>
teacli repo archive <owner>/<repo> <ref> [-F zip|tar.gz] [-o file]
teacli repo avatar <owner>/<repo> [-i image.png | --delete]
teacli repo reviewers <owner>/<repo>
teacli repo assignees <owner>/<repo>
teacli repo stargazers <owner>/<repo>

# Topics
teacli repo topic list|add|remove|set <owner>/<repo> [<topic>...]

# Push mirrors
teacli repo mirror list|get|create|delete <owner>/<repo> [<remote>]

# Server-side git hooks (admin only)
teacli repo hook list|get|set|delete <owner>/<repo> [<hook-name>]

# Team access
teacli repo team list|add|remove|check <owner>/<repo> [<team>]
```

### Files and git objects

```bash
teacli file list <owner>/<repo> [<path>] [--path <p>] [-r ref]
teacli file get <owner>/<repo> <path> [-r ref] [-o out]
teacli file info <owner>/<repo> <path> [-r ref]
teacli file create <owner>/<repo> <path> (-c content | -f file) [-m message] [-b branch]
teacli file update <owner>/<repo> <path> (-c content | -f file) [--sha <blob-sha>]
teacli file delete <owner>/<repo> <path> [--sha <blob-sha>]

teacli git refs <owner>/<repo> [<prefix>] [--ref-prefix <p>]
teacli git tree <owner>/<repo> <ref> [--recursive]
teacli git blob <owner>/<repo> <sha> [-o out]
teacli git note <owner>/<repo> <sha>
```

`file update` and `file delete` look the blob SHA up for you when `--sha` is
omitted. Content is base64-encoded on the wire; you always pass and receive
plain text.

### Commits, tags and statuses

```bash
teacli commit list <owner>/<repo> [--sha ref] [--path p] [--stat] [--files]
teacli commit get <owner>/<repo> <sha>
teacli commit diff <owner>/<repo> <sha>
teacli commit patch <owner>/<repo> <sha>
teacli commit compare <owner>/<repo> <base> <head>

teacli tag list|get|create|delete <owner>/<repo> [<tag>] [-m message] [--target ref]

teacli status list <owner>/<repo> <ref>
teacli status combined <owner>/<repo> <ref>
teacli status create <owner>/<repo> <sha> -s pending|success|error|failure|warning
                                          [-c context] [-d description] [-u target-url]
```

### Collaborators and protections

```bash
teacli collaborator list|add|remove|check|permission <owner>/<repo> [<username>...] [-p read|write|admin]

teacli protection branch list|get|create|update|delete <owner>/<repo> [<rule>]
    [--required-approvals N] [--status-checks a,b] [--push-whitelist-users u1,u2]
    [--require-signed-commits] [--block-outdated] [--dismiss-stale-approvals]
teacli protection tag list|get|create|update|delete <owner>/<repo> [<pattern-or-id>]
    [--whitelist-users u1,u2] [--whitelist-teams t1]
```

### Teams

```bash
teacli team list <org>
teacli team mine
teacli team get <team-id>
teacli team search <org> <query>
teacli team create <org> <name> [-p read|write|admin|owner] [-d desc] [--units repo.code,repo.issues]
teacli team update <team-id> [--name n] [-p perm] [--all-repos|--no-all-repos]
teacli team delete <team-id>
teacli team member list|add|remove|check <team-id> [<username>...]
teacli team repo list|add|remove <team-id> [<owner>/<repo>]
```

### Labels

```bash
teacli label list <owner>/<repo>
teacli label get <owner>/<repo> <label-id>
teacli label create <owner>/<repo> --name <name> --color <hex> [-d description]
teacli label update <owner>/<repo> <label-id> [--name <n>] [--color <c>]
teacli label delete <owner>/<repo> <label-id>
```

### Milestones

```bash
teacli milestone list <owner>/<repo> [-s open|closed|all]
teacli milestone get <owner>/<repo> <milestone-id>
teacli milestone create <owner>/<repo> --title <title> [-d description] [-D due-date]
teacli milestone update <owner>/<repo> <milestone-id> [options]
teacli milestone close <owner>/<repo> <milestone-id>
teacli milestone reopen <owner>/<repo> <milestone-id>
teacli milestone delete <owner>/<repo> <milestone-id>
```

### Organizations

```bash
teacli org list [--public]
teacli org get <org-name>
teacli org create <org-name> [-f full-name] [-d description]
teacli org members list <org-name>
teacli org members add <org-name> <username>
teacli org members remove <org-name> <username>
teacli org members check <org-name> <username>
teacli org delete <org-name> [-y]

teacli org update <org-name> [-f full-name] [-d desc] [-w website] [-l location]
                             [-v public|limited|private]
teacli org rename <org-name> <new-name>
teacli org repos <org-name>
teacli org permission <org-name> <username>
teacli org public-members <org-name>
teacli org publicize|conceal <org-name> <username>
teacli org avatar <org-name> [-i image.png | --delete]
teacli org activity <org-name>
teacli org label list|get|create|update|delete <org-name> [<label-id>] [-n name] [-c color]
```

### Users (Admin)

```bash
teacli whoami                 # top-level alias for `user current`
teacli user current           # also accepts `teacli user whoami`
teacli user get <username>
teacli user create <username> -e <email> -p <password> [-a admin]
teacli user update <username> [options]
teacli user delete <username>
teacli user list [-q query]

teacli user search <query>
teacli user orgs [<username>]
teacli user activity <username>
teacli user heatmap <username>
teacli user avatar [-i image.png | --delete]
teacli user settings [--full-name n] [--website w] [--location l] [--theme t]
                     [--hide-email|--show-email] [--hide-activity|--show-activity]
teacli user email list|add|remove [<email>...]
```

### Keys and tokens

```bash
teacli key ssh list|get|add|delete [<username>|<title>|<key-id>] [-k key | -f keyfile]
teacli key gpg list|get|add|delete|token|verify [<username>|<key-id>] [-k key | -f keyfile]
teacli key deploy list|get|add|delete <owner>/<repo> [<title>|<key-id>] [-k key] [--read-only]

teacli token list
teacli token create <name> [-s read:repository,write:issue]
teacli token delete <name-or-id>

teacli oauth list|get|create|update|delete [<app-id>] [-r https://redirect]
```

`token` uses the endpoints Gitea only exposes over basic auth, so it fails with
"only BasicAuth allowed" when the configured server uses a token. Create tokens
in the web UI, or via `teacli api` with a basic-auth header.

### Social and notifications

```bash
teacli star list|add|remove|check [<owner>/<repo>]
teacli watch list|add|remove|check [<owner>/<repo>]
teacli follow list|followers|add|remove|check [<username>] 
teacli block list|add|remove|check [<username>] [--org <org>]
teacli subscription list|add|remove|check <owner>/<repo> <issue-number> [<username>]

teacli notification list [<owner>/<repo>] [-s unread,read,pinned] [--type issue,pull]
teacli notification get <thread-id>
teacli notification count
teacli notification read [<thread-id>] [--repo owner/repo]
teacli notification unread <thread-id>
teacli notification pin <thread-id>

teacli reaction list|add|remove <owner>/<repo> <issue-number> [<reaction>]
teacli reaction comment-list|comment-add|comment-remove <owner>/<repo> <comment-id> [<reaction>]
```

### Time tracking

```bash
teacli time list <owner>/<repo> [<issue-number>] [-u user] [--since t] [--before t]
teacli time mine
teacli time add <owner>/<repo> <issue-number> <duration>    # 3600, 90m, 1h30m
teacli time delete <owner>/<repo> <issue-number> <time-id>
teacli time reset <owner>/<repo> <issue-number>

teacli stopwatch list|start|stop|cancel [<owner>/<repo> <issue-number>]
```

### Wiki

```bash
teacli wiki list <owner>/<repo>
teacli wiki get <owner>/<repo> <page>
teacli wiki create <owner>/<repo> <page> (-c content | -f file) [-m message]
teacli wiki update <owner>/<repo> <page> (-c content | -f file) [-m message]
teacli wiki delete <owner>/<repo> <page>
teacli wiki revisions <owner>/<repo> <page>
```

### Packages, attachments and server info

```bash
teacli package list <owner>
teacli package get|files|delete <owner> <type> <name> <version>
teacli package latest <owner> <type> <name>
teacli package link|unlink <owner> <type> <name> [<repo-name>]

teacli attachment list|get|rename|delete <owner>/<repo> <comment-id> [<attachment-id>]

teacli server-version
teacli settings
teacli markdown (-t text | -f file) [-m markdown|gfm|comment] [-c owner/repo]
teacli template gitignore|license|label [<name>]
```

### Admin

```bash
teacli admin orgs
teacli admin create-org <owner> <org-name>
teacli admin create-repo <owner> <name>
teacli admin rename-user <username> <new-username>
teacli admin unadopted [-q pattern]
teacli admin adopt|drop-unadopted <owner>/<repo>
teacli admin cron
teacli admin run-cron <task-name>
teacli admin emails
teacli admin hook list|get|delete [<hook-id>]
teacli admin badge list|add|remove <username> [<badge-slug>...]
```

### Webhooks

```bash
teacli webhook list <owner>/<repo>
teacli webhook get <owner>/<repo> <webhook-id>
teacli webhook create <owner>/<repo> -u <url> [-e events] [-s secret]
teacli webhook update <owner>/<repo> <webhook-id> [options]
teacli webhook delete <owner>/<repo> <webhook-id>
teacli webhook test <owner>/<repo> <webhook-id>
```

`webhook` covers repository webhooks. For organization and account webhooks use
`teacli hook`:

```bash
teacli hook list|get|create|delete [<hook-id>] [--org <org>] [-u url] [-e push,issues]
```

### Releases

```bash
teacli release list <owner>/<repo>
teacli release get <owner>/<repo> <tag>
teacli release create <owner>/<repo> --tag <tag> --name <name> [-b body]
teacli release delete <owner>/<repo> <tag>
teacli release upload <owner>/<repo> <tag> <file-path>
teacli release update <owner>/<repo> <tag> [-n name] [-b body] [--draft|--no-draft]
                                           [--prerelease|--no-prerelease]

teacli release asset list <owner>/<repo> <tag>
teacli release asset get <owner>/<repo> <tag> <asset-id>
teacli release asset upload <owner>/<repo> <tag> <file> [--name <n>]
teacli release asset download <owner>/<repo> <tag> <asset-id> [-o out]
teacli release asset rename <owner>/<repo> <tag> <asset-id> <new-name>
teacli release asset delete <owner>/<repo> <tag> <asset-id>
```

### Branches

```bash
teacli branch list <owner>/<repo>
teacli branch get <owner>/<repo> <branch-name>
teacli branch create <owner>/<repo> <branch-name> --from <source-branch>
teacli branch delete <owner>/<repo> <branch-name>
```

### Actions/CI/CD

```bash
teacli action list <owner>/<repo> [--status <status>] [--event <event>] [--branch <branch>]
teacli action get <owner>/<repo> <run-id>
teacli action delete <owner>/<repo> <run-id>
teacli action jobs <owner>/<repo> <run-id>
teacli action job <owner>/<repo> <job-id>
teacli action logs <owner>/<repo> <job-id>

# Dispatch a workflow. --ref defaults to the repo's default branch.
teacli action run <owner>/<repo> --workflow <file> [--ref <branch-or-tag>] [--inputs '<json>']

# Block until a run finishes. `wait` is an alias for `watch`.
teacli action watch <owner>/<repo> <run-id> [--interval 5s] [--timeout 30m]
```

`action watch` polls the run until it stops, then prints it exactly like
`action get`. It exits `0` only when the run concludes `success`; any other
conclusion (`failure`, `cancelled`, `skipped`) and a reached `--timeout` exit
`1`, so `teacli action watch 107 && teacli release create v1.0.0` does what it
looks like. `--timeout 0` waits indefinitely. Status changes are reported on
stderr, one line per change, and are suppressed under `--format json` so stdout
stays a single object.

`action get`, `action watch`, `action jobs`, and `action delete` accept either
the run's **run number** (the short number shown in the Gitea web UI URL, e.g.
`107`) or its database ID. The run number is resolved to the database ID the API requires,
so `teacli action get owner/repo 107` works directly. `action job` and
`action logs` take a job database ID (there is no separate job number).

There is no `action cancel`: Gitea 1.25 has no cancel-run endpoint.

```bash
# Secrets and variables. Pass --org <name> instead of owner/repo to target an
# organization; --org replaces the repo argument rather than adding to it.
teacli action secret list [<owner>/<repo>] [--org <org>]
teacli action secret set [<owner>/<repo>] <name> <value> [-d description]
teacli action secret delete [<owner>/<repo>] <name>

teacli action variable list [<owner>/<repo>] [--org <org>]
teacli action variable get [<owner>/<repo>] <name>
teacli action variable set [<owner>/<repo>] <name> <value>
teacli action variable delete [<owner>/<repo>] <name>

# Workflows
teacli action workflow list <owner>/<repo>
teacli action workflow get <owner>/<repo> <workflow-file>
teacli action workflow enable|disable <owner>/<repo> <workflow-file>
```

Secret values are write-only: the server never returns them, so
`action secret list` shows names and descriptions only.

### Raw API access

Every typed command above wraps an endpoint. For anything teacli has no typed
command for, `teacli api` reaches the endpoint directly, with the same auth,
error mapping and exit codes.

```bash
teacli api repos/owner/repo                         # GET, path relative to /api/v1
teacli api user/repos --paginate                    # walk every page, emit one JSON array
teacli api repos/o/r/issues -f title="bug" -f body="details"    # POST with string fields
teacli api repos/o/r/issues/1 -X PATCH -F state='"closed"'      # PATCH with typed fields
teacli api repos/o/r/releases --input release.json              # send a file verbatim
teacli api repos/o/r/issues -X POST --input - < issue.json      # send stdin
teacli api version --include                        # status line and headers to stderr
```

- `--field/-f` always sends a string. `--raw-field/-F` parses the value as JSON,
  so numbers, booleans, `null`, arrays and objects keep their type. Quote strings
  as JSON there: `-F state='"closed"'`.
- Giving a body switches the default method from GET to POST.
- `--paginate` only works on endpoints that return a JSON array.
- `--dry-run` prints the method, path and body of a non-GET request and sends
  nothing.

Prefer a typed command where one exists: those give you table output, argument
validation, and per-resource not-found messages.

## Output Formats

Two output formats exist: `text` (the default) and `json`. Format resolves in this
order: the `--format` flag, then the `TEACLI_FORMAT` environment variable, then
`text`.

```bash
# Text format (default, human-readable table)
teacli pr list owner/repo

# JSON format (machine-readable)
teacli pr list owner/repo --format json

# Set a default for the session
export TEACLI_FORMAT=json
```

## Schema Discovery

`teacli schema` (alias `teacli commands`) emits the full command tree as JSON,
including every command's path, description, args, and flags. Use it to discover
available commands programmatically instead of parsing `--help` output.

```bash
teacli schema --format json | jq '.commands[].path'
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

and JSON when you ask for it, for scripts that parse failures:

```bash
teacli pr get owner/repo 999 --format json
```

```json
{"error":true,"code":3,"message":"pull request not found: 999"}
```

stdout carries only successful output, so redirecting stderr away is safe.

## Global Flags

Persistent flags available on every subcommand, with no shorthand (to avoid
colliding with subcommand-specific flags):

- `--server <name>` - Use specific server from config
- `--format <format>` - Output format: `text` (default) or `json`
- `--dry-run` - Show what would be done without making changes
- `--config <path>` - Config file path
- `--verbose` - Enable verbose output
- `--all` - Auto-paginate: walk every page and return the full set

## Pagination

List commands return the server's first page (30 items) by default. That is a
silent truncation: nothing in the output says more exists. Pass `--all` when you
need the complete set.

```bash
teacli issue list owner/repo --all
teacli repo list --all
teacli pr list owner/repo --state all --all
teacli api user/repos --paginate
```

`--page` and `--per-page` select a single specific page instead. `--per-page` is
capped by the server's MAX_RESPONSE_ITEMS; `--all` accounts for that clamp, so
combining them is safe.

## Dry-Run Mode

Every mutating command supports `--dry-run`. It prints what would happen and
sends no request:

```bash
# Preview what would be done without making changes
teacli pr create owner/repo --head feature --base main --title "New feature" --dry-run
teacli label delete owner/repo 123 --dry-run
```

## Examples

### Common Workflows

**Review and approve a PR:**

```bash
# Review PR details
teacli pr get owner/repo 123

# View inline review comments
teacli pr review-comments owner/repo 123 456

# Approve with message
teacli pr approve owner/repo 123 -b "LGTM!"

# Merge after approval
teacli pr merge owner/repo 123 --method squash
```

**Create and manage a release:**

```bash
# Create release
teacli release create owner/repo --tag v0.1.0 --name "v0.1.0" -b "Initial release"

# Upload binary
teacli release upload owner/repo v0.1.0 ./dist/teacli-linux-amd64
```

**Script-friendly usage:**

```bash
#!/bin/bash
# List all open PRs (pass --format json for machine parsing)
teacli pr list owner/repo --format json | jq '.[] | select(.state == "open")'

# Get PR URL and merge
PR_URL=$(teacli pr get owner/repo 123 --format json | jq -r .html_url)
echo "Merging $PR_URL"
teacli pr merge owner/repo 123
```

## Build & Release

### Local Development

```bash
just build      # Build local binary
just test       # Run tests
just clean      # Clean artifacts
```

### Create Release

```bash
# Build multi-platform binaries
just release

# Generate checksums
just checksums

# Create and push tag (triggers CI/CD)
just release-tag 1.0.0   # positional, NOT version=1.0.0
```

### CI/CD Workflow

Pushing a tag triggers automatic build and release:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This creates a GitHub release with binaries for:

- Linux (amd64, arm64)
- macOS (amd64, arm64)
- Windows (amd64)

## Troubleshooting

**"no such host" error:**

```bash
# Check your configured servers
teacli auth config list

# Login to correct server
teacli auth login --token <token> https://gitea.example.com --default
```

**Flag ordering issues:**

- With cobra + pflag, flags work in ANY order
- Both `teacli pr list owner/repo --format json` and `teacli pr list --format json owner/repo` work

**Permission denied:**

- Ensure your API token has sufficient permissions
- Admin functions require admin privileges
