"use strict";

const assert = require("node:assert/strict");
const test = require("node:test");
const { pathToFileURL } = require("node:url");
const path = require("node:path");
const { LSPClient } = require("../lsp-client");

test("client exchanges framed messages with northframe lsp", async () => {
  const notifications = [];
  const client = new LSPClient(
    "go",
    ["run", "../../cmd/northframe", "lsp"],
    (method, params) => notifications.push({ method, params }),
    () => {},
  );
  try {
    const initialized = await client.start({ processId: process.pid, capabilities: {}, rootUri: null });
    assert.equal(initialized.serverInfo.name, "Northframe Language Server");

    const uri = pathToFileURL(path.join(process.cwd(), "test-page.north")).toString();
    client.notify("textDocument/didOpen", {
      textDocument: { uri, languageId: "northframe", version: 1, text: "<main></main>" },
    });
    const completion = await client.request("textDocument/completion", {
      textDocument: { uri },
      position: { line: 0, character: 6 },
    });
    assert.ok(completion.items.some((item) => item.label === "nf-enhance"));
    const diagnostics = notifications.find((item) => item.method === "textDocument/publishDiagnostics");
    assert.ok(diagnostics);
    assert.equal(diagnostics.params.version, 1);
  } finally {
    await client.stop();
  }
});
