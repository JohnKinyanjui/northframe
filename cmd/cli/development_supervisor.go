package main

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
)

const (
	developmentProbePath  = "/_northframe/dev"
	developmentSocketPath = "/_northframe/dev/reload"
)

type developmentSupervisor struct {
	port      int
	server    *http.Server
	listener  net.Listener
	backend   atomic.Pointer[url.URL]
	socketsMu sync.Mutex
	sockets   map[*websocket.Conn]struct{}
	started   bool
	startedMu sync.Mutex
}

func newDevelopmentSupervisor(port int) (*developmentSupervisor, error) {
	listener, err := net.Listen("tcp", net.JoinHostPort("", strconv.Itoa(port)))
	if err != nil {
		return nil, fmt.Errorf("development server port %d: %w", port, err)
	}
	supervisor := &developmentSupervisor{port: port, listener: listener, sockets: map[*websocket.Conn]struct{}{}}
	proxy := &httputil.ReverseProxy{
		Rewrite: func(request *httputil.ProxyRequest) {
			if backend := supervisor.backend.Load(); backend != nil {
				request.SetURL(backend)
				request.Out.Host = request.In.Host
			}
		},
		ErrorHandler: func(writer http.ResponseWriter, _ *http.Request, _ error) {
			http.Error(writer, "Northframe is rebuilding the application", http.StatusServiceUnavailable)
		},
	}
	mux := http.NewServeMux()
	mux.HandleFunc("HEAD "+developmentProbePath, func(writer http.ResponseWriter, _ *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		writer.WriteHeader(http.StatusNoContent)
	})
	mux.HandleFunc("GET "+developmentSocketPath, supervisor.acceptReloadSocket)
	mux.Handle("/", http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if supervisor.backend.Load() == nil {
			http.Error(writer, "Northframe is starting the application", http.StatusServiceUnavailable)
			return
		}
		proxy.ServeHTTP(writer, request)
	}))
	supervisor.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	return supervisor, nil
}

func (supervisor *developmentSupervisor) Start() error {
	supervisor.startedMu.Lock()
	defer supervisor.startedMu.Unlock()
	if supervisor.started {
		return nil
	}
	supervisor.started = true
	go func() {
		if err := supervisor.server.Serve(supervisor.listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			fmt.Fprintln(os.Stderr, "north: development supervisor:", err)
		}
	}()
	return nil
}

func (supervisor *developmentSupervisor) Switch(backend *url.URL) error {
	if backend == nil || backend.Scheme == "" || backend.Host == "" {
		return errors.New("development backend URL is invalid")
	}
	copy := *backend
	supervisor.backend.Store(&copy)
	return nil
}

func (supervisor *developmentSupervisor) acceptReloadSocket(writer http.ResponseWriter, request *http.Request) {
	connection, err := websocket.Accept(writer, request, nil)
	if err != nil {
		return
	}
	supervisor.socketsMu.Lock()
	supervisor.sockets[connection] = struct{}{}
	supervisor.socketsMu.Unlock()
	defer func() {
		supervisor.socketsMu.Lock()
		delete(supervisor.sockets, connection)
		supervisor.socketsMu.Unlock()
		_ = connection.CloseNow()
	}()
	for {
		if _, _, err := connection.Read(request.Context()); err != nil {
			return
		}
	}
}

func (supervisor *developmentSupervisor) ReloadBrowsers() {
	supervisor.socketsMu.Lock()
	sockets := make([]*websocket.Conn, 0, len(supervisor.sockets))
	for connection := range supervisor.sockets {
		sockets = append(sockets, connection)
	}
	supervisor.socketsMu.Unlock()
	for _, connection := range sockets {
		_ = connection.Close(websocket.StatusGoingAway, "application updated")
	}
}

func (supervisor *developmentSupervisor) Close() {
	supervisor.ReloadBrowsers()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_ = supervisor.server.Shutdown(ctx)
	_ = supervisor.listener.Close()
}

type developmentChild struct {
	command *exec.Cmd
	wait    chan error
	url     *url.URL
	exited  atomic.Bool
}

func startDevelopmentChild(binary string) (*developmentChild, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	port := listener.Addr().(*net.TCPAddr).Port
	if err := listener.Close(); err != nil {
		return nil, err
	}
	command := exec.Command(binary)
	configureChildProcess(command)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	projectEnvironment, err := loadDotEnv(".env", os.Environ())
	if err != nil {
		return nil, err
	}
	command.Env = setEnvironment(projectEnvironment, "PORT", strconv.Itoa(port))
	command.Env = setEnvironment(command.Env, "NORTHFRAME_ENV", "development-child")
	if err := command.Start(); err != nil {
		return nil, err
	}
	wait := make(chan error, 1)
	child := &developmentChild{command: command, wait: wait, url: &url.URL{Scheme: "http", Host: net.JoinHostPort("127.0.0.1", strconv.Itoa(port))}}
	go func() {
		err := command.Wait()
		child.exited.Store(true)
		wait <- err
	}()
	return child, nil
}

func (child *developmentChild) URL() *url.URL { return child.url }

func (child *developmentChild) WaitReady() error {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-child.wait:
			if err == nil {
				return errors.New("development application stopped before becoming ready")
			}
			return fmt.Errorf("development application stopped before becoming ready: %w", err)
		default:
		}
		connection, err := net.DialTimeout("tcp", child.url.Host, 100*time.Millisecond)
		if err == nil {
			_ = connection.Close()
			return nil
		}
		time.Sleep(40 * time.Millisecond)
	}
	return errors.New("development application did not become ready within 10 seconds")
}

func (child *developmentChild) Stop() {
	if child == nil || child.command == nil || child.exited.Load() {
		return
	}
	stopProcess(child.command, child.wait)
}

func waitForDevelopmentChange(root, generated, baseline string, child *developmentChild, interrupts <-chan os.Signal) (bool, error) {
	ticker := time.NewTicker(450 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-interrupts:
			return false, errInterrupted
		case processErr := <-child.wait:
			if processErr == nil || processWasInterrupted(processErr) {
				return false, errInterrupted
			}
			return false, fmt.Errorf("development application stopped: %w", processErr)
		case <-ticker.C:
			current, err := watchSignature(root, generated)
			if err != nil {
				return false, err
			}
			if current != baseline {
				return true, nil
			}
		}
	}
}
