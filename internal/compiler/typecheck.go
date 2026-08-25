package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	typedStateDeclaration    = regexp.MustCompile(`(?m)\b(?:let|const)\s+([A-Za-z_][A-Za-z0-9_]*)\s*:\s*([^=;\n]+)\s*=\s*([^;\n]+)`)
	inferredStateDeclaration = regexp.MustCompile(`(?m)\b(?:let|const)\s+([A-Za-z_][A-Za-z0-9_]*)\s*=\s*([^;\n]+)`)
	functionDeclaration      = regexp.MustCompile(`(?m)\bfunction\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(`)
	propsAccess              = regexp.MustCompile(`\bprops\.([A-Za-z_][A-Za-z0-9_]*)`)
	simpleAssignment         = regexp.MustCompile(`(?m)\b([A-Za-z_][A-Za-z0-9_]*)\s*=\s*(true|false|-?[0-9]+(?:\.[0-9]+)?|"[^"\n]*"|'[^'\n]*')\s*;`)
)

func checkClientTypes(componentName, source string, bindings []clientBinding, events []clientEventBinding, props []prop) error {
	state := map[string]string{}
	propTypes := map[string]string{}
	for _, field := range props {
		propTypes[field.Name] = goTypeToTypeScript(field.Type)
	}
	for _, match := range typedStateDeclaration.FindAllStringSubmatch(source, -1) {
		declared := normalizeTypeScriptType(match[2])
		state[match[1]] = declared
		if inferred := inferClientExpressionType(strings.TrimSpace(match[3]), state, propTypes); inferred != "" && !typesCompatible(declared, inferred) {
			return fmt.Errorf("%s TypeScript: %s is declared %s but initialized with %s", componentName, match[1], declared, inferred)
		}
	}
	for _, match := range inferredStateDeclaration.FindAllStringSubmatch(source, -1) {
		if _, exists := state[match[1]]; exists {
			continue
		}
		if inferred := inferClientExpressionType(strings.TrimSpace(match[2]), state, propTypes); inferred != "" {
			state[match[1]] = inferred
		}
	}
	functions := map[string]bool{}
	for _, match := range functionDeclaration.FindAllStringSubmatch(source, -1) {
		functions[match[1]] = true
	}
	for _, match := range propsAccess.FindAllStringSubmatch(source, -1) {
		if _, exists := propTypes[match[1]]; !exists {
			return fmt.Errorf("%s TypeScript: PageProps has no field %s", componentName, match[1])
		}
	}
	for _, binding := range bindings {
		if err := checkClientExpression(componentName, binding.Expression, state, propTypes); err != nil {
			return err
		}
	}
	for _, event := range events {
		handler := strings.TrimSpace(event.HandlerTS)
		if identifier.MatchString(handler) && !functions[handler] && state[handler] == "" {
			return fmt.Errorf("%s TypeScript: event handler %s is not declared", componentName, handler)
		}
	}
	for _, match := range simpleAssignment.FindAllStringSubmatch(source, -1) {
		declared := state[match[1]]
		assigned := inferLiteralType(match[2])
		if declared != "" && assigned != "" && !typesCompatible(declared, assigned) {
			return fmt.Errorf("%s TypeScript: cannot assign %s to %s (%s)", componentName, assigned, match[1], declared)
		}
	}
	return nil
}

func checkClientExpression(componentName, expression string, state, props map[string]string) error {
	parts := strings.Split(expression, ".")
	if parts[0] == "props" {
		if len(parts) < 2 || props[parts[1]] == "" {
			return fmt.Errorf("%s TypeScript: PageProps has no field %s", componentName, strings.Join(parts[1:], "."))
		}
		return nil
	}
	if state[parts[0]] == "" {
		return fmt.Errorf("%s TypeScript: browser state %s is not declared", componentName, parts[0])
	}
	return nil
}

func normalizeTypeScriptType(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func inferLiteralType(value string) string {
	value = strings.TrimSpace(value)
	if value == "true" || value == "false" {
		return "boolean"
	}
	if strings.HasPrefix(value, `"`) || strings.HasPrefix(value, `'`) {
		return "string"
	}
	if regexp.MustCompile(`^-?[0-9]+(?:\.[0-9]+)?$`).MatchString(value) {
		return "number"
	}
	return ""
}

func inferClientExpressionType(value string, state, props map[string]string) string {
	if literal := inferLiteralType(value); literal != "" {
		return literal
	}
	if strings.HasPrefix(value, "props.") {
		return props[strings.TrimPrefix(value, "props.")]
	}
	if identifier.MatchString(value) {
		return state[value]
	}
	return ""
}

func typesCompatible(declared, assigned string) bool {
	return declared == assigned || declared == "unknown" || strings.Contains(declared, assigned)
}
