# `northframe.toml` versus `package.json`

Northframe should keep `northframe.toml`. It is not intended to reproduce the
Node package manifest.

| Concern | Northframe application | Node application |
| --- | --- | --- |
| Go module and server dependencies | `go.mod` and `go.sum` | Not applicable |
| Northframe compiler settings | `northframe.toml` | Framework-specific configuration |
| Direct browser dependencies | `northframe.toml` | `package.json` |
| Exact browser dependency graph | `northframe.lock` | Package-manager lockfile |
| Downloaded browser packages | `.northframe/` | `node_modules/` or another package store |
| Development/build scripts | `northframe` commands and Go tooling | `package.json` scripts |
| Runtime | One Go executable | Commonly Node.js or a JavaScript runtime |

The narrower file gives Northframe room for route, compiler, client-source,
and future deployment settings without claiming that the application is a
publishable JavaScript package. Browser dependencies remain normal typed
imports, but Northframe downloads and verifies them without running lifecycle
scripts.

Developers may keep a separate `package.json` for unrelated tooling if they
choose, but Northframe does not require or read it. A Northframe application
should not need Node merely because one page imports a date or chart library.
