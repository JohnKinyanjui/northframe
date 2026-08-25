"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const {
  applyOffsetEdits,
  formatPropsBlock,
  formatNorthframeBlocks,
  formattingTemplate,
  htmlVirtualContent,
  restoreFormattedScripts,
  scriptRegionAt,
  scriptRegions,
  typescriptVirtualContent,
} = require("../embedded");

const source = `<main>🧭</main>
<script lang="ts">
let count:number=1
function add(){count+=1}
</script>
<button class="px-4" on:click={add}>{#count}</button>`;

test("embedded documents preserve source positions", () => {
  const regions = scriptRegions(source);
  assert.equal(regions.length, 1);
  assert.equal(regions[0].kind, "typescript");
  assert.equal(scriptRegionAt(source, regions[0].contentStart + 2).kind, "typescript");
  const typescript = typescriptVirtualContent(source);
  const html = htmlVirtualContent(source);
  assert.equal(typescript.length, source.length);
  assert.equal(html.length, source.length);
  assert.equal(typescript.slice(regions[0].contentStart, regions[0].contentEnd), regions[0].content);
  assert.match(html.slice(regions[0].contentStart, regions[0].contentEnd), /^\s+$/);
});

test("composite formatting restores formatted scripts", () => {
  const template = formattingTemplate(source);
  assert.match(template.source, /__NORTHFRAME_SCRIPT_0__/);
  const restored = restoreFormattedScripts(template, ["let count: number = 1;\nfunction add() {\n  count += 1;\n}"]);
  assert.doesNotMatch(restored, /__NORTHFRAME_SCRIPT/);
  assert.match(restored, /let count: number = 1;/);
  assert.match(restored, /  count \+= 1;/);
});

test("props blocks get predictable import and field spacing", () => {
  const formatted = formatPropsBlock(`
 import   models   "example.test/models"
Items     []models.Item
`);
  assert.equal(formatted, `  import models "example.test/models"\n  Items []models.Item`);
});

test("Props frontmatter is discovered and formatted", () => {
  const propsSource = `---
import   models   "example.test/models"

interface Props {
Items     []models.Item
}
---
<main></main>`;
  const regions = scriptRegions(propsSource);
  assert.equal(regions[0].kind, "props");
  assert.equal(formatPropsBlock(regions[0].content), `import models "example.test/models"

interface Props {
  Items []models.Item
}`);
  assert.match(htmlVirtualContent(propsSource), /^---\n\s+\n---/);
});

test("composite formatting keeps Props imports inside frontmatter", () => {
  const propsSource = `---

import   models   "example.test/models"

interface Props {
Items     []models.Item
}

---
<main></main>`;
  const template = formattingTemplate(propsSource);
  assert.match(template.source, /^<!--__NORTHFRAME_PROPS_0__-->/);
  const restored = restoreFormattedScripts(template, [formatPropsBlock(scriptRegions(propsSource)[0].content)]);
  assert.match(restored, /^---\nimport models "example\.test\/models"\n\ninterface Props \{\n  Items \[\]models\.Item\n\}\n---/);
  assert.equal((restored.match(/import models/g) || []).length, 1);
});

test("composite restoration preserves props indentation", () => {
  const propsSource = `<script context="props">
import   models   "example.test/models"
Items []models.Item
</script>`;
  const template = formattingTemplate(propsSource);
  const restored = restoreFormattedScripts(template, [formatPropsBlock(scriptRegions(propsSource)[0].content)]);
  assert.match(restored, /\n  import models "example\.test\/models"\n  Items \[\]models\.Item\n<\/script>/);
});

test("server control blocks indent component bodies", () => {
  const source = `<section>
  {for item:=range Props.Items}
  <InventoryCard Item={item} />
  {/for}
</section>`;
  const formatted = formatNorthframeBlocks(source);
  assert.equal(formatted, `<section>
  {for item := range Props.Items}
    <InventoryCard Item={item} />
  {/for}
</section>`);
  assert.equal(formatNorthframeBlocks(formatted), formatted);
});

test("server formatter preserves inline blocks and closing tags", () => {
  const source = `{for section := range Props.Sections}
  <section>
    {if section.Title}<h2>{section.Title}</h2>{/if}
  </section>
{/for}`;
  const formatted = formatNorthframeBlocks(source);
  assert.equal(formatted, source);
  assert.equal((formatted.match(/<\/section>/g) || []).length, 1);
  assert.equal((formatted.match(/\{\/for\}/g) || []).length, 1);
});

test("overlapping formatter edits cannot duplicate component tails", () => {
  const source = `<header><button>Search</button></header>`;
  const formatted = `<header>\n  <button>Search</button>\n</header>`;
  const result = applyOffsetEdits(source, [
    { start: 0, end: source.length, newText: formatted },
    { start: 8, end: 31, newText: `\n  <button>Search</button>\n` },
  ]);
  assert.equal(result, formatted);
  assert.equal((result.match(/<\/header>/g) || []).length, 1);
});
