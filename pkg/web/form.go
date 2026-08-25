package web

import (
	"encoding"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

const maxFormMemory = 8 << 20

var textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
var fileHeaderPointerType = reflect.TypeOf((*multipart.FileHeader)(nil))

type parsedForm struct {
	values map[string][]string
	files  map[string][]*multipart.FileHeader
}

// FieldErrors maps HTML form field names to safe, user-facing messages.
type FieldErrors map[string]string

// DecodeForm decodes and validates a request into a struct. Exported fields use
// their form tag, or their snake_case Go name when the tag is omitted.
//
// Supported validation rules are required, email, min=<number>, and
// max=<number>. min/max measure string length for strings and numeric value for
// numbers. Types implementing encoding.TextUnmarshaler can define custom input
// formats without changing Northframe's decoder.
func DecodeForm[T any](request *http.Request) (T, FieldErrors, error) {
	var result T
	form, err := parseRequestForm(request)
	if err != nil {
		return result, nil, fmt.Errorf("parse form: %w", err)
	}
	value := reflect.ValueOf(&result).Elem()
	if value.Kind() != reflect.Struct {
		return result, nil, fmt.Errorf("northframe: form input %T must be a struct", result)
	}

	errors := make(FieldErrors)
	decodeStruct(value, form, errors)
	if len(errors) == 0 {
		errors = nil
	}
	return result, errors, nil
}

func parseRequestForm(request *http.Request) (parsedForm, error) {
	mediaType, _, _ := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if mediaType == "multipart/form-data" {
		if err := request.ParseMultipartForm(maxFormMemory); err != nil {
			return parsedForm{}, err
		}
	} else if err := request.ParseForm(); err != nil {
		return parsedForm{}, err
	}
	result := parsedForm{values: request.Form}
	if request.MultipartForm != nil {
		result.files = request.MultipartForm.File
	}
	return result, nil
}

func decodeStruct(destination reflect.Value, form parsedForm, errors FieldErrors) {
	typeInfo := destination.Type()
	for index := 0; index < destination.NumField(); index++ {
		fieldInfo := typeInfo.Field(index)
		field := destination.Field(index)
		if !fieldInfo.IsExported() || !field.CanSet() {
			continue
		}
		name := formFieldName(fieldInfo)
		if name == "-" {
			continue
		}
		if isFileField(field.Type()) {
			decodeFileField(field, form.files[name], fieldInfo, name, errors)
			continue
		}
		rawValues, exists := form.values[name]
		raw := ""
		if len(rawValues) > 0 {
			raw = rawValues[0]
		}
		label := fieldLabel(fieldInfo, name)
		rules := fieldInfo.Tag.Get("validate")
		if message := validateRawFormValue(raw, exists, rules, label, field.Type()); message != "" {
			errors[name] = message
			continue
		}
		if !exists {
			continue
		}
		if raw == "" && !hasValidationRule(rules, "required") {
			continue
		}
		if err := assignFormValue(field, rawValues); err != nil {
			errors[name] = fmt.Sprintf("%s has an invalid value", label)
			continue
		}
		if message := validateDecodedFormValue(field, rules, label); message != "" {
			errors[name] = message
		}
	}
}

func isFileField(fieldType reflect.Type) bool {
	if fieldType == fileHeaderPointerType {
		return true
	}
	return fieldType.Kind() == reflect.Slice && fieldType.Elem() == fileHeaderPointerType
}

func decodeFileField(destination reflect.Value, files []*multipart.FileHeader, fieldInfo reflect.StructField, name string, fieldErrors FieldErrors) {
	label := fieldLabel(fieldInfo, name)
	if message := validateFiles(files, fieldInfo.Tag.Get("validate"), fieldInfo.Tag.Get("accept"), label); message != "" {
		fieldErrors[name] = message
		return
	}
	if len(files) == 0 {
		return
	}
	if destination.Type() == fileHeaderPointerType {
		destination.Set(reflect.ValueOf(files[0]))
		return
	}
	destination.Set(reflect.ValueOf(files))
}

func validateFiles(files []*multipart.FileHeader, rules, accepted, label string) string {
	if hasValidationRule(rules, "required") && len(files) == 0 {
		return label + " is required"
	}
	for _, file := range files {
		for _, rule := range validationRules(rules) {
			if rule.name == "maxbytes" && rule.value != "" {
				maximum, err := strconv.ParseInt(rule.value, 10, 64)
				if err == nil && maximum >= 0 && file.Size > maximum {
					return fmt.Sprintf("%s must be at most %s bytes", label, rule.value)
				}
			}
		}
		if accepted != "" {
			contentType, err := UploadedContentType(file)
			if err != nil || !contentTypeAccepted(contentType, accepted) {
				return label + " has an unsupported file type"
			}
		}
	}
	return ""
}

func contentTypeAccepted(contentType, accepted string) bool {
	for _, candidate := range strings.Split(accepted, ",") {
		candidate = strings.ToLower(strings.TrimSpace(candidate))
		if candidate == "" {
			continue
		}
		if candidate == contentType {
			return true
		}
		if strings.HasSuffix(candidate, "/*") && strings.HasPrefix(contentType, strings.TrimSuffix(candidate, "*")) {
			return true
		}
	}
	return false
}

// UploadedContentType detects a file's media type from its first 512 bytes
// instead of trusting the browser-provided Content-Type header.
func UploadedContentType(header *multipart.FileHeader) (string, error) {
	if header == nil {
		return "", fmt.Errorf("uploaded file is nil")
	}
	file, err := header.Open()
	if err != nil {
		return "", fmt.Errorf("open uploaded file: %w", err)
	}
	defer file.Close()
	buffer := make([]byte, 512)
	read, err := file.Read(buffer)
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("inspect uploaded file: %w", err)
	}
	return strings.ToLower(http.DetectContentType(buffer[:read])), nil
}

// UploadedFilename returns a display-safe base name. Applications should
// generate their own storage key instead of using it as a destination path.
func UploadedFilename(header *multipart.FileHeader) string {
	if header == nil {
		return ""
	}
	return filepath.Base(strings.ReplaceAll(header.Filename, "\\", "/"))
}

// SaveUploadedFile atomically streams an upload to an explicit application
// path. maxBytes <= 0 disables the size limit. Existing destinations are
// replaced only after the complete file has been written and synced.
func SaveUploadedFile(header *multipart.FileHeader, destination string, maxBytes int64) (saveErr error) {
	if header == nil {
		return fmt.Errorf("uploaded file is nil")
	}
	if maxBytes > 0 && header.Size > maxBytes {
		return fmt.Errorf("uploaded file exceeds %d bytes", maxBytes)
	}
	source, err := header.Open()
	if err != nil {
		return fmt.Errorf("open uploaded file: %w", err)
	}
	defer source.Close()

	directory := filepath.Dir(destination)
	temporary, err := os.CreateTemp(directory, ".northframe-upload-*")
	if err != nil {
		return fmt.Errorf("create temporary upload: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if saveErr != nil {
			_ = os.Remove(temporaryPath)
		}
	}()
	if err := temporary.Chmod(0o644); err != nil {
		return fmt.Errorf("set upload permissions: %w", err)
	}
	reader := io.Reader(source)
	if maxBytes > 0 {
		reader = io.LimitReader(source, maxBytes+1)
	}
	written, err := io.Copy(temporary, reader)
	if err != nil {
		return fmt.Errorf("write upload: %w", err)
	}
	if maxBytes > 0 && written > maxBytes {
		return fmt.Errorf("uploaded file exceeds %d bytes", maxBytes)
	}
	if err := temporary.Sync(); err != nil {
		return fmt.Errorf("sync upload: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close upload: %w", err)
	}
	if err := os.Rename(temporaryPath, destination); err != nil {
		return fmt.Errorf("store upload: %w", err)
	}
	return nil
}

func assignFormValue(destination reflect.Value, values []string) error {
	if destination.Kind() == reflect.Slice {
		result := reflect.MakeSlice(destination.Type(), len(values), len(values))
		for index, raw := range values {
			if err := assignScalar(result.Index(index), raw); err != nil {
				return err
			}
		}
		destination.Set(result)
		return nil
	}
	raw := ""
	if len(values) > 0 {
		raw = values[0]
	}
	return assignScalar(destination, raw)
}

func assignScalar(destination reflect.Value, raw string) error {
	if destination.Kind() == reflect.Pointer {
		if raw == "" {
			return nil
		}
		destination.Set(reflect.New(destination.Type().Elem()))
		return assignScalar(destination.Elem(), raw)
	}
	if destination.CanAddr() && destination.Addr().Type().Implements(textUnmarshalerType) {
		return destination.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(raw))
	}
	switch destination.Kind() {
	case reflect.String:
		destination.SetString(raw)
	case reflect.Bool:
		value, err := parseFormBool(raw)
		if err != nil {
			return err
		}
		destination.SetBool(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		value, err := strconv.ParseInt(raw, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetInt(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		value, err := strconv.ParseUint(raw, 10, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetUint(value)
	case reflect.Float32, reflect.Float64:
		value, err := strconv.ParseFloat(raw, destination.Type().Bits())
		if err != nil {
			return err
		}
		destination.SetFloat(value)
	default:
		return fmt.Errorf("unsupported form field type %s", destination.Type())
	}
	return nil
}

func validateRawFormValue(raw string, exists bool, rules, label string, fieldType reflect.Type) string {
	for fieldType.Kind() == reflect.Pointer {
		fieldType = fieldType.Elem()
	}
	for _, rule := range validationRules(rules) {
		switch rule.name {
		case "required":
			if !exists || strings.TrimSpace(raw) == "" {
				return label + " is required"
			}
		case "email":
			trimmed := strings.TrimSpace(raw)
			at := strings.LastIndexByte(trimmed, '@')
			if trimmed != "" && (at <= 0 || at == len(trimmed)-1 || !strings.Contains(trimmed[at+1:], ".")) {
				return label + " must be a valid email address"
			}
		case "min":
			if fieldType.Kind() == reflect.String && rule.value != "" && raw != "" && len([]rune(raw)) < parsedRuleInt(rule.value) {
				return fmt.Sprintf("%s must contain at least %s characters", label, rule.value)
			}
		case "max":
			if fieldType.Kind() == reflect.String && rule.value != "" && raw != "" && len([]rune(raw)) > parsedRuleInt(rule.value) {
				return fmt.Sprintf("%s must contain at most %s characters", label, rule.value)
			}
		}
	}
	return ""
}

func validateDecodedFormValue(value reflect.Value, rules, label string) string {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return ""
		}
		value = value.Elem()
	}
	if value.Kind() == reflect.String || value.Kind() == reflect.Slice {
		return ""
	}
	for _, rule := range validationRules(rules) {
		if rule.name != "min" && rule.name != "max" || rule.value == "" {
			continue
		}
		limit, err := strconv.ParseFloat(rule.value, 64)
		if err != nil {
			continue
		}
		actual, ok := numericFormValue(value)
		if !ok {
			continue
		}
		if rule.name == "min" && actual < limit {
			return fmt.Sprintf("%s must be at least %s", label, rule.value)
		}
		if rule.name == "max" && actual > limit {
			return fmt.Sprintf("%s must be at most %s", label, rule.value)
		}
	}
	return ""
}

type validationRule struct{ name, value string }

func validationRules(source string) []validationRule {
	if source == "" {
		return nil
	}
	parts := strings.Split(source, ",")
	result := make([]validationRule, 0, len(parts))
	for _, part := range parts {
		name, value, _ := strings.Cut(strings.TrimSpace(part), "=")
		result = append(result, validationRule{name: name, value: value})
	}
	return result
}

func hasValidationRule(source, expected string) bool {
	for _, rule := range validationRules(source) {
		if rule.name == expected {
			return true
		}
	}
	return false
}

func numericFormValue(value reflect.Value) (float64, bool) {
	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int()), true
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint()), true
	case reflect.Float32, reflect.Float64:
		return value.Float(), true
	}
	return 0, false
}

func parseFormBool(raw string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "on", "yes":
		return true, nil
	case "", "0", "false", "off", "no":
		return false, nil
	default:
		return false, fmt.Errorf("invalid boolean %q", raw)
	}
}

func formFieldName(field reflect.StructField) string {
	if tagged := strings.Split(field.Tag.Get("form"), ",")[0]; tagged != "" {
		return tagged
	}
	return snakeCase(field.Name)
}

func fieldLabel(field reflect.StructField, fallback string) string {
	if label := strings.TrimSpace(field.Tag.Get("label")); label != "" {
		return label
	}
	words := strings.Fields(strings.ReplaceAll(fallback, "_", " "))
	for index, word := range words {
		if strings.EqualFold(word, "id") {
			words[index] = "ID"
			continue
		}
		runes := []rune(word)
		if len(runes) > 0 && runes[0] >= 'a' && runes[0] <= 'z' {
			runes[0] -= 'a' - 'A'
		}
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

func snakeCase(value string) string {
	var output strings.Builder
	for index, current := range value {
		if index > 0 && current >= 'A' && current <= 'Z' {
			output.WriteByte('_')
		}
		output.WriteRune(current)
	}
	return strings.ToLower(output.String())
}

func parsedRuleInt(value string) int {
	result, _ := strconv.Atoi(value)
	return result
}
