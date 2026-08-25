package lsp

import (
	"net/url"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

func documentPath(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "file" {
		return ""
	}
	return filepath.FromSlash(parsed.Path)
}

func documentURI(path string) string {
	return (&url.URL{Scheme: "file", Path: filepath.ToSlash(path)}).String()
}

func byteOffsetToPosition(text string, offset int) position {
	if offset < 0 {
		offset = 0
	}
	if offset > len(text) {
		offset = len(text)
	}
	result := position{}
	for index := 0; index < offset; {
		r, size := utf8.DecodeRuneInString(text[index:])
		if r == '\n' {
			result.Line++
			result.Character = 0
		} else {
			result.Character += utf16.RuneLen(r)
		}
		index += size
	}
	return result
}

func positionToByteOffset(text string, target position) int {
	line := 0
	character := 0
	for index := 0; index < len(text); {
		if line == target.Line && character >= target.Character {
			return index
		}
		r, size := utf8.DecodeRuneInString(text[index:])
		if r == '\n' {
			if line == target.Line {
				return index
			}
			line++
			character = 0
		} else if line == target.Line {
			character += utf16.RuneLen(r)
		}
		index += size
	}
	return len(text)
}

func wordAt(text string, offset int) string {
	if offset > len(text) {
		offset = len(text)
	}
	start := offset
	for start > 0 && isWordByte(text[start-1]) {
		start--
	}
	end := offset
	for end < len(text) && isWordByte(text[end]) {
		end++
	}
	return strings.TrimSpace(text[start:end])
}

func wordRange(text string, offset int) protocolRange {
	if offset > len(text) {
		offset = len(text)
	}
	start := offset
	for start > 0 && isWordByte(text[start-1]) {
		start--
	}
	end := offset
	for end < len(text) && isWordByte(text[end]) {
		end++
	}
	return protocolRange{Start: byteOffsetToPosition(text, start), End: byteOffsetToPosition(text, end)}
}

func isWordByte(value byte) bool {
	return value == '_' || value == '.' || value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z' || value >= '0' && value <= '9'
}

func fullDocumentRange(text string) protocolRange {
	return protocolRange{Start: position{}, End: byteOffsetToPosition(text, len(text))}
}
