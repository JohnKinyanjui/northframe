package compiler

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"
)

type parsedComponentTag struct {
	Name        string
	Attributes  []componentAttribute
	Closing     bool
	SelfClosing bool
	End         int
}

func nextComponentTag(source string, position int) int {
	for position < len(source) {
		offset := strings.IndexByte(source[position:], '<')
		if offset < 0 {
			return -1
		}
		candidate := position + offset
		nameStart := candidate + 1
		if nameStart < len(source) && source[nameStart] == '/' {
			nameStart++
		}
		if nameStart < len(source) && unicode.IsUpper(rune(source[nameStart])) {
			return candidate
		}
		position = candidate + 1
	}
	return -1
}

func parseComponentTag(source string, start int) (parsedComponentTag, error) {
	position := start + 1
	tag := parsedComponentTag{}
	if position < len(source) && source[position] == '/' {
		tag.Closing = true
		position++
	}
	nameStart := position
	for position < len(source) && (unicode.IsLetter(rune(source[position])) || unicode.IsDigit(rune(source[position])) || source[position] == '_') {
		position++
	}
	tag.Name = source[nameStart:position]
	if tag.Name == "" || !unicode.IsUpper(rune(tag.Name[0])) || !identifier.MatchString(tag.Name) {
		return parsedComponentTag{}, fmt.Errorf("invalid component tag at byte %d", start)
	}
	end, err := componentTagEnd(source, position)
	if err != nil {
		return parsedComponentTag{}, err
	}
	inside := strings.TrimSpace(source[position:end])
	if tag.Closing {
		if inside != "" {
			return parsedComponentTag{}, fmt.Errorf("closing component %s cannot contain attributes", tag.Name)
		}
		tag.End = end + 1
		return tag, nil
	}
	if strings.HasSuffix(inside, "/") {
		tag.SelfClosing = true
		inside = strings.TrimSpace(strings.TrimSuffix(inside, "/"))
	}
	tag.Attributes, err = parseComponentAttributes(inside)
	if err != nil {
		return parsedComponentTag{}, fmt.Errorf("component %s: %w", tag.Name, err)
	}
	tag.End = end + 1
	return tag, nil
}

func componentTagEnd(source string, position int) (int, error) {
	quote := byte(0)
	braceDepth := 0
	for ; position < len(source); position++ {
		current := source[position]
		if quote != 0 {
			if current == quote {
				quote = 0
			}
			continue
		}
		switch current {
		case '\'', '"':
			quote = current
		case '{':
			braceDepth++
		case '}':
			braceDepth--
		case '>':
			if braceDepth == 0 {
				return position, nil
			}
		}
	}
	return 0, fmt.Errorf("unclosed component tag")
}

func parseComponentAttributes(source string) ([]componentAttribute, error) {
	var attributes []componentAttribute
	seen := map[string]bool{}
	for position := 0; position < len(source); {
		for position < len(source) && unicode.IsSpace(rune(source[position])) {
			position++
		}
		if position == len(source) {
			break
		}
		nameStart := position
		for position < len(source) && (unicode.IsLetter(rune(source[position])) || unicode.IsDigit(rune(source[position])) || source[position] == '_' || source[position] == '-') {
			position++
		}
		name := source[nameStart:position]
		if !componentEventAttribute(name) && (!identifier.MatchString(name) || !unicode.IsUpper(rune(name[0]))) {
			return nil, fmt.Errorf("invalid prop near %q; component props use exported Go field names", source[nameStart:])
		}
		if seen[name] {
			return nil, fmt.Errorf("prop %s is specified more than once", name)
		}
		seen[name] = true
		for position < len(source) && unicode.IsSpace(rune(source[position])) {
			position++
		}
		attribute := componentAttribute{Name: name, Value: "true", Literal: true}
		if position < len(source) && source[position] == '=' {
			position++
			for position < len(source) && unicode.IsSpace(rune(source[position])) {
				position++
			}
			if position >= len(source) {
				return nil, fmt.Errorf("prop %s is missing a value", name)
			}
			if source[position] == '$' {
				if position+1 >= len(source) || source[position+1] != '{' {
					return nil, fmt.Errorf("prop %s has an invalid server interpolation", name)
				}
				position++
			}
			switch source[position] {
			case '\'', '"':
				quote := source[position]
				position++
				valueStart := position
				for position < len(source) && source[position] != quote {
					position++
				}
				if position >= len(source) {
					return nil, fmt.Errorf("prop %s has an unclosed string", name)
				}
				attribute.Value = source[valueStart:position]
				attribute.Literal = true
				attribute.Quoted = true
				position++
			case '{':
				close := strings.IndexByte(source[position+1:], '}')
				if close < 0 {
					return nil, fmt.Errorf("prop %s has an unclosed expression", name)
				}
				close += position + 1
				expression := strings.TrimSpace(source[position+1 : close])
				if expression == "true" || expression == "false" {
					attribute.Value = expression
					attribute.Literal = true
					position = close + 1
					break
				}
				expression = strings.TrimPrefix(expression, "$")
				if !validExpression(expression) {
					return nil, fmt.Errorf("prop %s has invalid expression %q", name, expression)
				}
				attribute.Value = expression
				attribute.Literal = false
				position = close + 1
			default:
				valueStart := position
				for position < len(source) && !unicode.IsSpace(rune(source[position])) {
					position++
				}
				value := source[valueStart:position]
				if value != "true" && value != "false" {
					if _, err := strconv.ParseFloat(value, 64); err != nil {
						return nil, fmt.Errorf("prop %s must use a quoted literal or ${expression}", name)
					}
				}
				attribute.Value = value
				attribute.Literal = true
			}
		}
		attributes = append(attributes, attribute)
	}
	return attributes, nil
}

func componentEventAttribute(name string) bool {
	if !strings.HasPrefix(name, "data-north-event-") {
		return false
	}
	suffix := strings.TrimPrefix(name, "data-north-event-")
	if suffix == "" {
		return false
	}
	for _, current := range suffix {
		if !unicode.IsLetter(current) && !unicode.IsDigit(current) && current != '-' {
			return false
		}
	}
	return true
}
