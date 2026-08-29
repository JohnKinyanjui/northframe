// Package lsp implements the .north language server over standard JSON-RPC.
package lsp

import (
	"encoding/json"
	"errors"
	"io"
	"strings"
)

var errExit = errors.New("language server exit")

type server struct {
	transport *transport
	documents map[string]string
	shutdown  bool
}

// Run serves Language Server Protocol messages until the client exits or the
// input stream closes. Diagnostics use the production compiler parser.
func Run(reader io.Reader, writer io.Writer) error {
	current := &server{transport: newTransport(reader, writer), documents: make(map[string]string)}
	for {
		message, err := current.transport.read()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return err
		}
		if err := current.handle(message); errors.Is(err, errExit) {
			return nil
		} else if err != nil {
			return err
		}
	}
}

func (current *server) handle(message request) error {
	switch message.Method {
	case "initialize":
		return current.reply(message.ID, initializeResult())
	case "initialized":
		return nil
	case "shutdown":
		current.shutdown = true
		return current.reply(message.ID, json.RawMessage("null"))
	case "exit":
		return errExit
	case "textDocument/didOpen":
		return current.didOpen(message.Params)
	case "textDocument/didChange":
		return current.didChange(message.Params)
	case "textDocument/didClose":
		return current.didClose(message.Params)
	case "textDocument/hover":
		return current.hover(message.ID, message.Params)
	case "textDocument/completion":
		return current.completion(message.ID, message.Params)
	case "textDocument/definition":
		return current.definition(message.ID, message.Params)
	case "textDocument/references":
		return current.references(message.ID, message.Params)
	case "textDocument/prepareRename":
		return current.prepareRename(message.ID, message.Params)
	case "textDocument/rename":
		return current.rename(message.ID, message.Params)
	case "textDocument/documentSymbol":
		return current.documentSymbols(message.ID, message.Params)
	case "textDocument/formatting":
		return current.formatting(message.ID, message.Params)
	case "northframe/organizeImports":
		return current.organizeImports(message.ID, message.Params)
	default:
		if len(message.ID) == 0 {
			return nil
		}
		return current.transport.write(response{
			JSONRPC: "2.0", ID: message.ID,
			Error: &responseError{Code: -32601, Message: "method not found: " + message.Method},
		})
	}
}

func initializeResult() map[string]any {
	return map[string]any{
		"capabilities": map[string]any{
			"textDocumentSync":           1,
			"hoverProvider":              true,
			"definitionProvider":         true,
			"referencesProvider":         true,
			"renameProvider":             map[string]any{"prepareProvider": true},
			"documentSymbolProvider":     true,
			"documentFormattingProvider": true,
			"completionProvider": map[string]any{
				"triggerCharacters": []string{"{", ".", "<"},
			},
		},
		"serverInfo": map[string]string{"name": "Northframe Language Server"},
	}
}

func (current *server) reply(id json.RawMessage, result any) error {
	return current.transport.write(response{JSONRPC: "2.0", ID: id, Result: result})
}

func (current *server) didOpen(raw json.RawMessage) error {
	var params struct {
		TextDocument textDocumentItem `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	current.documents[params.TextDocument.URI] = params.TextDocument.Text
	return current.publishDiagnostics(params.TextDocument.URI, params.TextDocument.Text, params.TextDocument.Version)
}

func (current *server) didChange(raw json.RawMessage) error {
	var params struct {
		TextDocument   versionedTextDocument `json:"textDocument"`
		ContentChanges []struct {
			Text string `json:"text"`
		} `json:"contentChanges"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	if len(params.ContentChanges) == 0 {
		return nil
	}
	text := params.ContentChanges[len(params.ContentChanges)-1].Text
	current.documents[params.TextDocument.URI] = text
	return current.publishDiagnostics(params.TextDocument.URI, text, params.TextDocument.Version)
}

func (current *server) didClose(raw json.RawMessage) error {
	var params struct {
		TextDocument versionedTextDocument `json:"textDocument"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	delete(current.documents, params.TextDocument.URI)
	return current.transport.write(notification{
		JSONRPC: "2.0", Method: "textDocument/publishDiagnostics",
		Params: map[string]any{"uri": params.TextDocument.URI, "diagnostics": []diagnostic{}},
	})
}

func (current *server) document(uri string) string {
	return current.documents[uri]
}

func tokenRoot(value string) string {
	root, _, _ := strings.Cut(value, ".")
	return root
}
