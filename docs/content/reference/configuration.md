# Configuration

Northframe deliberately separates build configuration, Go dependencies, browser dependencies, and runtime secrets. This prevents one large configuration file from becoming a second application language.

## `northframe.toml`

The manifest currently configures shared browser source and browser package constraints:

```toml
[client]
source = "web/client"

[client.dependencies]
"date-fns" = "4.1.0"
"@floating-ui/dom" = "^1.7.4"
```

`client.source` is the directory used by `$client/...` imports in `<script lang="ts">`. It defaults to `web/client`. Keep it inside the project and outside generated output.

The dependency map accepts browser package names and version constraints. Prefer an exact version while Northframe is in beta and loosen constraints only when the team is prepared to review resolution changes.

## Lockfile and local store

`northframe.lock` records exact versions, resolution URLs, integrity hashes, constraints, and dependency edges. Commit both the manifest and lockfile.

`.northframe/modules/node_modules` is the local compiler store. It is ignored and recreated with:

```sh
northframe update
```

The directory layout lets esbuild resolve browser packages, but it does not make Node part of the application workflow or deployment.

Set `NORTHFRAME_NPM_REGISTRY` to use a compatible registry mirror:

```sh
NORTHFRAME_NPM_REGISTRY=https://registry.example.com northframe update
```

## `go.mod`

Go owns server dependencies and the module path:

```go
module example.com/team/shop

go 1.27

require github.com/JohnKinyanjui/northframe v0.1.0-beta
```

Frontmatter imports and sidecar imports resolve through this module. Changing the module path requires updating generated import references by running `northframe generate` and updating handwritten imports that used the old module.

## Environment values

`northframe run` and database helper commands read `.env`. Values already exported in the shell win:

```env
PORT=8000
DATABASE_URL=postgresql://postgres:postgres@localhost:5432/shop
SECRET_KEY=development-only-value
SMTP_HOST=localhost
```

Ignore `.env`. Commit an `.env.example` containing names and safe placeholders when the application needs onboarding documentation.

Production configuration should come from the process environment or a secret manager. Northframe does not automatically validate application-specific variables; validate required settings at startup and fail before accepting traffic.

## Generated and ignored directories

```gitignore
.env
.generated/
.northframe/
app
```

`.generated/routes` contains compiler output. `.northframe` contains restored browser packages. Both are reproducible and should not be used for handwritten application code.

## There is no application `package.json`

Northframe applications do not need `package.json`. The VS Code extension has its own manifest because VS Code extensions are JavaScript packages; that editor tooling file is not application configuration and is not copied into a Northframe project.

When configuration appears not to apply, verify the project root, inspect `northframe.toml` syntax, run `northframe update` after changing dependencies, and restart long-running `northframe run` or language-server processes after replacing the CLI.
