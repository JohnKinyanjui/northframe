package compiler

import (
	"fmt"
	"strings"
)

func routeComponentName(directory, kind string) string {
	if directory == "." {
		if kind == "page" {
			return "HomePage"
		}
		return "RootLayout"
	}
	return exportedName(strings.ReplaceAll(directory, "/", "-")) + exportedName(kind)
}

func routeURL(directory string) (string, error) {
	if directory == "." {
		return "/", nil
	}
	parts := strings.Split(strings.Trim(directory, "/"), "/")
	for index, part := range parts {
		switch {
		case strings.HasSuffix(part, "__"):
			name := strings.TrimSuffix(part, "__")
			if name == "" || index != len(parts)-1 {
				return "", fmt.Errorf("catch-all parameter folders must be named like path__ and appear last")
			}
			parts[index] = "{" + name + "...}"
		case strings.HasSuffix(part, "_"):
			name := strings.TrimSuffix(part, "_")
			if name == "" {
				return "", fmt.Errorf("parameter folder must be named like id_")
			}
			parts[index] = "{" + name + "}"
		}
	}
	return "/" + strings.Join(parts, "/"), nil
}

func routeAlias(directory string) string {
	if directory == "." {
		return "routeRoot"
	}
	return "route" + exportedName(strings.ReplaceAll(directory, "/", "-"))
}

func lowerFirst(value string) string {
	if value == "" {
		return value
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func toSnakeCase(value string) string {
	var output strings.Builder
	for index, current := range value {
		if index > 0 && current >= 'A' && current <= 'Z' {
			output.WriteByte('_')
		}
		output.WriteRune(current)
	}
	return strings.ToLower(output.String())
}
