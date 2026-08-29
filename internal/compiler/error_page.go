package compiler

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const errorPageProps = `---
interface Props {
    Status int
    Message string
    Path string
    RequestID string
}
---
`

func discoverErrorPage(routesDirectory string) (*componentView, error) {
	path := filepath.Join(routesDirectory, "error.north")
	source, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read root error.north: %w", err)
	}
	if _, _, found, propsErr := extractPropsBlock(string(source)); propsErr != nil {
		return nil, propsErr
	} else if found {
		return nil, fmt.Errorf("%s: error.north has implicit Status, Message, Path, and RequestID props; remove its Props declaration", path)
	}
	return &componentView{
		Name: "ApplicationError", Path: path, Source: append([]byte(errorPageProps), source...),
		Props: []prop{{Name: "Status", Type: "int"}, {Name: "Message", Type: "string"}, {Name: "Path", Type: "string"}, {Name: "RequestID", Type: "string"}},
	}, nil
}

func prepareErrorPage(source []byte) ([]byte, error) {
	text := string(source)
	if !strings.Contains(text, "</head>") || !strings.Contains(text, "</body>") {
		return nil, errors.New("root error.north must be a complete HTML document with <head> and <body>")
	}
	text = strings.Replace(text, "</head>", `<link rel="stylesheet" href="/_northframe/app.css"></head>`, 1)
	text = strings.Replace(text, "</body>", `<script type="module" src="/_northframe/runtime.js"></script></body>`, 1)
	return []byte(text), nil
}
