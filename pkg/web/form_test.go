package web

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type typedFormInput struct {
	Name     string   `form:"name" label:"Product name" validate:"required,min=3"`
	Stock    int32    `form:"stock" validate:"required,min=0,max=100"`
	Price    float64  `form:"price" validate:"required,min=0"`
	Featured bool     `form:"featured"`
	Tags     []string `form:"tag"`
	Optional int      `form:"optional"`
}

type uploadFormInput struct {
	Title string                `form:"title" validate:"required"`
	Image *multipart.FileHeader `form:"image" label:"Product image" validate:"required,maxbytes=64" accept:"image/*"`
}

func TestDecodeFormProducesTypedInput(t *testing.T) {
	request := formRequest(url.Values{
		"name": {"Serving bowl"}, "stock": {"12"}, "price": {"49.95"},
		"featured": {"on"}, "tag": {"ceramic", "table"}, "optional": {""},
	})
	input, fields, err := DecodeForm[typedFormInput](request)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Fatalf("fields = %#v", fields)
	}
	if input.Name != "Serving bowl" || input.Stock != 12 || input.Price != 49.95 || !input.Featured {
		t.Fatalf("input = %#v", input)
	}
	if len(input.Tags) != 2 || input.Tags[1] != "table" {
		t.Fatalf("tags = %#v", input.Tags)
	}
}

func TestDecodeFormReturnsFieldErrors(t *testing.T) {
	request := formRequest(url.Values{"name": {"x"}, "stock": {"-1"}, "price": {"not-money"}})
	_, fields, err := DecodeForm[typedFormInput](request)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"name", "stock", "price"} {
		if fields[name] == "" {
			t.Errorf("missing error for %s in %#v", name, fields)
		}
	}
}

func TestPostFormReturnsStructuredEnhancedValidation(t *testing.T) {
	called := false
	app := New()
	app.HandleAction("/products", PostForm("", func(_ *Context, input typedFormInput) (ActionResult, error) {
		called = true
		return ActionSuccess("created", input), nil
	}))
	request := formRequest(url.Values{"name": {""}, "stock": {"0"}, "price": {"1"}})
	request.URL.Path = "/products"
	request.Header.Set(enhanceHeader, "true")
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)

	if called {
		t.Fatal("handler was called for invalid form")
	}
	if response.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d", response.Code)
	}
	var result ActionResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Success || result.Errors["name"] == "" {
		t.Fatalf("result = %#v", result)
	}
}

func TestPostFormReturnsTypedSuccessAndRedirect(t *testing.T) {
	app := New()
	app.HandleAction("/products", PostForm("", func(_ *Context, input typedFormInput) (ActionResult, error) {
		if input.Stock != 7 {
			t.Fatalf("stock = %d", input.Stock)
		}
		return ActionRedirect("/inventory", http.StatusSeeOther), nil
	}))
	request := formRequest(url.Values{"name": {"Bowl"}, "stock": {"7"}, "price": {"2.50"}})
	request.URL.Path = "/products"
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/inventory" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Header().Get("Location"))
	}
}

func TestPostFormRejectsNonStructInputAtRegistration(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("PostForm did not reject a non-struct input")
		}
	}()
	_ = PostForm("/invalid", func(_ *Context, _ string) (ActionResult, error) {
		return ActionResult{}, nil
	})
}

func TestDecodeFormValidatesAndSavesTypedUpload(t *testing.T) {
	png := []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n', 0, 0, 0, 0}
	request := multipartRequest(t, map[string]string{"title": "Serving bowl"}, "image", "../bowl.png", png)
	input, fields, err := DecodeForm[uploadFormInput](request)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 || input.Image == nil {
		t.Fatalf("input = %#v, fields = %#v", input, fields)
	}
	if filename := UploadedFilename(input.Image); filename != "bowl.png" {
		t.Fatalf("filename = %q, want bowl.png", filename)
	}
	destination := filepath.Join(t.TempDir(), "stored.png")
	if err := SaveUploadedFile(input.Image, destination, 64); err != nil {
		t.Fatalf("save upload: %v", err)
	}
	stored, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(stored, png) {
		t.Fatalf("stored upload = %v, want %v", stored, png)
	}
}

func TestDecodeFormRejectsUploadTypeAndSize(t *testing.T) {
	request := multipartRequest(t, map[string]string{"title": "Notes"}, "image", "notes.txt", []byte(strings.Repeat("x", 80)))
	_, fields, err := DecodeForm[uploadFormInput](request)
	if err != nil {
		t.Fatal(err)
	}
	if fields["image"] == "" {
		t.Fatalf("missing image validation error: %#v", fields)
	}
}

func formRequest(values url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}

func multipartRequest(t *testing.T, values map[string]string, field, filename string, contents []byte) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for name, value := range values {
		if err := writer.WriteField(name, value); err != nil {
			t.Fatal(err)
		}
	}
	part, err := writer.CreateFormFile(field, filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(contents); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}
