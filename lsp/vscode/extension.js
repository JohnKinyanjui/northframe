"use strict";

const vscode = require("vscode");
const { LSPClient } = require("./lsp-client");
const { resolveServerCommand } = require("./server-command");
const {
  applyOffsetEdits,
  dedentBlock,
  formatPropsBlock,
  formatMarkupIndentation,
  formatNorthframeBlocks,
  formattingPreservesTokens,
  formattingTemplate,
  htmlVirtualContent,
  layoutDocumentMarkup,
  restoreFormattedScripts,
  scriptRegionAt,
  typescriptVirtualContent,
} = require("./embedded");

let activeController;

class EmbeddedDocuments {
  constructor() {
    this.contents = new Map();
    this.identities = new Map();
    this.changed = new vscode.EventEmitter();
    this.registration = vscode.workspace.registerTextDocumentContentProvider("northframe-embedded", {
      onDidChange: this.changed.event,
      provideTextDocumentContent: (uri) => this.contents.get(uri.toString()) || "",
    });
  }

  async open(source, language, content, identity = "document") {
    const extension = language === "typescript" ? "ts" : "html";
    const identityKey = `${source.uri.toString()}::${language}::${identity}`;
    const fingerprint = contentFingerprint(content);
    const uri = vscode.Uri.from({
      scheme: "northframe-embedded",
      authority: language,
      path: `${source.uri.path}.${identity}-${fingerprint}.${extension}`,
      query: encodeURIComponent(source.uri.toString()),
    });
    const key = uri.toString();
    const previous = this.identities.get(identityKey);
    if (previous && previous !== key) this.contents.delete(previous);
    this.identities.set(identityKey, key);
    if (this.contents.get(key) !== content) {
      this.contents.set(key, content);
      this.changed.fire(uri);
    }
    let document = await vscode.workspace.openTextDocument(uri);
    if (document.languageId !== language) document = await vscode.languages.setTextDocumentLanguage(document, language);
    return document;
  }

  dispose() {
    this.registration.dispose();
    this.changed.dispose();
    this.contents.clear();
    this.identities.clear();
  }
}

function contentFingerprint(content) {
  let hash = 2166136261;
  for (let index = 0; index < content.length; index++) {
    hash ^= content.charCodeAt(index);
    hash = Math.imul(hash, 16777619);
  }
  return (hash >>> 0).toString(36);
}

class NorthframeController {
  constructor(context) {
    this.context = context;
    this.client = null;
    this.output = vscode.window.createOutputChannel("Northframe");
    this.diagnostics = vscode.languages.createDiagnosticCollection("northframe");
    this.embedded = new EmbeddedDocuments();
    this.disposables = [this.output, this.diagnostics, this.embedded];
  }

  async activate() {
    const selector = { language: "northframe", scheme: "file" };
    this.disposables.push(
      vscode.workspace.onDidOpenTextDocument((document) => this.open(document)),
      vscode.workspace.onDidChangeTextDocument((event) => this.change(event.document)),
      vscode.workspace.onDidCloseTextDocument((document) => this.close(document)),
      vscode.workspace.onWillSaveTextDocument((event) => {
        if (event.document.languageId === "northframe") {
          event.waitUntil(this.organizeImports(event.document));
        }
      }),
      vscode.workspace.onDidChangeConfiguration((event) => {
        if (event.affectsConfiguration("northframe.server")) void this.restart();
      }),
      vscode.commands.registerCommand("northframe.restartLanguageServer", () => this.restart()),
      vscode.commands.registerCommand("northframe.showOutput", () => this.output.show(true)),
      vscode.languages.registerHoverProvider(selector, { provideHover: (document, position) => this.hover(document, position) }),
      vscode.languages.registerCompletionItemProvider(selector, { provideCompletionItems: (document, position, _token, completionContext) => this.completions(document, position, completionContext) }, "{", ".", "<", "\"", "'", "/"),
      vscode.languages.registerDefinitionProvider(selector, { provideDefinition: (document, position) => this.definition(document, position) }),
      vscode.languages.registerReferenceProvider(selector, { provideReferences: (document, position, context) => this.references(document, position, context) }),
      vscode.languages.registerRenameProvider(selector, {
        prepareRename: (document, position) => this.prepareRename(document, position),
        provideRenameEdits: (document, position, newName) => this.rename(document, position, newName),
      }),
      vscode.languages.registerDocumentSymbolProvider(selector, { provideDocumentSymbols: (document) => this.symbols(document) }),
      vscode.languages.registerDocumentFormattingEditProvider(selector, { provideDocumentFormattingEdits: (document, options) => this.format(document, options) }),
    );
    this.context.subscriptions.push(...this.disposables);
    await this.start();
  }

  async start() {
    const configuration = vscode.workspace.getConfiguration("northframe.server");
    const command = configuration.get("path", "north");
    const args = configuration.get("args", ["lsp"]);
    const folders = vscode.workspace.workspaceFolders || [];
    const server = resolveServerCommand({
      command,
      args,
      folders,
      pathSetting: configuration.inspect("path"),
      argsSetting: configuration.inspect("args"),
    });
    if (server.localCheckout) {
      this.log(`Using Northframe language server from the open source checkout at ${server.cwd}`);
    }
    const client = new LSPClient(
      server.command,
      server.args,
      (method, params) => this.notification(method, params),
      (message) => this.log(message),
      { cwd: server.cwd },
    );
    this.client = client;
    try {
      const initialization = await client.start({
        processId: process.pid,
        clientInfo: { name: "Northframe VS Code" },
        rootUri: folders[0]?.uri.toString() || null,
        workspaceFolders: folders.map((folder) => ({ uri: folder.uri.toString(), name: folder.name })),
        capabilities: {},
      });
      this.log("Northframe language server ready");
	  if (initialization?.capabilities?.referencesProvider !== true) {
		this.log("The configured north CLI is older than this extension and does not provide the current navigation contract");
		void vscode.window.showWarningMessage(
		  "The configured north CLI is older than the Northframe extension. Run `go install github.com/JohnKinyanjui/northframe/cmd/cli@latest`, then restart the language server.",
		  "Show Output",
		).then((choice) => {
		  if (choice === "Show Output") this.output.show(true);
		});
	  }
      for (const document of vscode.workspace.textDocuments) this.open(document);
    } catch (error) {
      if (this.client === client) this.client = null;
      this.log(`Could not start language server: ${error.message}`);
      void vscode.window.showErrorMessage(
        `Northframe language server could not start: ${error.message}. Install the north CLI or configure northframe.server.path.`,
        "Show Output",
      ).then((choice) => {
        if (choice === "Show Output") this.output.show(true);
      });
    }
  }

  async restart() {
    this.log("Restarting Northframe language server");
    const previous = this.client;
    this.client = null;
    this.diagnostics.clear();
    if (previous) await previous.stop();
    await this.start();
  }

  open(document) {
    if (!this.client || document.languageId !== "northframe") return;
    this.client.notify("textDocument/didOpen", {
      textDocument: {
        uri: document.uri.toString(),
        languageId: "northframe",
        version: document.version,
        text: document.getText(),
      },
    });
  }

  change(document) {
    if (!this.client || document.languageId !== "northframe") return;
    this.client.notify("textDocument/didChange", {
      textDocument: { uri: document.uri.toString(), version: document.version },
      contentChanges: [{ text: document.getText() }],
    });
  }

  close(document) {
    if (!this.client || document.languageId !== "northframe") return;
    this.client.notify("textDocument/didClose", { textDocument: { uri: document.uri.toString() } });
    this.diagnostics.delete(document.uri);
  }

  notification(method, params) {
    if (method === "textDocument/publishDiagnostics") {
      const uri = vscode.Uri.parse(params.uri);
      const document = vscode.workspace.textDocuments.find((current) => current.uri.toString() === uri.toString());
      if (document && Number.isInteger(params.version) && document.version !== params.version) {
        this.log(`Ignored stale diagnostics for ${uri.fsPath || uri.toString()} (version ${params.version}, current ${document.version})`);
        return;
      }
      const diagnostics = (params.diagnostics || []).map((item) => {
        const diagnostic = new vscode.Diagnostic(toRange(item.range), item.message, toDiagnosticSeverity(item.severity));
        diagnostic.source = item.source || "northframe";
        return diagnostic;
      });
      this.diagnostics.set(uri, diagnostics);
    }
    if (method === "northframe/serverExited") {
      this.client = null;
      void vscode.window.showWarningMessage(params.message, "Restart").then((choice) => {
        if (choice === "Restart") void this.restart();
      });
    }
  }

  async hover(document, position) {
    const [northframe, embedded] = await Promise.all([
      this.request("textDocument/hover", document, position),
      this.embeddedHovers(document, position),
    ]);
    const contents = [];
    if (northframe?.contents) {
      const value = typeof northframe.contents === "string" ? northframe.contents : northframe.contents.value || "";
      if (value) contents.push(new vscode.MarkdownString(value));
    }
    for (const hover of embedded || []) contents.push(...hover.contents);
    if (contents.length === 0) return undefined;
    const range = northframe?.range ? toRange(northframe.range) : embedded?.find((hover) => hover.range)?.range;
    return new vscode.Hover(contents, range);
  }

  async completions(document, position, completionContext) {
    const [result, embedded] = await Promise.all([
      this.request("textDocument/completion", document, position),
      this.embeddedCompletions(document, position, completionContext),
    ]);
    const rawItems = Array.isArray(result) ? result : result?.items || [];
    const northframeItems = rawItems.map((raw) => {
      const item = new vscode.CompletionItem(raw.label, toCompletionKind(raw.kind));
      item.detail = raw.detail;
      if (raw.documentation) item.documentation = typeof raw.documentation === "string" ? raw.documentation : raw.documentation.value;
      if (raw.textEdit) {
        item.range = toRange(raw.textEdit.range);
        item.insertText = raw.insertTextFormat === 2 ? new vscode.SnippetString(raw.textEdit.newText) : raw.textEdit.newText;
      } else if (raw.insertText) {
        item.insertText = raw.insertTextFormat === 2 ? new vscode.SnippetString(raw.insertText) : raw.insertText;
      }
      item.sortText = raw.sortText;
      item.filterText = raw.filterText;
      if (raw.additionalTextEdits) {
        item.additionalTextEdits = raw.additionalTextEdits.map((edit) =>
          vscode.TextEdit.replace(toRange(edit.range), edit.newText));
      }
      return item;
    });
    const embeddedItems = Array.isArray(embedded) ? embedded : embedded?.items || [];
    return new vscode.CompletionList(
      [...embeddedItems, ...northframeItems],
      Boolean(result?.isIncomplete || embedded?.isIncomplete),
    );
  }

  async embeddedCompletions(document, position, completionContext) {
    try {
      const virtual = await this.embeddedDocument(document, position);
      return await vscode.commands.executeCommand(
        "vscode.executeCompletionItemProvider",
        virtual.uri,
        position,
        completionContext?.triggerCharacter,
      );
    } catch (error) {
      this.log(`Embedded completion failed: ${error.message}`);
      return [];
    }
  }

  async embeddedHovers(document, position) {
    try {
      const virtual = await this.embeddedDocument(document, position);
      return await vscode.commands.executeCommand("vscode.executeHoverProvider", virtual.uri, position);
    } catch (error) {
      this.log(`Embedded hover failed: ${error.message}`);
      return [];
    }
  }

  embeddedDocument(document, position) {
    const source = document.getText();
    const offset = document.offsetAt(position);
    const region = scriptRegionAt(source, offset);
    if (region?.kind === "typescript") {
      return this.embedded.open(document, "typescript", typescriptVirtualContent(source), "features");
    }
    return this.embedded.open(document, "html", htmlVirtualContent(source), "features");
  }

  async definition(document, position) {
    const [result, embedded] = await Promise.all([
      this.request("textDocument/definition", document, position),
      this.embeddedLocations("vscode.executeDefinitionProvider", document, position),
    ]);
    const raw = result ? (Array.isArray(result) ? result : [result]) : [];
    const locations = raw.map((location) => new vscode.Location(vscode.Uri.parse(location.uri), toRange(location.range)));
    return uniqueLocations([...locations, ...(embedded || [])]);
  }

  async references(document, position, context) {
    const [result, embedded] = await Promise.all([
      this.request("textDocument/references", document, position, { context: { includeDeclaration: context?.includeDeclaration !== false } }),
      this.embeddedLocations("vscode.executeReferenceProvider", document, position),
    ]);
    const raw = Array.isArray(result) ? result : [];
    const locations = raw.map((location) => new vscode.Location(vscode.Uri.parse(location.uri), toRange(location.range)));
    return uniqueLocations([...locations, ...(embedded || [])]);
  }

  async embeddedLocations(command, document, position) {
    try {
      const virtual = await this.embeddedDocument(document, position);
      const locations = await vscode.commands.executeCommand(command, virtual.uri, position);
      return (locations || []).map((location) => {
        const target = location.targetUri
          ? new vscode.Location(location.targetUri, location.targetSelectionRange || location.targetRange)
          : location;
        if (target.uri?.scheme !== "northframe-embedded") return target;
        return new vscode.Location(document.uri, target.range);
      });
    } catch (error) {
      this.log(`Embedded navigation failed: ${error.message}`);
      return [];
    }
  }

  async prepareRename(document, position) {
    const result = await this.request("textDocument/prepareRename", document, position);
    if (!result?.range) throw new Error("This Northframe symbol cannot be renamed here.");
    return { range: toRange(result.range), placeholder: result.placeholder };
  }

  async rename(document, position, newName) {
    const result = await this.request("textDocument/rename", document, position, { newName });
    if (!result?.changes) return undefined;
    const edit = new vscode.WorkspaceEdit();
    for (const [rawURI, changes] of Object.entries(result.changes)) {
      const uri = vscode.Uri.parse(rawURI);
      for (const change of changes || []) edit.replace(uri, toRange(change.range), change.newText);
    }
    return edit;
  }

  async symbols(document) {
    const result = await this.request("textDocument/documentSymbol", document);
    return (result || []).map((raw) => new vscode.DocumentSymbol(
      raw.name,
      raw.detail || "",
      toSymbolKind(raw.kind),
      toRange(raw.range),
      toRange(raw.selectionRange || raw.range),
    ));
  }

  async format(document, options) {
    const original = document.getText();
    const importEdits = await this.organizeImports(document);
    const organized = applyTextEdits(document, importEdits);
    const indentation = indentationUnit(options);
    const template = formattingTemplate(organized, indentation);
    const htmlDocument = await this.embedded.open(document, "html", template.source, "format");
    const htmlEdits = await vscode.commands.executeCommand("vscode.executeFormatDocumentProvider", htmlDocument.uri, options);
    const formattedHTML = applyTextEdits(htmlDocument, htmlEdits || []);

    const formattedContents = [];
    for (let index = 0; index < template.regions.length; index++) {
      const region = template.regions[index];
      if (region.kind === "typescript") {
        const scriptSource = dedentBlock(region.content);
        const scriptDocument = await this.embedded.open(document, "typescript", scriptSource, `script-${index}`);
        const edits = await vscode.commands.executeCommand("vscode.executeFormatDocumentProvider", scriptDocument.uri, options);
        const candidate = applyTextEdits(scriptDocument, edits || []);
        if (formattingPreservesTokens(scriptSource, candidate)) {
          formattedContents.push(candidate.trim());
        } else {
          this.log(`Ignored unsafe TypeScript formatting edits in script ${index + 1}`);
          formattedContents.push(scriptSource);
        }
      } else if (region.kind === "props") {
        formattedContents.push(formatPropsBlock(region.content, region.frontmatter ? indentation : ""));
      } else {
        formattedContents.push(dedentBlock(region.content));
      }
    }

    const formatted = formatMarkupIndentation(
      layoutDocumentMarkup(
        formatNorthframeBlocks(
          restoreFormattedScripts({ ...template, source: formattedHTML }, formattedContents),
          indentation,
        ),
      ),
      indentation,
    );
    if (!formattingPreservesTokens(organized, formatted)) {
      this.log("Ignored unsafe document formatting edits because they changed Northframe tokens");
      return [];
    }
    if (formatted === original) return [];
    return [vscode.TextEdit.replace(fullRange(document), formatted)];
  }

  async organizeImports(document) {
    if (!this.client || document.languageId !== "northframe") return [];
    const result = await this.request("northframe/organizeImports", document);
    return (result || []).map((edit) => vscode.TextEdit.replace(toRange(edit.range), edit.newText));
  }

  request(method, document, position, additional = {}) {
    if (!this.client) return Promise.resolve(undefined);
    const params = { textDocument: { uri: document.uri.toString() }, ...additional };
    if (position) params.position = { line: position.line, character: position.character };
    return this.client.request(method, params).catch((error) => {
      this.log(`${method} failed: ${error.message}`);
      return undefined;
    });
  }

  log(message) {
    if (message) this.output.appendLine(`[${new Date().toISOString()}] ${message}`);
  }

  async dispose() {
    const client = this.client;
    this.client = null;
    if (client) await client.stop();
  }
}

function indentationUnit(options = {}) {
  if (options.insertSpaces === false) return "\t";
  return " ".repeat(Math.max(1, options.tabSize || 2));
}

function toRange(range) {
  return new vscode.Range(range.start.line, range.start.character, range.end.line, range.end.character);
}

function fullRange(document) {
  const lastLine = document.lineAt(document.lineCount - 1);
  return new vscode.Range(0, 0, lastLine.lineNumber, lastLine.text.length);
}

function applyTextEdits(document, edits) {
	return applyOffsetEdits(document.getText(), edits.map((edit) => ({
		start: document.offsetAt(edit.range.start),
		end: document.offsetAt(edit.range.end),
		newText: edit.newText,
	})));
}

function uniqueLocations(locations) {
  const seen = new Set();
  return locations.filter((location) => {
    if (!location?.uri || !location?.range) return false;
    const key = `${location.uri.toString()}:${location.range.start.line}:${location.range.start.character}:${location.range.end.line}:${location.range.end.character}`;
    if (seen.has(key)) return false;
    seen.add(key);
    return true;
  });
}

function toDiagnosticSeverity(severity) {
  if (severity === 1) return vscode.DiagnosticSeverity.Error;
  if (severity === 2) return vscode.DiagnosticSeverity.Warning;
  if (severity === 3) return vscode.DiagnosticSeverity.Information;
  return vscode.DiagnosticSeverity.Hint;
}

function toCompletionKind(kind) {
  if (!kind) return vscode.CompletionItemKind.Text;
  return Math.max(vscode.CompletionItemKind.Text, Math.min(vscode.CompletionItemKind.TypeParameter, kind - 1));
}

function toSymbolKind(kind) {
  if (!kind) return vscode.SymbolKind.Object;
  return Math.max(vscode.SymbolKind.File, Math.min(vscode.SymbolKind.TypeParameter, kind - 1));
}

async function activate(context) {
  activeController = new NorthframeController(context);
  await activeController.activate();
}

async function deactivate() {
  if (activeController) await activeController.dispose();
}

module.exports = { activate, deactivate };
