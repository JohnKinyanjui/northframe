package main

import (
	"strings"
	"testing"
)

func TestDockerfileBuildsOneBinaryAsNonRoot(t *testing.T) {
	result := dockerfile("1.27", "./cmd/server")
	for _, expected := range []string{"golang:1.27-alpine", "go build -trimpath", "./cmd/server", "USER north", "/api/health", `ENTRYPOINT ["/app/app"]`} {
		if !strings.Contains(result, expected) {
			t.Errorf("Dockerfile missing %q\n%s", expected, result)
		}
	}
}
