"use strict";

const scriptPattern = /<script\b([^>]*)>([\s\S]*?)<\/script\s*>/gi;
const propsFrontmatterPattern = /^---[ \t]*\r?\n([\s\S]*?)^---[ \t]*(?:\r?\n|$)/gm;
const serverDirectivePattern = /\{(?:\/(?:if|for)|if\s+[\s\S]*?|for\s+[A-Za-z_][A-Za-z0-9_]*\s*:=\s*range\s+[\s\S]*?)\}/g;

function normalizeServerDirectives(source) {
  return source.replace(serverDirectivePattern, (directive) =>
    directive.replace(/^\{(if|for)\s+/, "{$1 "));
}

function scriptRegions(source) {
  const regions = [];

  for (const match of source.matchAll(propsFrontmatterPattern)) {
    if (match.index !== 0) continue;
    const whole = match[0];
    const content = match[1] || "";
    const contentStart = match.index + whole.indexOf(content);
    regions.push({
      kind: "props",
      frontmatter: true,
      start: match.index,
      end: match.index + whole.length,
      contentStart,
      contentEnd: contentStart + content.length,
      content,
    });
  }
  for (const match of source.matchAll(scriptPattern)) {
    const attributes = match[1] || "";
    const whole = match[0];
    const content = match[2] || "";
    const contentStart = match.index + whole.indexOf(content);
    let kind = "script";
    if (/\blang\s*=\s*(?:"ts"|'ts')/i.test(attributes)) kind = "typescript";
    if (/\bcontext\s*=\s*(?:"props"|'props')/i.test(attributes)) kind = "props";
    regions.push({
      kind,
      start: match.index,
      end: match.index + whole.length,
      contentStart,
      contentEnd: contentStart + content.length,
      content,
    });
  }
  return regions.sort((left, right) => left.start - right.start);
}

function scriptRegionAt(source, offset) {
  return scriptRegions(source).find((region) => offset >= region.contentStart && offset <= region.contentEnd);
}

function typescriptVirtualContent(source) {
  const regions = scriptRegions(source).filter((region) => region.kind === "typescript");
  const included = new Array(source.length).fill(false);
  for (const region of regions) {
    for (let index = region.contentStart; index < region.contentEnd; index++) included[index] = true;
  }
  return source.split("").map((character, index) => character === "\n" || character === "\r" || included[index] ? character : " ").join("");
}

function htmlVirtualContent(source) {
  const characters = source.split("");
  for (const region of scriptRegions(source)) {
    for (let index = region.contentStart; index < region.contentEnd; index++) {
      if (characters[index] !== "\n" && characters[index] !== "\r") characters[index] = " ";
    }
  }
  return characters.join("");
}

function formattingTemplate(source) {
  const regions = scriptRegions(source);
  let result = source;
  for (let index = regions.length - 1; index >= 0; index--) {
    const region = regions[index];
    const marker = region.frontmatter
      ? `<!--__NORTHFRAME_PROPS_${index}__-->`
      : `/*__NORTHFRAME_SCRIPT_${index}__*/`;
    if (region.frontmatter) {
      result = result.slice(0, region.start) + marker + "\n" + result.slice(region.end);
    } else {
      result = result.slice(0, region.contentStart) + `\n${marker}\n` + result.slice(region.contentEnd);
    }
  }
  const directives = [];
  result = normalizeServerDirectives(result).replace(serverDirectivePattern, (value) => {
    const marker = `<!--__NORTHFRAME_DIRECTIVE_${directives.length}__-->`;
    directives.push({ marker, value });
    return marker;
  });
  return {
    source: result,
    directives,
    regions: regions.map((region, index) => ({
      ...region,
      marker: region.frontmatter
        ? `<!--__NORTHFRAME_PROPS_${index}__-->`
        : `/*__NORTHFRAME_SCRIPT_${index}__*/`,
      replaceWhole: Boolean(region.frontmatter),
    })),
  };
}

function restoreFormattedScripts(template, formattedContents) {
  let result = template.source;
  for (let index = 0; index < template.regions.length; index++) {
    const marker = template.regions[index].marker;
    const markerOffset = result.indexOf(marker);
    if (markerOffset < 0) continue;
    if (template.regions[index].replaceWhole) {
      const content = String(formattedContents[index] || "").trim();
      const replacement = `---\n${content}\n---`;
      result = result.slice(0, markerOffset) + replacement + result.slice(markerOffset + marker.length);
      continue;
    }
    const lineStart = result.lastIndexOf("\n", markerOffset) + 1;
    const indentation = /^\s*/.exec(result.slice(lineStart, markerOffset))[0];
    const content = String(formattedContents[index] || "")
      .replace(/^\s*\r?\n/, "")
      .replace(/\r?\n\s*$/, "");
    const replacement = content.split("\n").join("\n" + indentation);
    result = result.slice(0, markerOffset) + replacement + result.slice(markerOffset + marker.length);
  }
  for (const directive of template.directives || []) {
    result = result.replace(directive.marker, directive.value);
  }
  return result.replace(/[ \t]+\n/g, "\n").replace(/^\s*\n/, "").replace(/\s*$/, "\n");
}

function formatPropsBlock(source, indentation = "  ") {
  const lines = source
    .split(/\r?\n/)
    .map((line) => line.trim())
    .filter(Boolean);
  const interfaceStart = lines.findIndex((line) => /^interface\s+Props\s*\{$/.test(line));
  if (interfaceStart >= 0) {
    const interfaceEnd = lines.findIndex((line, index) => index > interfaceStart && line === "}");
    const outside = lines.filter((_, index) => index < interfaceStart || index > interfaceEnd);
    const imports = formatPropLines(outside);
    const body = formatPropLines(lines.slice(interfaceStart + 1, interfaceEnd < 0 ? lines.length : interfaceEnd));
    const contract = `interface Props {${body.length ? `\n${body.map((line) => indentation + line).join("\n")}\n` : ""}}`;
    return imports.length ? `${imports.join("\n")}\n\n${contract}` : contract;
  }
  return formatPropLines(lines).map((line) => indentation + line).join("\n");
}

function formatPropLines(lines) {
  return lines
    .map((line) => {
      if (/^\/\//.test(line)) return line;
      const imported = /^import\s+([A-Za-z_][A-Za-z0-9_]*)\s+(.+)$/.exec(line);
      if (imported) return `import ${imported[1]} ${imported[2]}`;
      return line.replace(/\s+/g, " ");
    });
}

function formatNorthframeBlocks(source, indentation = "  ") {
  const openBlocks = [];
  return normalizeServerDirectives(source)
    .split(/\r?\n/)
    .map((line) => {
      const trimmed = line.trim();
      const leading = /^\s*/.exec(line)[0];

      if (/^\{\/(?:if|for)\}$/.test(trimmed)) {
        const matchingIndent = openBlocks.pop();
        return matchingIndent === undefined ? line : matchingIndent + trimmed;
      }

      let directive = trimmed;
      const loop = /^\{for\s+([A-Za-z_][A-Za-z0-9_]*)\s*:=\s*range\s+(.+)\}$/.exec(trimmed);
      if (loop) directive = `{for ${loop[1]} := range ${loop[2].trim()}}`;
      const condition = /^\{if\s+(.+)\}$/.exec(trimmed);
      if (condition) directive = `{if ${condition[1].trim()}}`;

      let formatted = directive === trimmed ? line : leading + directive;
      if (trimmed && openBlocks.length > 0) {
        const minimumIndent = openBlocks[openBlocks.length - 1] + indentation;
        if (!leading.startsWith(minimumIndent)) formatted = minimumIndent + directive;
      }

      if (/^\{(?:if|for)\b[^}]*\}$/.test(directive)) {
        openBlocks.push(/^\s*/.exec(formatted)[0]);
      }
      return formatted;
    })
    .join("\n");
}

function applyOffsetEdits(source, edits) {
  const ordered = [...edits].sort((left, right) => left.start - right.start || right.end - left.end);
  const accepted = [];
  for (const edit of ordered) {
    const previous = accepted[accepted.length - 1];
    if (previous && edit.start < previous.end) continue;
    accepted.push(edit);
  }
  let result = source;
  for (let index = accepted.length - 1; index >= 0; index--) {
    const edit = accepted[index];
    result = result.slice(0, edit.start) + edit.newText + result.slice(edit.end);
  }
  return result;
}

module.exports = {
  applyOffsetEdits,
  formatPropsBlock,
  formatNorthframeBlocks,
  formattingTemplate,
  htmlVirtualContent,
  normalizeServerDirectives,
  restoreFormattedScripts,
  scriptRegionAt,
  scriptRegions,
  typescriptVirtualContent,
};
