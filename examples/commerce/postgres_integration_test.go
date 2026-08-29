package main

import (
	"context"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	generatedRoutes "github.com/JohnKinyanjui/northframe/examples/commerce/.generated/routes"
	commerceDB "github.com/JohnKinyanjui/northframe/examples/commerce/internal/db/generated"
	"github.com/JohnKinyanjui/northframe/examples/commerce/services/commerce"
	"github.com/JohnKinyanjui/northframe/pkg/database"
	"github.com/JohnKinyanjui/northframe/pkg/database/postgres"
	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func TestPostgresAddProductAction(t *testing.T) {
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL is not set")
	}
	handle, err := postgres.Open(databaseURL, database.Pool{MaxOpenConns: 4, MaxIdleConns: 1})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	migrations, err := fs.Sub(migrationFiles, "internal/db/migrations")
	if err != nil {
		t.Fatal(err)
	}
	if err := database.Migrate(ctx, handle, database.PostgreSQL, migrations); err != nil {
		t.Fatal(err)
	}

	tx, err := handle.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback()
	queries := commerceDB.New(tx)
	app := web.New()
	web.Provide[commerce.Store](app, queries)
	generatedRoutes.Register(app)

	getResponse := httptest.NewRecorder()
	app.ServeHTTP(getResponse, httptest.NewRequest(http.MethodGet, "/inventory", nil))
	if getResponse.Code != http.StatusOK {
		t.Fatalf("GET /inventory = %d: %s", getResponse.Code, getResponse.Body.String())
	}
	body := getResponse.Body.String()
	for _, expected := range []string{
		"Inventory", "Hand-thrown serving bowl", `action="/inventory/products"`,
		`class="hidden fixed inset-0`, `type="module"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("GET /inventory body does not contain %q", expected)
		}
	}
	if strings.Contains(body, "<InventoryCard") {
		t.Fatal("independent InventoryCard component leaked as a literal browser tag")
	}
	cookies := getResponse.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("inventory page did not issue a CSRF cookie")
	}

	rejectedRequest := httptest.NewRequest(http.MethodPost, "/inventory/products", strings.NewReader("name=Rejected"))
	rejectedRequest.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rejectedRequest.AddCookie(cookies[0])
	rejectedResponse := httptest.NewRecorder()
	app.ServeHTTP(rejectedResponse, rejectedRequest)
	if rejectedResponse.Code != http.StatusForbidden {
		t.Fatalf("POST without CSRF token = %d, want %d", rejectedResponse.Code, http.StatusForbidden)
	}

	helpResponse := httptest.NewRecorder()
	app.ServeHTTP(helpResponse, httptest.NewRequest(http.MethodGet, "/help?q=svelte", nil))
	helpBody := helpResponse.Body.String()
	if helpResponse.Code != http.StatusOK || !strings.Contains(helpBody, "How is Northframe different from SvelteKit?") || !strings.Contains(helpBody, `data-north-scope="answer-card"`) || strings.Contains(helpBody, "<AnswerCard") {
		t.Fatalf("GET /help?q=svelte = %d: %s", helpResponse.Code, helpResponse.Body.String())
	}
	values := url.Values{
		"sku": {"NF-" + time.Now().Format("150405.000000")}, "name": {"Northframe Test Vessel"},
		"category": {"Testware"}, "stock": {"12"}, "reorder_point": {"4"}, "price": {"149.50"},
		"_northframe_csrf": {cookies[0].Value},
	}
	request := httptest.NewRequest(http.MethodPost, "/inventory/products", strings.NewReader(values.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.AddCookie(cookies[0])
	response := httptest.NewRecorder()
	app.ServeHTTP(response, request)
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/inventory" {
		t.Fatalf("POST /inventory/products = (%d, %q, %q)", response.Code, response.Header().Get("Location"), response.Body.String())
	}
	var count int
	if err := tx.QueryRowContext(ctx, "select count(1) from products where name = $1", "Northframe Test Vessel").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("created product count = %d", count)
	}
}
