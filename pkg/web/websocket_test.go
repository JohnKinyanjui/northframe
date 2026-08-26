package web

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

func TestWebSocketJSONRoundTrip(t *testing.T) {
	app := New()
	app.WebSocket("/ws", SocketOptions{ReadLimit: 1024}, func(_ *Context, socket *Socket) error {
		var request struct {
			Message string `json:"message"`
		}
		if err := socket.ReadJSON(socket.Context(), &request); err != nil {
			return err
		}
		return socket.WriteJSON(socket.Context(), map[string]string{"message": strings.ToUpper(request.Message)})
	})

	server := httptest.NewServer(app)
	defer server.Close()
	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()

	if err := wsjson.Write(t.Context(), connection, map[string]string{"message": "north"}); err != nil {
		t.Fatalf("write JSON: %v", err)
	}
	var response map[string]string
	if err := wsjson.Read(t.Context(), connection, &response); err != nil {
		t.Fatalf("read JSON: %v", err)
	}
	if response["message"] != "NORTH" {
		t.Fatalf("message = %q, want NORTH", response["message"])
	}
}

func TestWebSocketContextUsesConnectionLifetimeAndRetainsDependencies(t *testing.T) {
	type service struct{ name string }
	app := New()
	Provide(app, service{name: "realtime"})
	app.WebSocket("GET /ws/{room}", SocketOptions{}, func(ctx *Context, socket *Socket) error {
		if ctx.Response != nil {
			t.Fatal("WebSocket context must not expose the upgraded HTTP response writer")
		}
		if ctx.Param("room") != "north" {
			t.Fatalf("room = %q, want north", ctx.Param("room"))
		}
		dependency := MustUse[service](ctx)
		if dependency.name != "realtime" {
			t.Fatalf("dependency = %#v", dependency)
		}
		if ctx.StdContext() != socket.Context() {
			t.Fatal("Context.StdContext does not use the socket lifetime")
		}
		return socket.WriteJSON(ctx.StdContext(), map[string]string{"room": ctx.Param("room")})
	})

	server := httptest.NewServer(app)
	defer server.Close()
	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws/north"), nil)
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()
	var response map[string]string
	if err := wsjson.Read(t.Context(), connection, &response); err != nil {
		t.Fatalf("read JSON: %v", err)
	}
	if response["room"] != "north" {
		t.Fatalf("room = %q, want north", response["room"])
	}
}

func TestWebSocketRejectsNonGETPattern(t *testing.T) {
	defer func() {
		if recovered := recover(); recovered == nil || !strings.Contains(recovered.(string), "must use GET") {
			t.Fatalf("panic = %v, want actionable GET error", recovered)
		}
	}()
	New().WebSocket("POST /ws", SocketOptions{}, func(*Context, *Socket) error { return nil })
}

func TestWebSocketRejectsUnknownOrigin(t *testing.T) {
	app := New()
	app.WebSocket("/ws", SocketOptions{}, func(_ *Context, _ *Socket) error { return nil })
	server := httptest.NewServer(app)
	defer server.Close()

	_, response, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), &websocket.DialOptions{
		HTTPHeader: http.Header{"Origin": []string{"https://untrusted.example"}},
	})
	if err == nil {
		t.Fatal("dial with an unknown origin succeeded")
	}
	if response == nil || response.StatusCode != http.StatusForbidden {
		if response == nil {
			t.Fatalf("response = nil, want status %d", http.StatusForbidden)
		}
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusForbidden)
	}
}

func TestWebSocketMiddlewareRunsBeforeUpgrade(t *testing.T) {
	app := New()
	handlerCalled := make(chan struct{}, 1)
	requireSession := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			http.Error(writer, "sign in required", http.StatusUnauthorized)
		})
	}
	app.WebSocket("/ws", SocketOptions{}, func(_ *Context, _ *Socket) error {
		handlerCalled <- struct{}{}
		return nil
	}, requireSession)
	server := httptest.NewServer(app)
	defer server.Close()

	_, response, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err == nil {
		t.Fatal("unauthenticated WebSocket upgrade succeeded")
	}
	if response == nil || response.StatusCode != http.StatusUnauthorized {
		t.Fatalf("response = %#v, want status %d", response, http.StatusUnauthorized)
	}
	select {
	case <-handlerCalled:
		t.Fatal("WebSocket handler ran before middleware accepted the request")
	default:
	}
}

func TestWebSocketNegotiatesSubprotocol(t *testing.T) {
	app := New()
	app.WebSocket("/ws", SocketOptions{Subprotocols: []string{"northframe.v1"}}, func(_ *Context, socket *Socket) error {
		return socket.WriteJSON(socket.Context(), map[string]string{"protocol": socket.Subprotocol()})
	})
	server := httptest.NewServer(app)
	defer server.Close()

	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), &websocket.DialOptions{Subprotocols: []string{"northframe.v1"}})
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()
	if connection.Subprotocol() != "northframe.v1" {
		t.Fatalf("subprotocol = %q, want northframe.v1", connection.Subprotocol())
	}
}

func TestWebSocketReadLimitClosesOversizedMessage(t *testing.T) {
	app := New()
	app.WebSocket("/ws", SocketOptions{ReadLimit: 16}, func(_ *Context, socket *Socket) error {
		_, _, err := socket.Read(socket.Context())
		return err
	})
	server := httptest.NewServer(app)
	defer server.Close()

	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()
	if err := connection.Write(t.Context(), websocket.MessageText, []byte(strings.Repeat("x", 64))); err != nil {
		t.Fatalf("write oversized message: %v", err)
	}
	_, _, err = connection.Read(t.Context())
	if status := websocket.CloseStatus(err); status != websocket.StatusMessageTooBig {
		t.Fatalf("close status = %d, want %d (error: %v)", status, websocket.StatusMessageTooBig, err)
	}
}

func TestWebSocketPanicIsContained(t *testing.T) {
	app := New()
	reported := make(chan error, 1)
	app.WebSocket("/ws", SocketOptions{OnError: func(_ *Context, err error) { reported <- err }}, func(_ *Context, _ *Socket) error {
		panic("broken handler")
	})
	server := httptest.NewServer(app)
	defer server.Close()

	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()
	_, _, err = connection.Read(t.Context())
	if status := websocket.CloseStatus(err); status != websocket.StatusInternalError {
		t.Fatalf("close status = %d, want %d (error: %v)", status, websocket.StatusInternalError, err)
	}
	select {
	case err := <-reported:
		if !strings.Contains(err.Error(), "broken handler") {
			t.Fatalf("reported error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("panic was not reported")
	}
}

func TestWebSocketConnectionLimit(t *testing.T) {
	app := New()
	connected := make(chan struct{})
	app.WebSocket("/ws", SocketOptions{MaxConnections: 1}, func(_ *Context, socket *Socket) error {
		close(connected)
		<-socket.Context().Done()
		return nil
	})
	server := httptest.NewServer(app)
	defer server.Close()

	first, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial first WebSocket: %v", err)
	}
	defer first.CloseNow()
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("first WebSocket handler did not start")
	}

	_, response, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err == nil {
		t.Fatal("second WebSocket exceeded the connection limit")
	}
	if response == nil || response.StatusCode != http.StatusServiceUnavailable {
		if response == nil {
			t.Fatalf("response = nil, want status %d", http.StatusServiceUnavailable)
		}
		t.Fatalf("status = %d, want %d", response.StatusCode, http.StatusServiceUnavailable)
	}
}

func TestShutdownWebSocketsSendsGoingAway(t *testing.T) {
	app := New()
	connected := make(chan struct{})
	app.WebSocket("/ws", SocketOptions{}, func(_ *Context, socket *Socket) error {
		close(connected)
		<-socket.Context().Done()
		return nil
	})
	server := httptest.NewServer(app)
	defer server.Close()

	connection, _, err := websocket.Dial(t.Context(), websocketTestURL(server.URL, "/ws"), nil)
	if err != nil {
		t.Fatalf("dial WebSocket: %v", err)
	}
	defer connection.CloseNow()
	select {
	case <-connected:
	case <-time.After(time.Second):
		t.Fatal("WebSocket handler did not start")
	}

	readResult := make(chan error, 1)
	go func() {
		_, _, err := connection.Read(context.Background())
		readResult <- err
	}()

	shutdownContext, cancel := context.WithTimeout(t.Context(), time.Second)
	defer cancel()
	if err := app.ShutdownWebSockets(shutdownContext); err != nil {
		t.Fatalf("shutdown WebSockets: %v", err)
	}
	select {
	case err := <-readResult:
		if status := websocket.CloseStatus(err); status != websocket.StatusGoingAway {
			t.Fatalf("close status = %d, want %d (error: %v)", status, websocket.StatusGoingAway, err)
		}
	case <-time.After(time.Second):
		t.Fatal("client did not receive the shutdown close frame")
	}
}

func websocketTestURL(serverURL, path string) string {
	return "ws" + strings.TrimPrefix(serverURL, "http") + path
}
