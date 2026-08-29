// Package assets exposes Northframe's shared brand assets to framework-owned
// applications such as the documentation site.
package assets

import _ "embed"

// Logo is the embedded Northframe flame mark.
//
//go:embed logo.png
var Logo []byte
