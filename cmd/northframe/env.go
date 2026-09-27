package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

func loadDotEnv(path string, base []string) ([]string, error) {
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return base, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	result := append([]string(nil), base...)
	existing := environmentNames(base)
	for index, raw := range strings.Split(string(contents), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimSpace(strings.TrimPrefix(line, "export "))
		name, value, found := strings.Cut(line, "=")
		name = strings.TrimSpace(name)
		if !found || !validEnvironmentName(name) {
			return nil, fmt.Errorf("%s:%d: invalid environment assignment", path, index+1)
		}
		if _, isSet := existing[name]; isSet {
			continue
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '\'' && value[len(value)-1] == '\'') || (value[0] == '"' && value[len(value)-1] == '"')) {
			value = value[1 : len(value)-1]
		}
		result = append(result, name+"="+value)
		existing[name] = struct{}{}
	}
	return result, nil
}

func environmentNames(values []string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		name, _, found := strings.Cut(value, "=")
		if found {
			result[name] = struct{}{}
		}
	}
	return result
}

func validEnvironmentName(value string) bool {
	if value == "" || !(value[0] == '_' || value[0] >= 'A' && value[0] <= 'Z' || value[0] >= 'a' && value[0] <= 'z') {
		return false
	}
	for index := 1; index < len(value); index++ {
		character := value[index]
		if character != '_' && !(character >= 'A' && character <= 'Z') && !(character >= 'a' && character <= 'z') && !(character >= '0' && character <= '9') {
			return false
		}
	}
	return true
}

func setEnvironment(values []string, name, value string) []string {
	prefix := name + "="
	result := make([]string, 0, len(values)+1)
	for _, current := range values {
		if !strings.HasPrefix(current, prefix) {
			result = append(result, current)
		}
	}
	return append(result, prefix+value)
}
