# CLI reference

Run commands from the project root containing `go.mod` and `web/routes`. Most commands discover the Go module and Northframe manifest by walking upward, but route compilation intentionally uses project-relative paths.

## Create a project

```sh
north create my-app
north create . --module example.com/team/shop
north create docs --template=docs
```

`--template` accepts `app` or `docs`. `--module` overrides the module derived from the directory name. The command checks every target file before writing and refuses to overwrite conflicting application files.

## Generate, run, and build

```sh
north generate
north run
north run -port 8123
north build -o ./bin/app
```

`generate` compiles views, components, API discovery, CSS, browser modules, and route registration into `.generated/routes` without starting a listener.

`run` loads `.env`, compiles the app, watches project files, and owns a stable public listener. A successful rebuild starts a new child on a private port, waits until it is ready, switches traffic, reloads browsers, and stops the old child. A compile failure keeps the last good child serving. Ctrl+C or SIGTERM gracefully stops the supervisor and child.

`build` regenerates, then runs Go build. Use `-target ./cmd/server` when the main package is elsewhere.

Advanced project flags shared by generation commands include `-routes`, `-api`, `-generated`, `-package`, `-route-import`, and `-api-import`. Most applications should keep their defaults.

## Database commands

```text
north db generate [-config sqlc.yaml]
north db create [-dir internal/db/migrations] NAME
north db migrate
north db rollback
north db status
north db version
north db verify
north db adopt
north db seed
```

`db generate` runs the installed `sqlc` executable. Migration operations run the application command at `cmd/migrator` or `cmd/migrate` and forward the operation. Seeding runs `cmd/seeder` or `cmd/seed`. These commands load `.env` while preserving values already exported by the shell.

## Browser dependency commands

```sh
north add date-fns
north add chart.js@4.5.0
north remove chart.js
north update
```

They update `northframe.toml`, resolve exact packages into `northframe.lock`, and install the local compiler store under `.northframe`. They do not create an application `package.json` or require a global Node installation.

## Upgrade and deploy

```sh
north upgrade --check
north upgrade
north deploy check
north deploy check -target ./cmd/server
north deploy docker -output Dockerfile
```

Upgrade refreshes only generated framework output and restores it if the validation build fails. Deployment check generates and builds a temporary production executable. Docker prints to standard output by default; `-output` writes a file.

## Language server

```sh
north lsp
```

The command speaks LSP over standard input and output and is normally started by the VS Code extension. Do not run it in an interactive terminal expecting a prompt.

If a command says `web/routes` is missing, verify the current directory. If `north` is not found, add `$(go env GOPATH)/bin` or `GOBIN` to `PATH`. `runserver` and `dev` remain compatibility aliases, but new documentation uses `north run`.
