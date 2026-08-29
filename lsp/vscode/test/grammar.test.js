"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const path = require("node:path");
const test = require("node:test");

const grammar = JSON.parse(fs.readFileSync(path.join(__dirname, "..", "syntaxes", "northframe.tmLanguage.json"), "utf8"));

test("props use dedicated Northframe scopes instead of invalid Go embedding", () => {
  const props = grammar.repository["props-script"];
  assert.equal(props.contentName, "meta.embedded.block.props.northframe");
  assert.equal(props.patterns.some((pattern) => pattern.include === "source.go"), false);
  assert.equal(props.patterns.some((pattern) => pattern.match?.includes("keyword") || pattern.captures?.[1]?.name === "keyword.control.import.go"), true);
  assert.equal(props.beginCaptures[4].name, "entity.other.attribute-name.html");
  assert.equal(props.beginCaptures[7].name, "string.quoted.double.html");
});

test("Astro-style Props frontmatter has dedicated Northframe scopes", () => {
  const props = grammar.repository["props-frontmatter"];
  assert.equal(props.contentName, "meta.embedded.block.props.northframe");
  assert.equal(props.patterns[0].captures[2].name, "entity.name.type.northframe");
  assert.equal(props.patterns.some((pattern) => pattern.include === "source.go"), false);
  assert.equal(grammar.patterns[0].include, "#props-frontmatter");
});

test("Go server directives embed Go grammar", () => {
  const loop = grammar.repository["server-blocks"].patterns.find((pattern) => pattern.begin?.includes("(for)"));
  const condition = grammar.repository["server-blocks"].patterns.find((pattern) => pattern.begin?.includes("(if)"));
  const html = grammar.repository["server-blocks"].patterns.find((pattern) => pattern.begin?.includes("(html)"));
  assert.equal(loop.contentName, "source.go");
  assert.equal(loop.patterns[0].include, "source.go");
  assert.equal(condition.contentName, "source.go");
  assert.equal(html.contentName, "source.go");
  assert.equal(html.patterns[0].include, "source.go");

  const component = grammar.repository.components.patterns[0];
  assert.equal(component.match.startsWith("(</?)"), true);
  assert.equal(component.captures[2].name, "support.class.component.northframe");
});

test("server and client interpolation use distinct runtime colors", () => {
  const expressions = grammar.repository.expressions.patterns;
  const server = expressions.find((pattern) => pattern.begin === "(\\$)(\\{)");
  const client = expressions.find((pattern) => pattern.match === "(#)(\\{)([A-Za-z_][A-Za-z0-9_.]*)(\\})");
  assert.equal(server.beginCaptures[1].name, "keyword.operator.server.northframe");
  assert.equal(server.contentName, "source.go");
  assert.equal(client.captures[1].name, "keyword.operator.client.northframe");
  assert.equal(client.captures[3].name, "variable.other.readwrite.ts.northframe");
});
