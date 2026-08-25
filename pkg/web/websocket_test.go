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
