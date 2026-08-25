package admin

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"

	"northframe.dev/northframe/pkg/auth"
	"northframe.dev/northframe/pkg/web"
)

type memoryRepository struct {
	mu      sync.Mutex
	records []Record
}

func (repository *memoryRepository) List(context.Context, ListQuery) (Page, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	records := make([]Record, len(repository.records))
	copy(records, repository.records)
	return Page{Records: records, Total: int64(len(records))}, nil
}

func (repository *memoryRepository) Get(_ context.Context, id string) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	for _, record := range repository.records {
		if recordID(record) == id {
			return record, nil
		}
	}
	return nil, web.NotFound("record not found")
}

func (repository *memoryRepository) Create(_ context.Context, record Record) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record["id"] = len(repository.records) + 1
	repository.records = append(repository.records, record)
	return record, nil
}

func (repository *memoryRepository) Update(_ context.Context, id string, record Record) (Record, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()
	record["id"] = id
	return record, nil
}

func (repository *memoryRepository) Delete(context.Context, string) error { return nil }

func TestMountedAdminRendersAndCreatesThroughRepository(t *testing.T) {
	app := web.New()
	sessions, err := auth.New(auth.Config{Store: auth.NewMemoryStore()})
	if err != nil {
		t.Fatal(err)
	}
	repository := &memoryRepository{records: []Record{{"id": 1, "name": "Serving bowl", "active": true}}}
	registry := NewRegistry()
	registry.MustRegister(Resource{
		Name: "products", Label: "Product", Description: "Store catalogue",
		Repository: repository,
		Fields: []Field{
			{Name: "id", ReadOnly: true},
			{Name: "name", Required: true, Searchable: true},
			{Name: "price", Kind: FieldMoney, Required: true},
			{Name: "active", Kind: FieldBoolean},
			{Name: "status", Kind: FieldSelect, Required: true, Options: []Option{{Value: "draft", Label: "Draft"}, {Value: "active", Label: "Active"}}},
		},
		Validate: func(_ context.Context, record Record) web.FieldErrors {
			if record["name"] == "Reserved" {
				return web.FieldErrors{"name": "This product name is reserved"}
			}
			return nil
		},
	})
	if err := Mount(app, registry, sessions, Options{BasePath: "/staff", LoginPath: "/login", Title: "TopDuka Admin"}); err != nil {
		t.Fatal(err)
	}

	started := httptest.NewRecorder()
	if _, err := sessions.Start(t.Context(), started, auth.SessionInput{
		Subject: "staff-1", Permissions: []string{"admin.products.*"},
	}); err != nil {
		t.Fatal(err)
	}
	sessionCookie := started.Result().Cookies()[0]

	dashboardRequest := httptest.NewRequest(http.MethodGet, "/staff/", nil)
	dashboardRequest.AddCookie(sessionCookie)
	dashboardResponse := httptest.NewRecorder()
	app.ServeHTTP(dashboardResponse, dashboardRequest)
	if dashboardResponse.Code != http.StatusOK || !strings.Contains(dashboardResponse.Body.String(), "TopDuka Admin") || !strings.Contains(dashboardResponse.Body.String(), "Products") {
		t.Fatalf("dashboard = (%d, %q)", dashboardResponse.Code, dashboardResponse.Body.String())
	}
	csrfCookie := cookieNamed(dashboardResponse.Result().Cookies(), csrfCookieForTest)
	if csrfCookie == nil {
		t.Fatal("admin dashboard did not issue a CSRF cookie")
	}

	listRequest := httptest.NewRequest(http.MethodGet, "/staff/products", nil)
	listRequest.AddCookie(sessionCookie)
	listRequest.AddCookie(csrfCookie)
	listResponse := httptest.NewRecorder()
	app.ServeHTTP(listResponse, listRequest)
	if listResponse.Code != http.StatusOK || !strings.Contains(listResponse.Body.String(), "Serving bowl") {
		t.Fatalf("list = (%d, %q)", listResponse.Code, listResponse.Body.String())
	}

	form := url.Values{
		"_northframe_csrf": {csrfCookie.Value}, "name": {"Tea cup"}, "price": {"12.50"}, "active": {"on"}, "status": {"active"},
	}
	createRequest := httptest.NewRequest(http.MethodPost, "/staff/products", strings.NewReader(form.Encode()))
	createRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	createRequest.AddCookie(sessionCookie)
	createRequest.AddCookie(csrfCookie)
	createResponse := httptest.NewRecorder()
	app.ServeHTTP(createResponse, createRequest)
	if createResponse.Code != http.StatusSeeOther || createResponse.Header().Get("Location") != "/staff/products" {
		t.Fatalf("create = (%d, %q, %q)", createResponse.Code, createResponse.Header().Get("Location"), createResponse.Body.String())
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	if len(repository.records) != 2 || repository.records[1]["name"] != "Tea cup" || repository.records[1]["price"] != 12.5 {
		t.Fatalf("created records = %#v", repository.records)
	}
}

func TestAdminFormRejectsUnknownSelectOptionAndCustomValidation(t *testing.T) {
	resource := normalizeResource(Resource{
		Name: "products", Repository: &memoryRepository{},
		Fields: []Field{
			{Name: "name", Required: true},
			{Name: "status", Kind: FieldSelect, Options: []Option{{Value: "active"}}},
		},
		Validate: func(_ context.Context, record Record) web.FieldErrors {
			if record["name"] == "Reserved" {
				return web.FieldErrors{"name": "This product name is reserved"}
			}
			return nil
		},
	})
	form := url.Values{"name": {"Reserved"}, "status": {"deleted"}}
	request := httptest.NewRequest(http.MethodPost, "/admin/products", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	_, fields, err := decodeRecord(request, resource)
	if err != nil {
		t.Fatal(err)
	}
	if fields["name"] != "This product name is reserved" || fields["status"] != "Status has an invalid value" {
		t.Fatalf("field errors = %#v", fields)
	}
}

func TestMountedAdminRejectsMissingResourcePermission(t *testing.T) {
	app := web.New()
	sessions, _ := auth.New(auth.Config{Store: auth.NewMemoryStore()})
	registry := NewRegistry()
	registry.MustRegister(Resource{Name: "orders", Repository: &memoryRepository{}, Fields: []Field{{Name: "id"}}})
	if err := Mount(app, registry, sessions, Options{}); err != nil {
		t.Fatal(err)
	}
	started := httptest.NewRecorder()
	_, _ = sessions.Start(t.Context(), started, auth.SessionInput{Subject: "staff", Permissions: []string{"admin.products.*"}})
	request := httptest.NewRequest(http.MethodGet, "/admin/orders", nil)
	request.AddCookie(started.Result().Cookies()[0])
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusForbidden)
	}
}

func TestMountWithAccessUsesApplicationOwnedAuthentication(t *testing.T) {
	app := web.New()
	registry := NewRegistry()
	registry.MustRegister(Resource{Name: "products", Repository: &memoryRepository{}, Fields: []Field{{Name: "id"}}})
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			web.SetLocal(request, "application.account", "owner-1")
			next.ServeHTTP(writer, request)
		})
	}
	resolver := func(ctx *web.Context) (auth.Session, bool) {
		subject, ok := ctx.Locals["application.account"].(string)
		return auth.Session{Subject: subject, Permissions: []string{"admin.*"}}, ok
	}
	if err := MountWithAccess(app, registry, Access{
		Middleware: []web.Middleware{middleware}, Session: resolver,
	}, Options{}); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Products") {
		t.Fatalf("dashboard = (%d, %q)", response.Code, response.Body.String())
	}
}

func TestMountWithAccessRejectsMissingResolverSession(t *testing.T) {
	app := web.New()
	registry := NewRegistry()
	if err := MountWithAccess(app, registry, Access{
		Middleware: []web.Middleware{func(next http.Handler) http.Handler { return next }},
		Session:    func(*web.Context) (auth.Session, bool) { return auth.Session{}, false },
	}, Options{}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/admin/", nil)
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

const csrfCookieForTest = "northframe_csrf"

func cookieNamed(cookies []*http.Cookie, name string) *http.Cookie {
	for _, cookie := range cookies {
		if cookie.Name == name {
			return cookie
		}
	}
	return nil
}
