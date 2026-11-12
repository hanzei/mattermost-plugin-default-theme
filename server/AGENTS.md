# AGENTS.md (server)

## Rules

- `UserHasBeenCreated` runs for every new user and `UserHasJoinedTeam` for every team join. Keep them fast, never panic, and log failures with `p.API.LogError` using key/value pairs (`"user_id", user.Id, "err", err.Error()`) rather than formatting them into the message.
- The plugin runs on every cluster node. Don't rely on in-memory state being shared between nodes.
- Log IDs, not theme or user content.
- If you add HTTP routes, slash commands, KV storage or other entry points, follow the security rules in the starter template's [`server/AGENTS.md`](https://github.com/mattermost/mattermost-plugin-starter-template/blob/master/server/AGENTS.md): take the acting user only from the `Mattermost-User-ID` header, check permissions on the server, validate IDs with `model.IsValidId`, and write audit records for state changes.
- Before releasing: `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`.

## Tests

- The existing tests are unit tests against a mocked `plugintest.API` (`plugin_test.go`, `configuration_test.go`); follow that pattern for hook and configuration logic.
- For behavior that depends on a real server (preferences actually persisting, hook wiring), prefer integration tests with `github.com/mattermost/mattermost/server/public/pluginapi/testhelper`, which runs Mattermost and Postgres via testcontainers. None exist yet, so adding one also means re-adding the starter template's `min-server-version` CI job.
- Use testify `require`/`assert`, table-driven when there's more than one case of the same shape, and no new test or mocking libraries.
