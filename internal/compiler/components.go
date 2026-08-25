package compiler

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type componentView struct {
	Name         string
	Path         string
	Source       []byte
	Contract     string
	Props        []prop
	Imports      []componentImport
	HasSlot      bool
	ClientModule *clientModule
}

func discoverComponents(root string) ([]componentView, error) {
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return nil, nil
	} else if err != nil {
		return nil, fmt.Errorf("inspect components directory: %w", err)
	}
	seen := map[string]string{}
	var result []componentView
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || filepath.Ext(path) != ".north" {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := exportedName(strings.TrimSuffix(filepath.ToSlash(relative), ".north"))
		if previous := seen[name]; previous != "" {
			return fmt.Errorf("component name %s collides between %s and %s", name, previous, path)
		}
		seen[name] = path
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		props, imports, err := componentProps(source)
		if err != nil {
			return fmt.Errorf("component %s: %w", name, err)
		}
		result = append(result, componentView{
			Name: name, Path: path, Source: source, Props: props, Imports: imports, HasSlot: strings.Contains(string(source), "<slot"), Contract: typeScriptContract(name+"Props", props),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result, nil
}

func componentProps(source []byte) ([]prop, []componentImport, error) {
	block, _, found, err := extractPropsBlock(string(source))
	if err != nil {
		return nil, nil, err
	}
	if !found {
		return nil, nil, nil
	}
	return parseProps(block)
}

func validateComponentReferences(owner string, source []byte, known map[string]componentView) error {
	withoutClient := typeScriptBlock.ReplaceAll(source, nil)
	parsed, err := parseComponent(owner, string(withoutClient))
	if err != nil {
		return err
	}
	return validateComponentNodes(owner, parsed.Children, known)
}

func validateComponentNodes(owner string, nodes []node, known map[string]componentView) error {
	for _, raw := range nodes {
		switch current := raw.(type) {
		case componentNode:
			definition, exists := known[current.Name]
			if !exists {
				return fmt.Errorf("%s references unknown component <%s>; create web/components/%s.north", owner, current.Name, toSnakeCase(current.Name))
			}
			provided := map[string]bool{}
			allowed := map[string]bool{}
			for _, field := range definition.Props {
				allowed[field.Name] = true
			}
			for _, attribute := range current.Attributes {
				if !allowed[attribute.Name] {
					return fmt.Errorf("%s passes unknown prop %s to <%s>", owner, attribute.Name, current.Name)
				}
				provided[attribute.Name] = true
			}
			for _, field := range definition.Props {
				if !provided[field.Name] && !strings.HasPrefix(strings.TrimSpace(field.Type), "*") {
					return fmt.Errorf("%s must pass required prop %s to <%s>", owner, field.Name, current.Name)
				}
			}
			if len(current.Children) > 0 && !definition.HasSlot {
				return fmt.Errorf("%s passes child content to <%s>, but the component has no <slot />", owner, current.Name)
			}
			if err := validateComponentNodes(owner, current.Children, known); err != nil {
				return err
			}
		case ifNode:
			if err := validateComponentNodes(owner, current.Children, known); err != nil {
				return err
			}
		case eachNode:
			if err := validateComponentNodes(owner, current.Children, known); err != nil {
				return err
			}
		}
	}
	return nil
}
