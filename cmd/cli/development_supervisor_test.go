package main

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"
	"time"

	"github.com/coder/websocket"
)

func TestDevelopmentSupervisorSwitchesBackendsWithoutRestartingPublicListener(t *testing.T) {
	first := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "first")
	}))
	defer first.Close()
	second := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		_, _ = io.WriteString(writer, "second")
	}))
	defer second.Close()

	supervisor, err := newDevelopmentSupervisor(0)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	firstURL, _ := url.Parse(first.URL)
	if err := supervisor.Switch(firstURL); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	publicURL := supervisorTestURL(supervisor)
	if body := supervisorResponse(t, publicURL); body != "first" {
		t.Fatalf("first response = %q", body)
	}

	secondURL, _ := url.Parse(second.URL)
	if err := supervisor.Switch(secondURL); err != nil {
		t.Fatal(err)
	}
	if body := supervisorResponse(t, publicURL); body != "second" {
		t.Fatalf("second response = %q", body)
	}
}

func TestDevelopmentSupervisorOwnsReloadSocketAcrossBackendSwitches(t *testing.T) {
	backend := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusNoContent)
	}))
	defer backend.Close()
	supervisor, err := newDevelopmentSupervisor(0)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	backendURL, _ := url.Parse(backend.URL)
	if err := supervisor.Switch(backendURL); err != nil {
		t.Fatal(err)
	}
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	socketURL := "ws" + supervisorTestURL(supervisor)[4:] + developmentSocketPath
	connection, _, err := websocket.Dial(ctx, socketURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer connection.CloseNow()

	reloaded := make(chan struct{})
	go func() {
		supervisor.ReloadBrowsers()
		close(reloaded)
	}()
	_, _, readErr := connection.Read(ctx)
	if status := websocket.CloseStatus(readErr); status != websocket.StatusGoingAway {
		t.Fatalf("reload close status = %d, want %d (error %v)", status, websocket.StatusGoingAway, readErr)
	}
	select {
	case <-reloaded:
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
}

func TestDevelopmentSupervisorProbeDoesNotDependOnApplicationBackend(t *testing.T) {
	supervisor, err := newDevelopmentSupervisor(0)
	if err != nil {
		t.Fatal(err)
	}
	defer supervisor.Close()
	if err := supervisor.Start(); err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodHead, supervisorTestURL(supervisor)+developmentProbePath, nil)
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusNoContent {
		t.Fatalf("probe status = %d", response.StatusCode)
	}
	if response.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("probe cache control = %q", response.Header.Get("Cache-Control"))
	}
}

func supervisorTestURL(supervisor *developmentSupervisor) string {
	port := supervisor.listener.Addr().(*net.TCPAddr).Port
	return "http://127.0.0.1:" + strconv.Itoa(port)
}

func supervisorResponse(t *testing.T, address string) string {
	t.Helper()
	response, err := http.Get(address)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}
