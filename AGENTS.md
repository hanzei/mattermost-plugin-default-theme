# AGENTS.md

Mattermost plugin that applies an admin-configured theme to every newly created user. It is a server-only Go plugin (`server/`); there is no webapp. Admins paste theme JSON into the `CustomTheme` setting, `OnConfigurationChange` validates it, and `UserHasBeenCreated` saves it as the new user's `theme` preference. The `TeamThemes` setting maps team names to themes; `UserHasJoinedTeam` saves the matching one as the user's per-team `theme` preference, unless the user has already chosen a theme.

**Before editing `server/`, read `server/AGENTS.md`.**

## Commands

While iterating, run the narrow command; `make test` and `make check-style` reinstall Go tools first, and `make test` also rebuilds the bundle.

```bash
go test ./server/...                           # server unit tests
go test ./server -run TestUserHasBeenCreated   # single test
go vet ./... && golangci-lint run ./...        # lint
make test                                      # full check before pushing
make dist                                      # build the plugin bundle
```

`make help` lists the rest (deploy, logs, release bumps, debugger attach).

## Architecture

- `plugin.json` is the source of truth for ID, version, and settings. `make apply` generates `server/manifest.go` from it; it's gitignored, so never edit it. Never edit `dist/` or `server/dist/`.
- Project-specific make targets go in `build/custom.mk`.
- Build tooling (`Makefile`, `build/`, `.golangci.yml`, `.github/workflows/ci.yml`) is synced from [mattermost-plugin-starter-template](https://github.com/mattermost/mattermost-plugin-starter-template). Keep local changes to those files minimal so future syncs stay easy.
- `min_server_version` is 11.7.0, the oldest supported ESR. Don't use plugin API methods newer than that without bumping it.

## Rules that apply everywhere

- The server only authenticates plugin requests. If you add routes or slash commands that touch data, check the user's permission on that object, on the server; hiding UI is not a check.
- Anything that changes state uses POST, PUT, PATCH or DELETE. The server's CSRF check doesn't cover GET.
- Never send internal errors to clients or write secrets or tokens to logs.
- Don't add dependencies the host or the standard library already provide. Fix CodeQL alerts instead of adding exclusions to `.github/codeql/codeql-config.yml`; the only one is `go/log-injection`, which the server excludes too.
- In GitHub workflows, pin third-party actions to a commit SHA with the version in a comment, and start from `permissions: {}`.

## Pull requests and commits

- Commit subject: one succinct line. Don't add a `Co-Authored-By` line for the agent.
- In the PR description, call out any change to permissions, authentication, secrets or `min_server_version`.
