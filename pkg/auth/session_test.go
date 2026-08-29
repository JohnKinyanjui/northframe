package auth

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/JohnKinyanjui/northframe/pkg/web"
)

func TestSessionLifecycleUsesOpaqueCookie(t *testing.T) {
	store := NewMemoryStore()
	manager, err := New(Config{Store: store, Lifetime: time.Hour})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	session, err := manager.Start(t.Context(), response, SessionInput{
		Subject: "user-42", Values: map[string]string{"store": "store-7"},
		Permissions: []string{"Orders.Read", "orders.read", "catalog.*"},
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	cookies := response.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("cookies = %d, want 1", len(cookies))
	}
	cookie := cookies[0]
	if !cookie.HttpOnly || cookie.SameSite != http.SameSiteLaxMode {
		t.Fatalf("cookie security = HttpOnly:%v SameSite:%v", cookie.HttpOnly, cookie.SameSite)
	}
	if cookie.Value == session.ID || tokenDigest(cookie.Value) != session.ID {
		t.Fatal("session store received the raw browser token")
	}

	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(cookie)
	loaded, err := manager.Get(t.Context(), request)
	if err != nil {
		t.Fatalf("load session: %v", err)
	}
	if loaded.Subject != "user-42" || loaded.Values["store"] != "store-7" {
		t.Fatalf("unexpected session: %#v", loaded)
	}
	if len(loaded.Permissions) != 2 || !loaded.Can("orders.read") || !loaded.Can("catalog.products.write") {
		t.Fatalf("unexpected permissions: %#v", loaded.Permissions)
	}

	endResponse := httptest.NewRecorder()
	if err := manager.End(t.Context(), endResponse, request); err != nil {
		t.Fatalf("end session: %v", err)
	}
	if _, err := manager.Get(t.Context(), request); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("load ended session error = %v, want ErrSessionNotFound", err)
	}
	if ended := endResponse.Result().Cookies()[0]; ended.MaxAge != -1 {
		t.Fatalf("ended cookie MaxAge = %d, want -1", ended.MaxAge)
	}
}

func TestRequireLoadsSessionAndEnforcesPermissions(t *testing.T) {
	manager, err := New(Config{Store: NewMemoryStore()})
	if err != nil {
		t.Fatal(err)
	}
	started := httptest.NewRecorder()
	if _, err := manager.Start(t.Context(), started, SessionInput{
		Subject: "staff-1", Permissions: []string{"orders.*"},
	}); err != nil {
		t.Fatal(err)
	}
	cookie := started.Result().Cookies()[0]

	app := web.New()
	app.HandleFunc("GET /orders", func(writer http.ResponseWriter, request *http.Request) {
		session := MustCurrent(web.ContextFor(request))
		_, _ = writer.Write([]byte(session.Subject))
	}, Require(manager, GuardOptions{Permissions: []string{"orders.read"}}))
	app.HandleFunc("GET /settings", func(http.ResponseWriter, *http.Request) {}, Require(manager, GuardOptions{Permissions: []string{"settings.write"}}))

	ordersRequest := httptest.NewRequest(http.MethodGet, "/orders", nil)
	ordersRequest.AddCookie(cookie)
	ordersResponse := httptest.NewRecorder()
	app.ServeHTTP(ordersResponse, ordersRequest)
	if ordersResponse.Code != http.StatusOK || strings.TrimSpace(ordersResponse.Body.String()) != "staff-1" {
		t.Fatalf("orders response = (%d, %q)", ordersResponse.Code, ordersResponse.Body.String())
	}

	settingsRequest := httptest.NewRequest(http.MethodGet, "/settings", nil)
	settingsRequest.AddCookie(cookie)
	settingsResponse := httptest.NewRecorder()
	app.ServeHTTP(settingsResponse, settingsRequest)
	if settingsResponse.Code != http.StatusForbidden {
		t.Fatalf("settings status = %d, want %d", settingsResponse.Code, http.StatusForbidden)
	}
}

func TestRequireRedirectsAnonymousBrowser(t *testing.T) {
	manager, err := New(Config{Store: NewMemoryStore()})
	if err != nil {
		t.Fatal(err)
	}
	handler := Require(manager, GuardOptions{LoginPath: "/login"})(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if response.Code != http.StatusSeeOther || response.Header().Get("Location") != "/login" {
		t.Fatalf("response = (%d, %q)", response.Code, response.Header().Get("Location"))
	}
}

func TestExpiredSessionIsRejected(t *testing.T) {
	now := time.Now().UTC()
	store := NewMemoryStore()
	store.clock = func() time.Time { return now }
	manager, err := New(Config{Store: store, Lifetime: time.Minute, Clock: func() time.Time { return now }})
	if err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	if _, err := manager.Start(context.Background(), response, SessionInput{Subject: "user"}); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.AddCookie(response.Result().Cookies()[0])
	now = now.Add(2 * time.Minute)
	if _, err := manager.Get(context.Background(), request); !errors.Is(err, ErrSessionNotFound) {
		t.Fatalf("expired session error = %v, want ErrSessionNotFound", err)
	}
}
