package web

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

// SocketMessageType identifies a text or binary WebSocket message.
type SocketMessageType = websocket.MessageType

const (
	SocketMessageText   SocketMessageType = websocket.MessageText
	SocketMessageBinary SocketMessageType = websocket.MessageBinary
)

// SocketStatus is an RFC 6455 WebSocket close status.
type SocketStatus = websocket.StatusCode

const (
	SocketStatusNormalClosure   SocketStatus = websocket.StatusNormalClosure
	SocketStatusGoingAway       SocketStatus = websocket.StatusGoingAway
	SocketStatusPolicyViolation SocketStatus = websocket.StatusPolicyViolation
	SocketStatusInternalError   SocketStatus = websocket.StatusInternalError
)

// SocketCompression controls per-message deflate negotiation.
type SocketCompression = websocket.CompressionMode

const (
	SocketCompressionDisabled          SocketCompression = websocket.CompressionDisabled
	SocketCompressionContextTakeover   SocketCompression = websocket.CompressionContextTakeover
	SocketCompressionNoContextTakeover SocketCompression = websocket.CompressionNoContextTakeover
)

// SocketOptions configures a WebSocket endpoint. Same-origin requests are
// accepted by default. Use OriginPatterns to explicitly allow other origins.
type SocketOptions struct {
	Subprotocols         []string
	OriginPatterns       []string
	InsecureSkipVerify   bool
	Compression          SocketCompression
	CompressionThreshold int
	ReadLimit            int64
	ReadTimeout          time.Duration
	WriteTimeout         time.Duration
	MaxConnections       int
	OnPing               func(context.Context, []byte) bool
	OnPong               func(context.Context, []byte)
	OnError              func(*Context, error)
}

// SocketHandler owns a WebSocket connection until it returns. Northframe
// closes and unregisters the connection after the handler finishes.
type SocketHandler func(*Context, *Socket) error

// Socket is Northframe's context-aware WebSocket connection.
type Socket struct {
	connection   *websocket.Conn
	ctx          context.Context
	cancel       context.CancelFunc
	readTimeout  time.Duration
	writeTimeout time.Duration
	cancelOnce   sync.Once
}

// WebSocket registers a GET endpoint that upgrades to WebSocket without a
// reverse proxy or second server. Global and route middleware run before the
// upgrade, so authentication and permission checks work normally.
func (app *App) WebSocket(pattern string, options SocketOptions, handler SocketHandler, middleware ...Middleware) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		panic("northframe: WebSocket pattern cannot be empty")
	}
	if handler == nil {
		panic("northframe: WebSocket handler cannot be nil")
	}
	if !strings.Contains(pattern, " ") {
		pattern = http.MethodGet + " " + pattern
	}

	var capacity chan struct{}
	if options.MaxConnections > 0 {
		capacity = make(chan struct{}, options.MaxConnections)
	}

	app.Handle(pattern, http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if capacity != nil {
			select {
			case capacity <- struct{}{}:
				defer func() { <-capacity }()
			default:
				http.Error(writer, "WebSocket connection limit reached", http.StatusServiceUnavailable)
				return
			}
		}

		connection, err := websocket.Accept(writer, request, &websocket.AcceptOptions{
			Subprotocols:         options.Subprotocols,
			OriginPatterns:       options.OriginPatterns,
			InsecureSkipVerify:   options.InsecureSkipVerify,
			CompressionMode:      options.Compression,
			CompressionThreshold: options.CompressionThreshold,
			OnPingReceived:       options.OnPing,
			OnPongReceived:       options.OnPong,
		})
		if err != nil {
			return
		}
		if options.ReadLimit > 0 {
			connection.SetReadLimit(options.ReadLimit)
		}

		socketContext, cancel := context.WithCancel(context.WithoutCancel(request.Context()))
		socket := &Socket{
			connection:   connection,
			ctx:          socketContext,
			cancel:       cancel,
			readTimeout:  options.ReadTimeout,
			writeTimeout: options.WriteTimeout,
		}
		app.sockets.add(socket)
		defer app.sockets.remove(socket)
		defer socket.CloseNow()

		current := newContext(writer, request)
		if err := runSocketHandler(current, socket, handler); err != nil && SocketCloseStatus(err) == -1 {
			if options.OnError != nil {
				options.OnError(current, err)
			} else {
				log.Printf("northframe: WebSocket %s failed: %v", request.URL.Path, err)
			}
			_ = socket.Close(SocketStatusInternalError, "internal server error")
		}
	}), middleware...)
}

func runSocketHandler(current *Context, socket *Socket, handler SocketHandler) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			err = fmt.Errorf("WebSocket panic: %v", recovered)
		}
	}()
	return handler(current, socket)
}

// Context lives until the socket closes or the application shuts down. It is
// intentionally detached from the HTTP request cancellation that follows an
// upgrade.
func (socket *Socket) Context() context.Context { return socket.ctx }

func (socket *Socket) Subprotocol() string { return socket.connection.Subprotocol() }

func (socket *Socket) Read(ctx context.Context) (SocketMessageType, []byte, error) {
	ctx, cancel := socket.operationContext(ctx, socket.readTimeout)
	defer cancel()
	return socket.connection.Read(ctx)
}

func (socket *Socket) ReadJSON(ctx context.Context, value any) error {
	ctx, cancel := socket.operationContext(ctx, socket.readTimeout)
	defer cancel()
	return wsjson.Read(ctx, socket.connection, value)
}

func (socket *Socket) Write(ctx context.Context, messageType SocketMessageType, payload []byte) error {
	ctx, cancel := socket.operationContext(ctx, socket.writeTimeout)
	defer cancel()
	return socket.connection.Write(ctx, messageType, payload)
}

func (socket *Socket) WriteJSON(ctx context.Context, value any) error {
	ctx, cancel := socket.operationContext(ctx, socket.writeTimeout)
	defer cancel()
	return wsjson.Write(ctx, socket.connection, value)
}

func (socket *Socket) Ping(ctx context.Context) error {
	ctx, cancel := socket.operationContext(ctx, socket.writeTimeout)
	defer cancel()
	return socket.connection.Ping(ctx)
}

func (socket *Socket) Close(status SocketStatus, reason string) error {
	socket.cancelContext()
	return socket.connection.Close(status, reason)
}

func (socket *Socket) CloseNow() error {
	socket.cancelContext()
	return socket.connection.CloseNow()
}

func (socket *Socket) operationContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = socket.ctx
	}
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(socket.ctx, cancel)
	cleanup := func() {
		stop()
		cancel()
	}
	if timeout <= 0 {
		return ctx, cleanup
	}
	ctx, timeoutCancel := context.WithTimeout(ctx, timeout)
	return ctx, func() {
		timeoutCancel()
		cleanup()
	}
}

func (socket *Socket) cancelContext() {
	socket.cancelOnce.Do(socket.cancel)
}

// SocketCloseStatus returns an RFC 6455 status from a connection error, or -1
// when the error is not a WebSocket close frame.
func SocketCloseStatus(err error) SocketStatus {
	if err == nil {
		return -1
	}
	return websocket.CloseStatus(err)
}

// ShutdownWebSockets gracefully closes every active application socket. If
// the context expires, remaining connections are closed immediately.
func (app *App) ShutdownWebSockets(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	sockets := app.sockets.snapshot()
	if len(sockets) == 0 {
		return nil
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		var wait sync.WaitGroup
		wait.Add(len(sockets))
		for _, socket := range sockets {
			go func(socket *Socket) {
				defer wait.Done()
				_ = socket.Close(SocketStatusGoingAway, "server shutting down")
			}(socket)
		}
		wait.Wait()
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		for _, socket := range sockets {
			_ = socket.CloseNow()
		}
		return errors.Join(errors.New("shut down WebSockets"), ctx.Err())
	}
}

type socketRegistry struct {
	mu      sync.Mutex
	sockets map[*Socket]struct{}
}

func newSocketRegistry() *socketRegistry {
	return &socketRegistry{sockets: make(map[*Socket]struct{})}
}

func (registry *socketRegistry) add(socket *Socket) {
	registry.mu.Lock()
	registry.sockets[socket] = struct{}{}
	registry.mu.Unlock()
}

func (registry *socketRegistry) remove(socket *Socket) {
	registry.mu.Lock()
	delete(registry.sockets, socket)
	registry.mu.Unlock()
}

func (registry *socketRegistry) snapshot() []*Socket {
	registry.mu.Lock()
	defer registry.mu.Unlock()
	result := make([]*Socket, 0, len(registry.sockets))
	for socket := range registry.sockets {
		result = append(result, socket)
	}
	return result
}
