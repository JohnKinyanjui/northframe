package lsp

import "encoding/json"

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  any             `json:"result,omitempty"`
	Error   *responseError  `json:"error,omitempty"`
}

type responseError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type notification struct {
	JSONRPC string `json:"jsonrpc"`
	Method  string `json:"method"`
	Params  any    `json:"params"`
}

type position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

type protocolRange struct {
	Start position `json:"start"`
	End   position `json:"end"`
}

type diagnostic struct {
	Range    protocolRange `json:"range"`
	Severity int           `json:"severity"`
	Source   string        `json:"source"`
	Message  string        `json:"message"`
}

type textDocumentItem struct {
	URI     string `json:"uri"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

type versionedTextDocument struct {
	URI     string `json:"uri"`
	Version int    `json:"version"`
}

type textDocumentPosition struct {
	TextDocument versionedTextDocument `json:"textDocument"`
	Position     position              `json:"position"`
}
