package compiler

import (
	"fmt"
	"go/parser"
	"strings"
)

func parseNodes(source string, position int) ([]node, int, string, error) {
	var nodes []node
	for position < len(source) {
		open, token := nextTemplateToken(source, position)
		if open < 0 {
			nodes = appendText(nodes, source[position:])
			return nodes, len(source), "", nil
		}
		nodes = appendText(nodes, source[position:open])
		if token == '<' {
			tag, err := parseComponentTag(source, open)
			if err != nil {
				return nil, open, "", err
			}
			position = tag.End
			if tag.Closing {
				return nodes, position, "component:" + tag.Name, nil
			}
			current := componentNode{Name: tag.Name, Attributes: tag.Attributes}
			if !tag.SelfClosing {
				children, next, end, childErr := parseNodes(source, position)
				if childErr != nil {
					return nil, next, "", childErr
				}
				if end != "component:"+tag.Name {
					return nil, next, "", fmt.Errorf("component %s at byte %d closed by %q", tag.Name, open, end)
				}
				current.Children = children
				position = next
			}
			nodes = append(nodes, current)
			continue
		}
		closeOffset := strings.IndexByte(source[open+1:], '}')
		if closeOffset < 0 {
			return nil, open, "", fmt.Errorf("unclosed expression at byte %d", open)
		}
		close := open + closeOffset + 1
		directive := strings.TrimSpace(source[open+1 : close])
		position = close + 1
		if strings.HasPrefix(directive, "__north_slot ") {
			name := strings.TrimSpace(strings.TrimPrefix(directive, "__north_slot"))
			if !identifier.MatchString(name) {
				return nil, open, "", fmt.Errorf("invalid named slot %q at byte %d", name, open)
			}
			nodes = append(nodes, slotNode{Name: name})
			continue
		}
		if strings.HasPrefix(directive, "#if ") || strings.HasPrefix(directive, "#each ") {
			return nil, open, "", fmt.Errorf("Svelte-style server blocks are no longer supported at byte %d; use `{if Props.Condition}` or `{for item := range Props.Items}`", open)
		}

		if strings.HasPrefix(directive, "/") {
			return nodes, position, strings.TrimPrefix(directive, "/"), nil
		}
		if match := ifStart.FindStringSubmatch(directive); match != nil {
			condition := strings.TrimSpace(match[1])
			if !validCondition(condition) {
				return nil, open, "", fmt.Errorf("invalid if expression %q at byte %d", condition, open)
			}
			children, next, end, err := parseNodes(source, position)
			if err != nil {
				return nil, next, "", err
			}
			if end != "if" {
				return nil, next, "", fmt.Errorf("if block at byte %d closed by %q", open, end)
			}
			nodes = append(nodes, ifNode{Condition: condition, Children: children})
			position = next
			continue
		}
		if match := forStart.FindStringSubmatch(directive); match != nil {
			collection := strings.TrimSpace(match[2])
			if !validExpression(collection) {
				return nil, open, "", fmt.Errorf("invalid range expression %q at byte %d", collection, open)
			}
			children, next, end, err := parseNodes(source, position)
			if err != nil {
				return nil, next, "", err
			}
			if end != "for" {
				return nil, next, "", fmt.Errorf("for block at byte %d closed by %q", open, end)
			}
			nodes = append(nodes, eachNode{Collection: collection, Item: match[1], Children: children})
			position = next
			continue
		}
		if forKeyword.MatchString(directive) {
			return nil, open, "", fmt.Errorf("invalid for expression %q at byte %d; use `{for item := range Props.Items}`", directive, open)
		}
		if match := htmlStart.FindStringSubmatch(directive); match != nil {
			expression := strings.TrimSpace(match[1])
			if !validExpression(expression) {
				return nil, open, "", fmt.Errorf("invalid html expression %q at byte %d", expression, open)
			}
			nodes = append(nodes, htmlNode{Expression: expression})
			continue
		}
		if !validExpression(directive) {
			return nil, open, "", fmt.Errorf("invalid interpolation %q at byte %d", directive, open)
		}
		nodes = append(nodes, exprNode{Expression: directive})
	}
	return nodes, position, "", nil
}

func nextTemplateToken(source string, position int) (int, byte) {
	brace := strings.IndexByte(source[position:], '{')
	if brace >= 0 {
		brace += position
	}
	tag := nextComponentTag(source, position)
	if tag >= 0 && (brace < 0 || tag < brace) {
		return tag, '<'
	}
	if brace >= 0 {
		return brace, '{'
	}
	return -1, 0
}

func appendText(nodes []node, value string) []node {
	if value == "" {
		return nodes
	}
	return append(nodes, textNode{Value: value})
}

func validExpression(value string) bool {
	if strings.TrimSpace(value) == "" {
		return false
	}
	_, err := parser.ParseExpr(value)
	return err == nil
}

func validCondition(value string) bool {
	return validExpression(value)
}
