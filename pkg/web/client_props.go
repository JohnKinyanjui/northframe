package web

import (
	"encoding/json"
	"fmt"
	"html"
	"io"
)

// WriteClientProps serializes SSR props into an inert JSON script consumed by
// the generated TypeScript module. encoding/json escapes HTML-significant
// characters, so user content cannot terminate the script element.
func WriteClientProps(w io.Writer, marker string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode client props: %w", err)
	}
	if _, err := io.WriteString(w, `<script type="application/json" data-north-props="`+html.EscapeString(marker)+`">`); err != nil {
		return err
	}
	if _, err := w.Write(encoded); err != nil {
		return err
	}
	_, err = io.WriteString(w, "</script>")
	return err
}
