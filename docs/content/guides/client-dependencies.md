# Browser dependencies

Northframe can bundle browser-compatible JavaScript libraries without making Node or a package manager part of the application workflow.

```sh
northframe add date-fns
northframe add chart.js@4.5.0
northframe update
northframe remove chart.js
```

Declarations live in `northframe.toml`; exact resolutions live in `northframe.lock`:

```toml
[client]
source = "web/client"

[client.dependencies]
"date-fns" = "latest"
```

Import them from a TypeScript script:

```html
<script lang="ts">
import { format } from "date-fns";
let today: string = format(new Date(), "yyyy-MM-dd");
</script>
```

Relative imports resolve from the component. `$client/` resolves from the configured client source. HTTP(S) ESM imports remain external. Northframe bundles interactive route/component modules with esbuild and tree-shakes the result. Packages requiring Node built-ins, native addons, or lifecycle scripts are outside the supported browser boundary.

Commit `northframe.toml` and `northframe.lock`; ignore `.northframe/`.

Packages must be browser-compatible. A package that expects Node built-ins, native addons, or lifecycle scripts may install unsuccessfully or fail during bundling; it cannot be made available to Go loaders by adding it to this manifest.

## A complete example

Create `web/routes/orders/page.north` with a TypeScript import, run `northframe add date-fns`, and then `northframe run`:

```html
<script lang="ts">
import { format } from "date-fns";
let checkedAt: string = format(new Date(), "yyyy-MM-dd HH:mm");
</script>
<main><h1>Orders</h1><p>Checked at #{checkedAt}</p></main>
```

Northframe writes the declaration to `northframe.toml`, records exact versions in `northframe.lock`, and bundles the import. Use bare imports for managed packages, relative imports for nearby modules, and `$client/` for shared client modules. Do not import a browser package into a Go sidecar: Go dependencies belong in `go.mod`.

Commit the manifest and lockfile so CI can run `northframe update` reproducibly. If the package expects Node built-ins or native addons, choose a browser ESM build or another library before continuing.
