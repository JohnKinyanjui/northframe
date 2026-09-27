# Client dependencies

Northframe applications can use browser JavaScript libraries without Node,
Deno, npm, or an application `package.json`.

## Commands

```sh
northframe add date-fns
northframe add chart.js@4.5.0
northframe add @floating-ui/dom@^1.7.0
northframe remove chart.js
northframe update
```

An omitted version is saved as `latest`. An explicit version, dist-tag, or
semantic-version range is preserved in `northframe.toml`. `northframe update`
resolves that intent again and recreates the local package store.

Set `NORTHFRAME_NPM_REGISTRY` to use a compatible registry mirror. The default
is `https://registry.npmjs.org`.

## Files

```toml
[client]
source = "web/client"

[client.dependencies]
"date-fns" = "latest"
"@floating-ui/dom" = "^1.7.0"
```

Commit `northframe.toml` and `northframe.lock`. Do not commit `.northframe/`.
The lockfile contains exact versions, dependency edges, source tarballs, and
integrity values so CI and another developer can reconstruct the same inputs.

## Imports

```html
<script lang="ts">
import { format } from "date-fns";
import { positionMenu } from "$client/menu";
import confetti from "https://esm.sh/canvas-confetti@1.9.3";
</script>
```

- Bare imports must be declared with `northframe add`.
- Relative imports resolve from the `.north` component containing the script.
- `$client/` resolves from `client.source` and supports `.ts`, `.tsx`, `.js`,
  `.jsx`, and matching `index` modules.
- HTTP(S) imports stay external and are fetched by the browser.

The compiler creates one ESM bundle for each interactive route or independent
component and embeds it in the generated Go router. Tree shaking is handled by
esbuild.

## Compatibility boundary

The first dependency format intentionally supports browser-compatible ESM and
CommonJS packages. Northframe does not execute lifecycle scripts and does not
provide Node built-ins or native addons. Packages that depend on those features
fail during installation or compilation with an actionable error.

CSS imported by a JavaScript package is not emitted yet. Keep application styles
in colocated `page.css` or `layout.css` until CSS bundle assets are supported.
