"use strict";

const assert = require("node:assert/strict");
const fs = require("node:fs");
const os = require("node:os");
const path = require("node:path");
const test = require("node:test");
const { resolveServerCommand } = require("../server-command");

test("uses the open Northframe checkout instead of a stale default CLI", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "northframe-vscode-"));
  try {
    fs.mkdirSync(path.join(root, "cmd", "cli"), { recursive: true });
    fs.writeFileSync(path.join(root, "go.mod"), "module github.com/JohnKinyanjui/northframe\n\ngo 1.27\n");
    fs.writeFileSync(path.join(root, "cmd", "cli", "main.go"), "package main\n");
    assert.deepEqual(resolveServerCommand({
      command: "northframe",
      args: ["lsp"],
      folders: [{ uri: { fsPath: root } }],
    }), {
      command: "go",
      args: ["run", "./cmd/northframe", "lsp"],
      cwd: root,
      localCheckout: true,
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("finds the Northframe checkout above an example workspace", () => {
  const root = fs.mkdtempSync(path.join(os.tmpdir(), "northframe-vscode-parent-"));
  try {
    fs.mkdirSync(path.join(root, "cmd", "cli"), { recursive: true });
    fs.mkdirSync(path.join(root, "examples", "demo"), { recursive: true });
    fs.writeFileSync(path.join(root, "go.mod"), "module github.com/JohnKinyanjui/northframe\n\ngo 1.27\n");
    fs.writeFileSync(path.join(root, "cmd", "cli", "main.go"), "package main\n");
    assert.deepEqual(resolveServerCommand({
      command: "northframe",
      args: ["lsp"],
      folders: [{ uri: { fsPath: path.join(root, "examples", "demo") } }],
    }), {
      command: "go",
      args: ["run", "./cmd/northframe", "lsp"],
      cwd: root,
      localCheckout: true,
    });
  } finally {
    fs.rmSync(root, { recursive: true, force: true });
  }
});

test("preserves an explicitly configured language server", () => {
  assert.deepEqual(resolveServerCommand({
    command: "/opt/north/bin/north",
    args: ["lsp"],
    folders: [],
    pathSetting: { workspaceValue: "/opt/north/bin/north" },
  }), {
    command: "/opt/north/bin/north",
    args: ["lsp"],
    cwd: undefined,
    localCheckout: false,
  });
});
