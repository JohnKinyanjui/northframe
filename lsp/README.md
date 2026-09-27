# Northframe editor support

Northframe's language server is implemented in Go under `internal/lsp` and is
started with:

```sh
northframe lsp
```

The `vscode` directory contains the dependency-free Visual Studio Code
extension. Other editors can connect directly to the same stdio command.
