package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var goArrayType = regexp.MustCompile(`^\[[^]]*\](.+)$`)

func typeScriptContract(name string, props []prop) string {
	var output strings.Builder
	fmt.Fprintf(&output, "type %s = {\n", name)
	for _, field := range props {
		fmt.Fprintf(&output, "  readonly %s: %s;\n", field.Name, goTypeToTypeScript(field.Type))
	}
	output.WriteString("};")
	return output.String()
}

func goTypeToTypeScript(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "*") {
		return goTypeToTypeScript(strings.TrimPrefix(value, "*")) + " | null"
	}
	if strings.HasPrefix(value, "[]") {
		return "ReadonlyArray<" + goTypeToTypeScript(strings.TrimPrefix(value, "[]")) + ">"
	}
	if match := goArrayType.FindStringSubmatch(value); match != nil {
		return "ReadonlyArray<" + goTypeToTypeScript(match[1]) + ">"
	}
	if strings.HasPrefix(value, "map[string]") {
		return "Readonly<Record<string, " + goTypeToTypeScript(strings.TrimPrefix(value, "map[string]")) + ">>"
	}
	switch value {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "uint", "uint8", "uint16", "uint32", "float32", "float64":
		return "number"
	case "int64", "uint64":
		return "number"
	case "time.Time":
		return "string"
	case "any", "interface{}":
		return "unknown"
	}
	return "unknown"
}
