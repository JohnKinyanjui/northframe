package compiler

import (
	"fmt"
	"regexp"
	"strings"
)

var (
	propsBlock            = regexp.MustCompile(`(?s)<script\s+context=["']props["']\s*>(.*?)</script>`)
	frontmatterPropsBlock = regexp.MustCompile(`(?ms)\A[ \t]*---[ \t]*\r?\n(.*?)^[ \t]*---[ \t]*(?:\r?\n|\z)`)
	frontmatterPropsType  = regexp.MustCompile(`(?ms)^[ \t]*interface[ \t]+Props[ \t]*\{[ \t]*\r?\n(.*?)^[ \t]*\}[ \t]*(?:\r?\n|\z)`)
	interfacePropsBlock   = regexp.MustCompile(`(?ms)^[ \t]*interface[ \t]+props[ \t]*\{[ \t]*\r?\n(.*?)^[ \t]*\}[ \t]*(?:\r?\n|$)`)
	styleBlock            = regexp.MustCompile(`(?s)<style\s*>(.*?)</style>`)
	namedSlotTag          = regexp.MustCompile(`<slot\s+name=["']([A-Za-z_][A-Za-z0-9_]*)["']\s*/?>`)
	identifier            = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
	forStart              = regexp.MustCompile(`^for\s+([a-z_][A-Za-z0-9_]*)\s*:=\s*range\s+(.+)$`)
)

const (
	styleSentinel          = "\x00NORTHFRAME_COMPONENT_STYLE\x00"
	slotSentinel           = "\x00NORTHFRAME_LAYOUT_SLOT\x00"
	propsSentinel          = "\x00NORTHFRAME_CLIENT_PROPS\x00"
	interfacePropsSentinel = "\x00NORTHFRAME_INTERFACE_PROPS\x00"
)

func parseComponent(name, source string) (component, error) {
	result := component{Name: name}
	source = namedSlotTag.ReplaceAllString(source, `{__north_slot $1}`)
	source = strings.ReplaceAll(source, "<slot />", slotSentinel)
	source = strings.ReplaceAll(source, "<slot/>", slotSentinel)

	if block, legacy, found, err := extractPropsBlock(source); err != nil {
		return component{}, err
	} else if found {
		props, imports, err := parseProps(block)
		if err != nil {
			return component{}, err
		}
		result.Props = props
		result.Imports = imports
		if legacy {
			source = propsBlock.ReplaceAllString(source, "")
		} else {
			source = frontmatterPropsBlock.ReplaceAllString(source, "")
			source = interfacePropsBlock.ReplaceAllString(source, "")
		}
	}
	if match := styleBlock.FindStringSubmatchIndex(source); match != nil {
		result.Style = strings.TrimSpace(source[match[2]:match[3]])
		source = source[:match[0]] + styleSentinel + source[match[1]:]
	}

	children, position, end, err := parseNodes(source, 0)
	if err != nil {
		return component{}, err
	}
	if end != "" {
		return component{}, fmt.Errorf("unexpected closing block %q at byte %d", end, position)
	}
	result.Children = restoreProps(restoreSlot(restoreStyle(children, result.Style)))
	return result, nil
}

func extractPropsBlock(source string) (block string, legacy, found bool, err error) {
	legacyMatch := propsBlock.FindStringSubmatch(source)
	frontmatterMatch := frontmatterPropsBlock.FindStringSubmatch(source)
	interfaceMatch := interfacePropsBlock.FindStringSubmatch(source)
	foundCount := 0
	for _, match := range [][]string{legacyMatch, frontmatterMatch, interfaceMatch} {
		if match != nil {
			foundCount++
		}
	}
	if foundCount > 1 {
		return "", false, false, fmt.Errorf("declare props once in `--- interface Props { ... } ---` frontmatter")
	}
	if frontmatterMatch != nil {
		block, err := normalizeFrontmatterProps(frontmatterMatch[1])
		return block, false, err == nil, err
	}
	if interfaceMatch != nil {
		return interfaceMatch[1], false, true, nil
	}
	if legacyMatch != nil {
		return legacyMatch[1], true, true, nil
	}
	return "", false, false, nil
}

func normalizeFrontmatterProps(content string) (string, error) {
	matches := frontmatterPropsType.FindAllStringSubmatchIndex(content, -1)
	if len(matches) != 1 {
		return "", fmt.Errorf("frontmatter must declare exactly one `interface Props { ... }`")
	}
	match := matches[0]
	body := content[match[2]:match[3]]
	outside := content[:match[0]] + "\n" + content[match[1]:]
	imports := make([]string, 0)
	for _, raw := range strings.Split(outside, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.HasPrefix(line, "import ") {
			return "", fmt.Errorf("frontmatter only allows Go imports before `interface Props`")
		}
		imports = append(imports, line)
	}
	return strings.Join(append(imports, body), "\n"), nil
}

func restoreProps(nodes []node) []node {
	return restoreSentinel(nodes, propsSentinel, func() node { return propsNode{} })
}

func restoreSlot(nodes []node) []node {
	return restoreSentinel(nodes, slotSentinel, func() node { return slotNode{} })
}

func restoreStyle(nodes []node, style string) []node {
	return restoreSentinel(nodes, styleSentinel, func() node { return styleNode{Value: style} })
}

func restoreSentinel(nodes []node, sentinel string, replacement func() node) []node {
	var restored []node
	for _, raw := range nodes {
		switch current := raw.(type) {
		case textNode:
			parts := strings.Split(current.Value, sentinel)
			for index, part := range parts {
				restored = appendText(restored, part)
				if index < len(parts)-1 {
					restored = append(restored, replacement())
				}
			}
		case ifNode:
			current.Children = restoreSentinel(current.Children, sentinel, replacement)
			restored = append(restored, current)
		case eachNode:
			current.Children = restoreSentinel(current.Children, sentinel, replacement)
			restored = append(restored, current)
		case componentNode:
			current.Children = restoreSentinel(current.Children, sentinel, replacement)
			restored = append(restored, current)
		default:
			restored = append(restored, raw)
		}
	}
	return restored
}
