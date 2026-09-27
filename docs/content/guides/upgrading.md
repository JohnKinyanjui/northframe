# Upgrading

Northframe separates three kinds of change: replacing the `northframe` executable, regenerating framework-owned output, and updating browser dependencies. An upgrade must not rewrite application routes, services, database files, or UI.

## Replace the CLI

Install the newer command using the same Go workflow as the first installation:

```sh
go install github.com/JohnKinyanjui/northframe/cmd/northframe@v0.1.0-beta
northframe help
```

Restart `northframe run` and the VS Code language server after replacing the executable. A running process keeps the compiler version it started with.

Northframe is currently beta. Pin an exact tag such as `@v0.1.0-beta` in development images and production automation rather than assuming `@latest` remains unchanged. Read the release notes before moving between beta tags because compatibility is not frozen until `v1.0.0`.

## Preview generated changes

From the application root:

```sh
northframe upgrade --check
```

The command compiles the current sources in memory and reports which files under `.generated/routes` would be created, refreshed, or removed. It does not write files in check mode.

Review the output. A large change can be legitimate after compiler work, but it should still correspond to generated renderers, route registration, TypeScript contracts, or assets—not handwritten code.

## Apply the upgrade

```sh
northframe upgrade
```

Northframe reads the existing generated tree, writes the new generated files, and builds the application into a temporary directory. If validation fails, it restores the previous generated files. A successful upgrade changes only Northframe-managed generated sources.

The command does not rewrite:

- `.north` templates or `.north.go` sidecars
- components, services, middleware, or tests
- migrations, sqlc queries, or generated database code
- `northframe.toml` or browser dependency constraints
- application dependency versions in `go.mod`

## Update browser packages separately

```sh
northframe add date-fns@4.1.0
northframe update
```

`northframe add` changes the manifest constraint and refreshes `northframe.lock`. `northframe update` resolves all current constraints. Review dependency changes independently from framework generation.

## Verify the application

```sh
northframe generate
go test ./...
northframe deploy check
git diff
```

Open representative pages and test client interactions because a compile proves contracts, not visual or browser behavior. If the language server reports behavior from an older compiler, restart the server and verify which `northframe` path the extension uses.

If validation fails, correct the source diagnostic and rerun generation. Never patch `.generated` to make an upgrade pass; the next compile would erase the change.
