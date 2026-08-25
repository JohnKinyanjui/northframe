package compiler

import (
	"fmt"
	"strings"
)

func prepareLayout(source []byte, root bool) ([]byte, error) {
	text := string(source)
	if !strings.Contains(text, "<slot />") && !strings.Contains(text, "<slot/>") {
		return nil, fmt.Errorf("layout must contain <slot />")
	}
	if root {
		if !strings.Contains(text, "</head>") || !strings.Contains(text, "</body>") {
			return nil, fmt.Errorf("root layout must contain <head> and <body>")
		}
		text = strings.Replace(text, "</head>", `<link rel="stylesheet" href="/_northframe/app.css"></head>`, 1)
		text = strings.Replace(text, "</body>", `<script type="module" src="/_northframe/runtime.js"></script></body>`, 1)
	}
	return []byte(text), nil
}
