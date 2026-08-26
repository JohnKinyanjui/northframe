package observability

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPAddsCorrelationAndStructuredLog(t *testing.T) {
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := HTTP(logger)(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if RequestID(request.Context()) == "" || TraceID(request.Context()) == "" {
			t.Fatal("correlation missing from context")
		}
		writer.WriteHeader(http.StatusCreated)
		_, _ = writer.Write([]byte("ok"))
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/orders", nil))
	if recorder.Header().Get("X-Request-ID") == "" || !strings.HasPrefix(recorder.Header().Get("traceparent"), "00-") {
		t.Fatalf("headers = %#v", recorder.Header())
	}
	if !strings.Contains(output.String(), `"status":201`) || !strings.Contains(output.String(), `"path":"/orders"`) {
		t.Fatalf("log = %s", output.String())
	}
}

func TestCorrelationHelpersAreEmptyWithoutMiddleware(t *testing.T) {
	if RequestID(context.Background()) != "" || TraceID(context.Background()) != "" {
		t.Fatal("unexpected correlation")
	}
}
