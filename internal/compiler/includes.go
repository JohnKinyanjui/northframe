package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var includeDirective = regexp.MustCompile(`\{#include\s+([A-Za-z0-9_/-]+)\}`)

func expandIncludes(source []byte, componentsRoot string, stack []string) ([]byte, error) {
	text := string(source)
	for {
		match := includeDirective.FindStringSubmatchIndex(text)
		if match == nil {
			return []byte(text), nil
		}
		name := text[match[2]:match[3]]
		cleanName := filepath.ToSlash(filepath.Clean(name))
		if cleanName == "." || strings.HasPrefix(cleanName, "../") || filepath.IsAbs(name) {
			return nil, fmt.Errorf("invalid component include %q", name)
		}
		for _, active := range stack {
			if active == cleanName {
				return nil, fmt.Errorf("component include cycle involving %q", cleanName)
			}
		}
		componentPath := filepath.Join(componentsRoot, filepath.FromSlash(cleanName)+".north")
		component, err := os.ReadFile(componentPath)
		if err != nil {
			if os.IsNotExist(err) {
				return nil, fmt.Errorf("component %q not found at %s", cleanName, componentPath)
			}
			return nil, err
		}
		expanded, err := expandIncludes(component, componentsRoot, append(stack, cleanName))
		if err != nil {
			return nil, err
		}
		text = text[:match[0]] + string(expanded) + text[match[1]:]
	}
}
