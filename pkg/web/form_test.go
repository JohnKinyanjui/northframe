package web

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
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

func formRequest(values url.Values) *http.Request {
	request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	return request
}
