"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const {
  applyOffsetEdits,
  dedentBlock,
  formatPropsBlock,
  formatMarkupIndentation,
  formatNorthframeBlocks,
  formattingPreservesTokens,
  formattingTokenSignature,
  formattingTemplate,
  htmlVirtualContent,
  layoutDocumentMarkup,
  layoutMarkup,
  layoutServerDirectives,
  normalizeServerDirectives,
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

test("markup indentation preserves embedded TypeScript structure", () => {
  const source = `<script lang="ts">
  function state(value: string): string {
    const normalized = value.toLowerCase();
    if (normalized === "ready") return "active";
    return "idle";
  }
</script>
<main>Ready</main>`;
  assert.equal(formatMarkupIndentation(source), source);
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

test("Props fields align like Go declarations", () => {
  const formatted = formatPropsBlock(`interface Props {
Name string
Value string
Placeholder string
ClearHref string
}`);
  assert.equal(formatted, `interface Props {
  Name        string
  Value       string
  Placeholder string
  ClearHref   string
}`);
});

test("markup layout separates children and wraps long attribute lists", () => {
  const source = `<label class="relative block"><span class="sr-only">Search</span><input name="q" value="" type="search" autocomplete="off" placeholder="Search"></label>`;
  const formatted = layoutMarkup(source);
  assert.equal(formatted, `<label class="relative block">
<span class="sr-only">Search</span>
<input
name="q"
value=""
type="search"
autocomplete="off"
placeholder="Search">
</label>`);
  assert.equal(layoutMarkup(formatted), formatted);
  assert.equal(formattingPreservesTokens(source, formatted), true);
});

test("markup indentation follows HTML and Northframe block nesting", () => {
  const source = `<label class="relative">
<span>Name</span>
<input
name="q"
placeholder="Search"
>
{if Props.ClearHref}
<a href="{Props.ClearHref}">Clear</a>
{/if}
</label>`;
  const formatted = formatMarkupIndentation(layoutDocumentMarkup(source), "    ");
  assert.equal(formatted, `<label class="relative">
    <span>Name</span>
    <input
        name="q"
        placeholder="Search">
    {if Props.ClearHref}
        <a href="{Props.ClearHref}">Clear</a>
    {/if}
</label>`);
  assert.equal(formatMarkupIndentation(layoutDocumentMarkup(formatted), "    "), formatted);
});

test("document markup layout preserves scripts and is stable", () => {
  const source = `<script lang="ts">\nfunction compare(a: number, b: number): boolean { return a < b; }\n</script>\n<label><input name="q" value="" type="search" autocomplete="off" class="field"></label>`;
  const once = formatMarkupIndentation(layoutDocumentMarkup(source), "    ");
  const twice = formatMarkupIndentation(layoutDocumentMarkup(once), "    ");
  assert.match(once, /return a < b;/);
  assert.match(once, /<input\n        name="q"/);
  assert.equal(twice, once);
  assert.equal(formattingPreservesTokens(source, once), true);
});

test("multiline tags keep their closing bracket with the final attribute", () => {
  const source = `<button data-close-dialog type="button" aria-label="Close activity dialog" class="flex h-9 w-9 items-center justify-center"></button>\n<SearchField Label="Search activity sheet" Placeholder="Search event, aircraft, mission, location, or decision" />`;
  const formatted = formatMarkupIndentation(layoutDocumentMarkup(source));
  assert.equal(formatted, `<button
  data-close-dialog
  type="button"
  aria-label="Close activity dialog"
  class="flex h-9 w-9 items-center justify-center">
</button>
<SearchField
  Label="Search activity sheet"
  Placeholder="Search event, aircraft, mission, location, or decision" />`);
  assert.equal(formatMarkupIndentation(layoutDocumentMarkup(formatted)), formatted);
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
  const restored = restoreFormattedScripts(template, [formatPropsBlock(scriptRegions(propsSource)[0].content, "")]);
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
  assert.equal(formatted, `{for section := range Props.Sections}
  <section>
    {if section.Title}
      <h2>{section.Title}</h2>
    {/if}
  </section>
{/for}`);
  assert.equal(formatNorthframeBlocks(formatted), formatted);
  assert.equal((formatted.match(/<\/section>/g) || []).length, 1);
  assert.equal((formatted.match(/\{\/for\}/g) || []).length, 1);
});

test("multiline server directives are normalized and protected from HTML formatting", () => {
  const source = `{if
  !Props.CashEnabled}<span>Disabled</span>{/if}
{for
  item := range Props.Items}<span>{item.Name}</span>{/for}`;
  assert.equal(
    normalizeServerDirectives(source),
    `{if !Props.CashEnabled}<span>Disabled</span>{/if}\n{for item := range Props.Items}<span>{item.Name}</span>{/for}`,
  );
  const template = formattingTemplate(source);
  assert.doesNotMatch(template.source, /\{(?:if|for|\/(?:if|for))/);
  assert.equal(template.directives.length, 4);
  const restored = restoreFormattedScripts(template, []);
  assert.equal(restored, `{if !Props.CashEnabled}\n<span>Disabled</span>\n{/if}\n{for item := range Props.Items}\n<span>{item.Name}</span>\n{/for}\n`);
});

test("block directives get readable lines while conditional attributes stay inline", () => {
  const source = `<section>{if Props.Description}<p>{Props.Description}</p>{/if}</section>
<input type="checkbox" {if Props.Enabled}checked{/if}>`;
  const laidOut = layoutServerDirectives(source);
  assert.equal(laidOut, `<section>\n{if Props.Description}\n<p>{Props.Description}</p>\n{/if}\n</section>\n<input type="checkbox" {if Props.Enabled}checked{/if}>`);
  const template = formattingTemplate(source);
  assert.match(template.source, /<!--__NORTHFRAME_DIRECTIVE_0__-->/);
  assert.match(template.source, /data-northframe-directive-2="" checked data-northframe-directive-3=""/);
  assert.doesNotMatch(template.source, /<input[^>]*<!--/);
  assert.equal(formattingPreservesTokens(source, restoreFormattedScripts(template, [])), true);
});

test("format safety rejects deleted script tokens and duplicated markup", () => {
  const script = `function close(): void {\n  dispatch("close");\n}`;
  assert.equal(formattingPreservesTokens(script, `gu nction close(): void {\n  spatch("close");\n}`), false);
  const markup = `<div><span>Ready</span></div>`;
  assert.equal(formattingPreservesTokens(markup, `${markup}</div>`), false);
  assert.equal(formattingPreservesTokens(markup, `<div>\n  <span>Ready</span>\n</div>`), true);
  assert.equal(formattingPreservesTokens(`function close(): void`, `function close(): voi d`), false);
  assert.equal(formattingPreservesTokens(`dispatch("close")`, `dispatch("clos e")`), false);
  assert.equal(formattingTokenSignature(`function close():void`), formattingTokenSignature(`function close(): void`));
});

test("script restoration preserves complete event functions", () => {
  const source = `<script lang="ts">\nfunction close(): void {\n  dispatch("close");\n}\n</script>\n<button on:click={close}></button>`;
  const template = formattingTemplate(source);
  const formattedScript = `function close(): void {\n  dispatch("close");\n}`;
  const restored = restoreFormattedScripts(template, [formattedScript]);
  assert.match(restored, /function close\(\): void \{/);
  assert.match(restored, /dispatch\("close"\);/);
  assert.match(restored, /<script lang="ts">\n  function close/);
  assert.equal(formattingPreservesTokens(source, restored), true);
});

test("embedded scripts dedent once and remain stable across saves", () => {
  const indented = `
  import "iconify-icon";

  function close(): void {
    dispatch("close");
  }
`;
  assert.equal(dedentBlock(indented), `import "iconify-icon";

function close(): void {
  dispatch("close");
}`);
  assert.equal(dedentBlock(dedentBlock(indented)), dedentBlock(indented));
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
