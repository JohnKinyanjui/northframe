package lsp

import (
	"regexp"
	"strings"
)

var serverLoopToken = regexp.MustCompile(`\{for\s+([a-z_][A-Za-z0-9_]*)\s*:=\s*range\s+([^}]+)\}|\{/for\}`)

type templateValue struct {
	Name          string
	Type          string
	Owner         string
	Documentation string
	URI           string
	Range         protocolRange
	Package       *propsImport
	Prop          bool
}

type templateBinding struct {
	Value templateValue
}

func resolveTemplateValue(uri, source, expression string, offset int) (templateValue, bool) {
	expression = strings.TrimSpace(expression)
	if expression == "" {
		return templateValue{}, false
	}
	parts := strings.Split(expression, ".")
	var current templateValue
	var ok bool
	if parts[0] == "Props" {
		if len(parts) < 2 {
			return templateValue{}, false
		}
		field, exists := findDocumentProp(uri, source, parts[1])
		if !exists {
			return templateValue{}, false
		}
		current = templateValue{Name: field.Name, Type: field.Type, Owner: "Props", URI: field.URI, Range: field.Range, Prop: true}
		parts = parts[2:]
		ok = true
	} else {
		binding, exists := activeTemplateBindings(uri, source, offset)[parts[0]]
		if !exists {
			return templateValue{}, false
		}
		current = binding.Value
		parts = parts[1:]
		ok = true
	}
	for _, fieldName := range parts {
		current, ok = resolveTemplateField(uri, source, current, fieldName)
		if !ok {
			return templateValue{}, false
		}
	}
	return current, ok
}

func activeTemplateBindings(uri, source string, offset int) map[string]templateBinding {
	if offset < 0 || offset > len(source) {
		offset = len(source)
	}
	bindings := map[string]templateBinding{}
	type scope struct {
		name     string
		previous templateBinding
		had      bool
	}
	var stack []scope
	for _, match := range serverLoopToken.FindAllStringSubmatchIndex(source[:offset], -1) {
		if match[2] < 0 {
			if len(stack) == 0 {
				continue
			}
			last := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if last.had {
				bindings[last.name] = last.previous
			} else {
				delete(bindings, last.name)
			}
			continue
		}
		name := source[match[2]:match[3]]
		collection := source[match[4]:match[5]]
		value, ok := resolveExpressionWithBindings(uri, source, strings.TrimSpace(collection), bindings)
		if !ok {
			continue
		}
		value = rangeElement(value)
		value.Name = name
		value.Owner = "range variable"
		value.URI = uri
		value.Range = protocolRange{Start: byteOffsetToPosition(source, match[2]), End: byteOffsetToPosition(source, match[3])}
		previous, had := bindings[name]
		stack = append(stack, scope{name: name, previous: previous, had: had})
		bindings[name] = templateBinding{Value: value}
	}
	return bindings
}

func resolveExpressionWithBindings(uri, source, expression string, bindings map[string]templateBinding) (templateValue, bool) {
	parts := strings.Split(strings.TrimSpace(expression), ".")
	if len(parts) == 0 {
		return templateValue{}, false
	}
	var current templateValue
	if parts[0] == "Props" {
		if len(parts) < 2 {
			return templateValue{}, false
		}
		field, ok := findDocumentProp(uri, source, parts[1])
		if !ok {
			return templateValue{}, false
		}
		current = templateValue{Name: field.Name, Type: field.Type, Owner: "Props", URI: field.URI, Range: field.Range, Prop: true}
		parts = parts[2:]
	} else {
		binding, ok := bindings[parts[0]]
		if !ok {
			return templateValue{}, false
		}
		current = binding.Value
		parts = parts[1:]
	}
	for _, field := range parts {
		var ok bool
		current, ok = resolveTemplateField(uri, source, current, field)
		if !ok {
			return templateValue{}, false
		}
	}
	return current, true
}

func resolveTemplateField(uri, source string, parent templateValue, name string) (templateValue, bool) {
	symbol, imported, ok := resolveTemplateType(uri, source, parent.Type, parent.Package)
	if !ok {
		return templateValue{}, false
	}
	for _, field := range symbol.Fields {
		if field.Name != name {
			continue
		}
		return templateValue{
			Name: field.Name, Type: field.Type, Owner: imported.Alias + "." + symbol.Name,
			Documentation: field.Documentation, URI: field.URI, Range: field.Range, Package: &imported,
		}, true
	}
	return templateValue{}, false
}

func resolveTemplateType(uri, source, raw string, fallback *propsImport) (goTypeSymbol, propsImport, bool) {
	typeName := baseTemplateType(raw)
	imported := propsImport{}
	name := typeName
	if alias, remainder, found := strings.Cut(typeName, "."); found {
		name = remainder
		for _, candidate := range propsImports(source) {
			if candidate.Alias == alias {
				imported = candidate
				break
			}
		}
	} else if fallback != nil {
		imported = *fallback
	}
	if imported.Path == "" {
		return goTypeSymbol{}, propsImport{}, false
	}
	for _, symbol := range importedGoTypes(uri, imported) {
		if symbol.Name == name {
			return symbol, imported, true
		}
	}
	return goTypeSymbol{}, propsImport{}, false
}

func baseTemplateType(raw string) string {
	value := strings.TrimSpace(raw)
	for strings.HasPrefix(value, "*") {
		value = strings.TrimSpace(strings.TrimPrefix(value, "*"))
	}
	return value
}

func rangeElement(value templateValue) templateValue {
	raw := strings.TrimSpace(value.Type)
	for strings.HasPrefix(raw, "*") {
		raw = strings.TrimSpace(strings.TrimPrefix(raw, "*"))
	}
	if strings.HasPrefix(raw, "map[") {
		if close := strings.Index(raw, "]"); close >= 0 {
			raw = strings.TrimSpace(raw[close+1:])
		}
	} else if strings.HasPrefix(raw, "[") {
		if close := strings.Index(raw, "]"); close >= 0 {
			raw = strings.TrimSpace(raw[close+1:])
		}
	}
	value.Type = raw
	value.Prop = false
	return value
}

func templateValueAt(uri, source string, offset int) (templateValue, bool) {
	return resolveTemplateValue(uri, source, wordAt(source, offset), offset)
}

func templateFieldsFor(uri, source, expression string, offset int) []goFieldSymbol {
	value, ok := resolveTemplateValue(uri, source, expression, offset)
	if !ok {
		return nil
	}
	symbol, _, ok := resolveTemplateType(uri, source, value.Type, value.Package)
	if !ok {
		return nil
	}
	return symbol.Fields
}
