# Agent instructions

`teacli` is a non-interactive CLI for Gitea and Forgejo.

Read [SKILL.md](SKILL.md) before using or changing it. It is the reference for
every command, flag, output format, and exit code.

Rules:

- Never guess a command or flag. Run `teacli schema --format json` to get the
  full command tree, or check SKILL.md.
- Inside a clone, the leading `<owner>/<repo>` can be left out and is read from
  the git remote. `teacli repo current` says what it resolves to. Naming the
  repository still works everywhere and always wins.
- Pass `--format json` when you need to parse output; text is the default.
- List commands return only the first page. Pass `--all` whenever a partial list
  would give you a wrong answer ("is there an open PR for X?").
- Use `--dry-run` first for anything destructive (delete, merge, close). Every
  mutating command supports it.
- Nothing prompts. Deletes execute immediately, so check before you run one.
- If no typed command covers what you need, use `teacli api <endpoint>` rather
  than curl: it carries the same auth, error mapping and exit codes.
- Errors go to stderr as `teacli: <message>`. Add `--format json` to get them as
  `{"error":true,"code":N,"message":"..."}` instead. Check the exit code:
  `1` general, `2` validation, `3` not found, `4` permission denied.
- When you add or change a command, update SKILL.md in the same commit.

Before committing a change: `gofmt -l .` (must print nothing), `go vet ./...`,
`go test ./...`. `just build` and `just test` wrap the last two.
