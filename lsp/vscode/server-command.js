"use strict";

const fs = require("node:fs");
const path = require("node:path");

const NORTHFRAME_MODULE = "github.com/JohnKinyanjui/northframe";

function hasExplicitValue(inspected) {
  if (!inspected) return false;
  return [
    inspected.globalValue,
    inspected.workspaceValue,
    inspected.workspaceFolderValue,
    inspected.globalLanguageValue,
    inspected.workspaceLanguageValue,
    inspected.workspaceFolderLanguageValue,
  ].some((value) => value !== undefined);
}

function northframeCheckout(folders) {
  for (const folder of folders || []) {
    let root = folder?.uri?.fsPath || folder?.fsPath || folder;
    if (typeof root !== "string" || root === "") continue;
    while (true) {
      const moduleFile = path.join(root, "go.mod");
      const entrypoint = path.join(root, "cmd", "cli", "main.go");
      if (fs.existsSync(moduleFile) && fs.existsSync(entrypoint)) {
        try {
          const moduleSource = fs.readFileSync(moduleFile, "utf8");
          if (new RegExp(`^module\\s+${escapeRegExp(NORTHFRAME_MODULE)}\\s*$`, "m").test(moduleSource)) {
            return root;
          }
        } catch (_) {
          // Keep walking: an unreadable nested module must not hide a parent checkout.
        }
      }
      const parent = path.dirname(root);
      if (parent === root) break;
      root = parent;
    }
  }
  return undefined;
}

function resolveServerCommand({ command, args, folders, pathSetting, argsSetting }) {
  const explicit = hasExplicitValue(pathSetting) || hasExplicitValue(argsSetting);
  if (command === "north" && !explicit) {
    const checkout = northframeCheckout(folders);
    if (checkout) {
      return {
        command: "go",
        args: ["run", "./cmd/cli", "lsp"],
        cwd: checkout,
        localCheckout: true,
      };
    }
  }
  return { command, args, cwd: undefined, localCheckout: false };
}

function escapeRegExp(value) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

module.exports = { hasExplicitValue, northframeCheckout, resolveServerCommand };
