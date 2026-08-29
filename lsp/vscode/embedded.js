"use strict";

const scriptPattern = /<script\b([^>]*)>([\s\S]*?)<\/script\s*>/gi;
const propsFrontmatterPattern = /^---[ \t]*\r?\n([\s\S]*?)^---[ \t]*(?:\r?\n|$)/gm;
const serverDirectivePattern = /\{(?:\/(?:if|for)|if\s+[\s\S]*?|for\s+[A-Za-z_][A-Za-z0-9_]*\s*:=\s*range\s+[\s\S]*?)\}/g;

function normalizeServerDirectives(source) {
  return source.replace(serverDirectivePattern, (directive) =>
    directive.replace(/^\{(if|for)\s+/, "{$1 "));
}

function insideHTMLTagAt(source, offset) {
  return source.lastIndexOf("<", offset) > source.lastIndexOf(">", offset);
}

function layoutServerDirectives(source) {
  return normalizeServerDirectives(source).replace(serverDirectivePattern, (directive, offset, document) => {
    if (insideHTMLTagAt(document, offset)) return directive;
    const before = document.slice(0, offset).trimEnd();
    const after = document.slice(offset + directive.length).trimStart();
    const opensMarkup = !directive.startsWith("{/") && after.startsWith("<");
    const closesMarkup = directive.startsWith("{/") && before.endsWith(">");
    if (!opensMarkup && !closesMarkup) return directive;
    const lineStart = document.lastIndexOf("\n", offset - 1) + 1;
    const nextLine = document.indexOf("\n", offset + directive.length);
    const lineEnd = nextLine < 0 ? document.length : nextLine;
    const atLineStart = /^\s*$/.test(document.slice(lineStart, offset));
    const atLineEnd = /^\s*$/.test(document.slice(offset + directive.length, lineEnd));
    return `${atLineStart ? "" : "\n"}${directive}${atLineEnd ? "" : "\n"}`;
  });
}

function splitTagAttributes(source) {
  const attributes = [];
  let current = "";
  let quote = "";
  let braces = 0;
  for (const character of source.trim()) {
    if (quote) {
      current += character;
      if (character === quote) quote = "";
      continue;
    }
    if (character === '"' || character === "'") {
      quote = character;
      current += character;
      continue;
    }
    if (character === "{") braces++;
    if (character === "}" && braces > 0) braces--;
    if (/\s/.test(character) && braces === 0) {
      if (current) attributes.push(current);
      current = "";
      continue;
    }
    current += character;
  }
  if (current) attributes.push(current);
  return attributes;
}

function layoutMarkup(source) {
  let result = "";
  let cursor = 0;
  while (cursor < source.length) {
    const start = source.indexOf("<", cursor);
    if (start < 0) {
      result += source.slice(cursor);
      break;
    }
    result += source.slice(cursor, start);
    const next = source[start + 1];
    if (!next || next === "/" || next === "!" || next === "?") {
      result += "<";
      cursor = start + 1;
      continue;
    }

    let quote = "";
    let braces = 0;
    let end = start + 1;
    for (; end < source.length; end++) {
      const character = source[end];
      if (quote) {
        if (character === quote) quote = "";
        continue;
      }
      if (character === '"' || character === "'") {
        quote = character;
        continue;
      }
      if (character === "{") braces++;
      if (character === "}" && braces > 0) braces--;
      if (character === ">" && braces === 0) break;
    }
    if (end >= source.length) {
      result += source.slice(start);
      break;
    }

    const tag = source.slice(start, end + 1);
    const parsed = /^<([A-Za-z][A-Za-z0-9:._-]*)([\s\S]*?)(\/?)>$/.exec(tag);
    if (!parsed) {
      result += tag;
      cursor = end + 1;
      continue;
    }
    const attributes = splitTagAttributes(parsed[2]);
    if (attributes.length >= 4 || tag.length > 100 || /\r?\n/.test(tag)) {
      const ending = parsed[3] ? " />" : ">";
      const last = attributes.length - 1;
      const formattedAttributes = attributes
        .map((attribute, index) => index === last ? attribute + ending : attribute)
        .join("\n");
      result += `<${parsed[1]}\n${formattedAttributes}`;
    } else {
      result += tag;
    }
    cursor = end + 1;
  }

  return result
    .replace(/>[ \t]*</g, ">\n<")
    .replace(/<([A-Za-z][A-Za-z0-9:._-]*)([^<>\r\n]*)>\n<\/\1>/gi, "<$1$2></$1>");
}

function formattingPreservesTokens(before, after) {
  return formattingTokenSignature(before) === formattingTokenSignature(after);
}

function formattingTokenSignature(source) {
  let result = "";
  let quote = "";
  let escaped = false;
  let whitespace = false;
  for (const character of String(source)) {
    if (quote) {
      result += character;
      if (escaped) {
        escaped = false;
      } else if (character === "\\") {
        escaped = true;
      } else if (character === quote) {
        quote = "";
      }
      continue;
    }
    if (/\s/.test(character)) {
      whitespace = true;
      continue;
    }
    if (whitespace) {
      const previous = result[result.length - 1] || "";
      if (/[A-Za-z0-9_$]/.test(previous) && /[A-Za-z0-9_$]/.test(character)) result += " ";
      whitespace = false;
    }
    result += character;
    if (character === '"' || character === "'" || character === "`") quote = character;
  }
  return result;
}

function dedentBlock(source) {
  const lines = String(source)
    .replace(/^\s*\r?\n/, "")
    .replace(/\r?\n\s*$/, "")
    .split(/\r?\n/);
  const indents = lines
    .filter((line) => line.trim())
    .map((line) => /^[ \t]*/.exec(line)[0].length);
  const minimum = indents.length ? Math.min(...indents) : 0;
  return lines.map((line) => line.slice(Math.min(minimum, /^[ \t]*/.exec(line)[0].length))).join("\n");
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

function layoutDocumentMarkup(source) {
  const regions = scriptRegions(source);
  const preserved = [];
  let result = source;
  for (let index = regions.length - 1; index >= 0; index--) {
    const region = regions[index];
    const marker = `__NORTHFRAME_PRESERVED_REGION_${index}__`;
    preserved[index] = { marker, content: region.content };
    result = result.slice(0, region.contentStart) + marker + result.slice(region.contentEnd);
  }
  result = layoutMarkup(result);
  for (const region of preserved) result = result.replace(region.marker, region.content);
  return result;
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

function formattingTemplate(source, indentation = "  ") {
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
      const lineStart = result.lastIndexOf("\n", region.start) + 1;
      const blockIndent = /^[ \t]*/.exec(result.slice(lineStart, region.start))[0];
      const markerIndent = blockIndent + indentation;
      result = result.slice(0, region.contentStart) + `\n${markerIndent}${marker}\n${blockIndent}` + result.slice(region.contentEnd);
    }
  }
  result = layoutMarkup(layoutServerDirectives(result));
  const directives = [];
  result = result.replace(serverDirectivePattern, (value, offset, document) => {
    const index = directives.length;
    const marker = insideHTMLTagAt(document, offset)
      ? `data-northframe-directive-${index}=""`
      : `<!--__NORTHFRAME_DIRECTIVE_${index}__-->`;
    directives.push({ marker, value });
    return insideHTMLTagAt(document, offset) ? ` ${marker} ` : marker;
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
    const imports = formatPropLines(outside).sort(comparePropImports);
    const body = formatPropFields(lines.slice(interfaceStart + 1, interfaceEnd < 0 ? lines.length : interfaceEnd));
    const contract = `interface Props {${body.length ? `\n${body.map((line) => indentation + line).join("\n")}\n` : ""}}`;
    return imports.length ? `${imports.join("\n")}\n\n${contract}` : contract;
  }
  return formatPropLines(lines).map((line) => indentation + line).join("\n");
}

function comparePropImports(left, right) {
  const leftImport = /^import\s+(?:[A-Za-z_][A-Za-z0-9_]*\s+)?["']([^"']+)["']$/.exec(left);
  const rightImport = /^import\s+(?:[A-Za-z_][A-Za-z0-9_]*\s+)?["']([^"']+)["']$/.exec(right);
  if (leftImport && rightImport) return leftImport[1].localeCompare(rightImport[1]);
  if (leftImport) return -1;
  if (rightImport) return 1;
  return 0;
}

function formatPropFields(lines) {
  const formatted = formatPropLines(lines);
  const parsed = formatted.map((line) => /^([A-Za-z_][A-Za-z0-9_]*)\s+(.+)$/.exec(line));
  const width = parsed.reduce((maximum, match) => match ? Math.max(maximum, match[1].length) : maximum, 0);
  return formatted.map((line, index) => {
    const match = parsed[index];
    return match ? `${match[1].padEnd(width)} ${match[2]}` : line;
  });
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
  return layoutServerDirectives(source)
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

const voidElements = new Set([
  "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr",
]);

function formatMarkupIndentation(source, indentation = "  ") {
  let depth = 0;
  let frontmatter = false;
  let pendingTag = null;
  let rawTag = "";

  return String(source).split(/\r?\n/).map((line) => {
    const trimmed = line.trim();
    if (!trimmed) return "";
    if (trimmed === "---") {
      frontmatter = !frontmatter;
      return trimmed;
    }
    if (frontmatter) return line;

    if (pendingTag) {
      if (/\/?>$/.test(trimmed)) {
        const standaloneEnding = /^\/?>$/.test(trimmed);
        const formatted = indentation.repeat(depth + (standaloneEnding ? 0 : 1)) + trimmed;
        if (!trimmed.endsWith("/>") && !voidElements.has(pendingTag.toLowerCase())) {
          depth++;
          if (pendingTag === "script" || pendingTag === "style") rawTag = pendingTag;
        }
        pendingTag = null;
        return formatted;
      }
      return indentation.repeat(depth + 1) + trimmed;
    }

    if (rawTag && !new RegExp(`^<\\/${rawTag}\\b`, "i").test(trimmed)) {
      // The embedded TypeScript/CSS formatter has already established the
      // relative indentation inside a raw block. Re-trimming here flattened
      // every function and object member to the script tag's depth on save.
      return line.replace(/[ \t]+$/, "");
    }

    const incompleteTag = /^<([A-Za-z][A-Za-z0-9:._-]*)\b[^>]*$/.exec(trimmed);
    if (incompleteTag) {
      pendingTag = incompleteTag[1].toLowerCase();
      return indentation.repeat(depth) + trimmed;
    }

    const startsWithClosingTag = /^<\/[A-Za-z][A-Za-z0-9:._-]*\s*>/.test(trimmed);
    const startsWithClosingBlock = /^\{\/(?:if|for)\}/.test(trimmed);
    const lineDepth = Math.max(0, depth - (startsWithClosingTag || startsWithClosingBlock ? 1 : 0));
    const formatted = indentation.repeat(lineDepth) + trimmed;

    let opens = 0;
    let closes = 0;
    for (const match of trimmed.matchAll(/<\/?([A-Za-z][A-Za-z0-9:._-]*)\b[^>]*>/g)) {
      const token = match[0];
      const name = match[1].toLowerCase();
      if (token.startsWith("</")) {
        closes++;
        if (name === rawTag) rawTag = "";
      } else if (!token.endsWith("/>") && !voidElements.has(name)) {
        opens++;
        if (name === "script" || name === "style") rawTag = name;
      }
    }
    if (/^\{(?:if|for)\b[^}]*\}$/.test(trimmed)) opens++;
    if (/^\{\/(?:if|for)\}$/.test(trimmed)) closes++;
    depth = Math.max(0, depth + opens - closes);
    return formatted;
  }).join("\n");
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
  dedentBlock,
  formatPropsBlock,
  formatNorthframeBlocks,
  formatMarkupIndentation,
  formattingPreservesTokens,
  formattingTokenSignature,
  formattingTemplate,
  htmlVirtualContent,
  insideHTMLTagAt,
  layoutDocumentMarkup,
  layoutMarkup,
  layoutServerDirectives,
  normalizeServerDirectives,
  restoreFormattedScripts,
  scriptRegionAt,
  scriptRegions,
  typescriptVirtualContent,
};
