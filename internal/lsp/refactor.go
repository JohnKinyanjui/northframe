package lsp

import (
	"encoding/json"
	"regexp"
	"strings"
)

var renameIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

type renameTarget struct {
	Name        string
	Range       protocolRange
	Occurrences []protocolRange
}

func (current *server) prepareRename(id json.RawMessage, raw json.RawMessage) error {
	var params textDocumentPosition
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	text := current.document(params.TextDocument.URI)
	target, ok := refactorTarget(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))
	if !ok {
		return current.reply(id, nil)
	}
	return current.reply(id, map[string]any{"range": target.Range, "placeholder": target.Name})
}

func (current *server) rename(id json.RawMessage, raw json.RawMessage) error {
	var params struct {
		TextDocument versionedTextDocument `json:"textDocument"`
		Position     position              `json:"position"`
		NewName      string                `json:"newName"`
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return err
	}
	if !renameIdentifier.MatchString(params.NewName) {
		return current.reply(id, nil)
	}
	text := current.document(params.TextDocument.URI)
	target, ok := refactorTarget(params.TextDocument.URI, text, positionToByteOffset(text, params.Position))
	if !ok {
		return current.reply(id, nil)
	}
	edits := make([]map[string]any, 0, len(target.Occurrences))
	for _, occurrence := range target.Occurrences {
		edits = append(edits, map[string]any{"range": occurrence, "newText": params.NewName})
	}
	return current.reply(id, map[string]any{"changes": map[string]any{params.TextDocument.URI: edits}})
}

func refactorTarget(uri, text string, offset int) (renameTarget, bool) {
	word := wordAt(text, offset)
	name := tokenRoot(strings.TrimPrefix(word, "Props."))
	if field, ok := findDocumentProp(uri, text, name); ok && field.URI == uri {
		occurrences := []protocolRange{field.Range}
		pattern := regexp.MustCompile(`\bProps\.` + regexp.QuoteMeta(field.Name) + `\b`)
		for _, match := range pattern.FindAllStringIndex(text, -1) {
			start := match[1] - len(field.Name)
			occurrences = appendUniqueRange(occurrences, protocolRange{Start: byteOffsetToPosition(text, start), End: byteOffsetToPosition(text, match[1])})
		}
		return renameTarget{Name: field.Name, Range: occurrenceRangeAt(occurrences, text, offset), Occurrences: occurrences}, true
	}
	root := tokenRoot(word)
	binding, ok := activeTemplateBindings(uri, text, offset)[root]
	if !ok {
		return renameTarget{}, false
	}
	value := binding.Value
	if value.Owner != "range variable" || value.URI != uri {
		return renameTarget{}, false
	}
	start, end, ok := loopScope(text, value.Range)
	if !ok {
		return renameTarget{}, false
	}
	occurrences := []protocolRange{}
	pattern := regexp.MustCompile(`\b` + regexp.QuoteMeta(value.Name) + `\b`)
	for _, match := range pattern.FindAllStringIndex(text[start:end], -1) {
		absoluteStart, absoluteEnd := start+match[0], start+match[1]
		occurrences = append(occurrences, protocolRange{Start: byteOffsetToPosition(text, absoluteStart), End: byteOffsetToPosition(text, absoluteEnd)})
	}
	return renameTarget{Name: value.Name, Range: occurrenceRangeAt(occurrences, text, offset), Occurrences: occurrences}, len(occurrences) > 0
}

func loopScope(text string, declaration protocolRange) (int, int, bool) {
	declarationOffset := positionToByteOffset(text, declaration.Start)
	open := strings.LastIndex(text[:declarationOffset], "{for")
	if open < 0 {
		return 0, 0, false
	}
	depth := 0
	for _, match := range serverLoopToken.FindAllStringSubmatchIndex(text[open:], -1) {
		if match[2] >= 0 {
			depth++
		} else {
			depth--
		}
		if depth == 0 {
			return open, open + match[1], true
		}
	}
	return 0, 0, false
}

func occurrenceRangeAt(items []protocolRange, text string, offset int) protocolRange {
	for _, item := range items {
		start, end := positionToByteOffset(text, item.Start), positionToByteOffset(text, item.End)
		if offset >= start && offset <= end {
			return item
		}
	}
	if len(items) > 0 {
		return items[0]
	}
	return protocolRange{}
}

func appendUniqueRange(items []protocolRange, candidate protocolRange) []protocolRange {
	for _, item := range items {
		if item == candidate {
			return items
		}
	}
	return append(items, candidate)
}
