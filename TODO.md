# Known gaps

The typed commands cover the Gitea API surface that day-to-day work needs:
repos, files, commits, branches, tags, PRs and reviews, issues, labels,
milestones, releases and assets, webhooks, Actions, orgs, teams, users, keys,
wikis, notifications, time tracking, protections, search, stars, reactions,
forks, migration and admin operations. `teacli api` reaches anything that has no
typed command.

What is still missing:

- `teacli token` uses endpoints Gitea only exposes over basic auth, so it fails
  against a token-authenticated server. Either add basic-auth support to the
  config, or drop the command and document `teacli api` with `-H` instead.
- ActivityPub (`/activitypub/*`) and NodeInfo have no typed commands. They are
  federation plumbing rather than day-to-day operations, and `teacli api`
  covers them.
- There is no `action cancel`, because Gitea has no cancel-run endpoint.

## Implementation notes

- All commands support `--dry-run`, `--format text|json`, `--server <name>`,
  and map errors onto the documented exit codes.
- Shared helpers live in `cmd/helpers.go` (argument parsing, dry-run, paging
  flags) and `cmd/render.go` (the `detail` key/value renderer and the shared
  table emitters). Use them rather than hand-rolling output.
- `cmd/api.go` holds both the raw `teacli api` command and the `apiRequest`
  helper that typed commands use for endpoints the SDK does not wrap.
- `cmd/repotarget.go` resolves `owner/repo` from the git remote when the
  argument is omitted.
