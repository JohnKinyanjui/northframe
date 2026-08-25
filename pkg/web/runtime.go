// Package web provides the HTTP application and rendering runtime shared by generated routes.
package web

import (
	"fmt"
	"html"
	"io"
	"reflect"
)

// Fragment is a native SSR render function. Layouts receive their child page
// as a Fragment, so nesting never needs a server-side JavaScript runtime.
type Fragment func(io.Writer) error

// WriteEscaped writes a value into HTML text or an HTML attribute safely.
func WriteEscaped(w io.Writer, value any) error {
	_, err := io.WriteString(w, html.EscapeString(fmt.Sprint(value)))
	return err
}

// Truthy provides predictable conditional rendering for generated templates.
func Truthy(value any) bool {
	if value == nil {
		return false
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Bool:
		return v.Bool()
	case reflect.String, reflect.Array, reflect.Slice, reflect.Map:
		return v.Len() > 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return v.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return v.Uint() != 0
	case reflect.Float32, reflect.Float64:
		return v.Float() != 0
	case reflect.Interface, reflect.Pointer:
		return !v.IsNil()
	default:
		return true
	}
}
