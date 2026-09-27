# Upgrading a Northframe application

Update the `northframe` executable first, then preview the application refresh:

```sh
northframe upgrade --check
```

The preview compiles routes and reports only the framework-managed files that
would change. It does not write application files.

Apply the refresh with:

```sh
northframe upgrade
```

Northframe writes only the protected `.generated/routes` tree—including the
route-mirrored `props_generated.go` contracts—builds the application into a temporary
directory, and restores the previous generated output if that build fails. It
does not rewrite handwritten templates, loaders, components, services,
database files, `northframe.toml`, or dependency versions.

JavaScript packages have a separate lifecycle:

```sh
northframe add date-fns@^4.4.0
northframe update
```

`northframe add` changes declared dependencies. `northframe update` resolves those
declarations again. `northframe upgrade` intentionally leaves both alone so a
framework refresh cannot silently change browser-library behavior.

Restart a running `northframe run` process after replacing the `northframe` executable.
An already-running watcher keeps the older compiler in memory until restart.
