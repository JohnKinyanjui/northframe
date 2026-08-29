package admin

import (
	"fmt"
	"mime"
	"net/http"
	"strconv"
	"strings"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func decodeRecord(request *http.Request, resource Resource) (Record, web.FieldErrors, error) {
	mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if mediaType == "multipart/form-data" {
		if err := request.ParseMultipartForm(8 << 20); err != nil {
			return nil, nil, err
		}
	} else if err := request.ParseForm(); err != nil {
		return nil, nil, err
	}
	record := make(Record)
	errors := make(web.FieldErrors)
	for _, field := range resource.Fields {
		if field.ReadOnly || field.Hidden {
			continue
		}
		raw := strings.TrimSpace(request.FormValue(field.Name))
		if field.Required && raw == "" && field.Kind != FieldBoolean {
			errors[field.Name] = field.Label + " is required"
			continue
		}
		if raw == "" && field.Kind != FieldBoolean {
			record[field.Name] = ""
			continue
		}
		switch field.Kind {
		case FieldBoolean:
			record[field.Name] = raw == "on" || raw == "true" || raw == "1"
		case FieldNumber, FieldMoney:
			value, err := strconv.ParseFloat(raw, 64)
			if err != nil {
				errors[field.Name] = field.Label + " must be a number"
				continue
			}
			record[field.Name] = value
		case FieldSelect:
			if !validOption(field.Options, raw) {
				errors[field.Name] = field.Label + " has an invalid value"
				continue
			}
			record[field.Name] = raw
		default:
			record[field.Name] = raw
		}
	}
	if resource.Validate != nil {
		for name, message := range resource.Validate(request.Context(), record) {
			if _, exists := errors[name]; !exists {
				errors[name] = message
			}
		}
	}
	if len(errors) == 0 {
		errors = nil
	}
	return record, errors, nil
}

func validOption(options []Option, value string) bool {
	for _, option := range options {
		if option.Value == value {
			return true
		}
	}
	return false
}

func recordID(record Record) string {
	for _, key := range []string{"id", "ID", "Id"} {
		if value, ok := record[key]; ok {
			return fmt.Sprint(value)
		}
	}
	return ""
}
