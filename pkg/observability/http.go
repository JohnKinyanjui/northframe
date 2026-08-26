// Package observability provides optional structured request logging and
// W3C-compatible trace correlation without requiring a global logger.
package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"northframe.dev/northframe/pkg/web"
)

type contextKey string

const requestIDKey contextKey = "northframe.request_id"
const traceIDKey contextKey = "northframe.trace_id"

func JSONLogger(level slog.Leveler) *slog.Logger {
	if level == nil {
		level = slog.LevelInfo
	}
	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

func HTTP(logger *slog.Logger) web.Middleware {
	if logger == nil {
		logger = slog.Default()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			started := time.Now()
			requestID := strings.TrimSpace(request.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = randomHex(12)
			}
			traceID := incomingTraceID(request.Header.Get("traceparent"))
			if traceID == "" {
				traceID = randomHex(16)
			}
			spanID := randomHex(8)
			writer.Header().Set("X-Request-ID", requestID)
			writer.Header().Set("traceparent", "00-"+traceID+"-"+spanID+"-01")
			ctx := context.WithValue(request.Context(), requestIDKey, requestID)
			ctx = context.WithValue(ctx, traceIDKey, traceID)
			observed := &responseObserver{ResponseWriter: writer, status: http.StatusOK}
			next.ServeHTTP(observed, request.WithContext(ctx))
			logger.LogAttrs(ctx, slog.LevelInfo, "http.request", slog.String("method", request.Method), slog.String("path", request.URL.Path), slog.Int("status", observed.status), slog.Int64("bytes", observed.bytes), slog.Duration("duration", time.Since(started)), slog.String("request_id", requestID), slog.String("trace_id", traceID))
		})
	}
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDKey).(string)
	return value
}
func TraceID(ctx context.Context) string { value, _ := ctx.Value(traceIDKey).(string); return value }

type responseObserver struct {
	http.ResponseWriter
	status      int
	bytes       int64
	wroteHeader bool
}

func (writer *responseObserver) WriteHeader(status int) {
	if writer.wroteHeader {
		return
	}
	writer.wroteHeader = true
	writer.status = status
	writer.ResponseWriter.WriteHeader(status)
}
func (writer *responseObserver) Write(value []byte) (int, error) {
	if !writer.wroteHeader {
		writer.WriteHeader(http.StatusOK)
	}
	size, err := writer.ResponseWriter.Write(value)
	writer.bytes += int64(size)
	return size, err
}
func (writer *responseObserver) Unwrap() http.ResponseWriter { return writer.ResponseWriter }

func incomingTraceID(value string) string {
	parts := strings.Split(strings.TrimSpace(value), "-")
	if len(parts) != 4 || parts[0] != "00" || len(parts[1]) != 32 || len(parts[2]) != 16 || len(parts[3]) != 2 {
		return ""
	}
	if _, err := hex.DecodeString(parts[1] + parts[2] + parts[3]); err != nil || parts[1] == strings.Repeat("0", 32) || parts[2] == strings.Repeat("0", 16) {
		return ""
	}
	return strings.ToLower(parts[1])
}
func randomHex(size int) string {
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		return strings.Repeat("0", size*2)
	}
	return hex.EncodeToString(value)
}
