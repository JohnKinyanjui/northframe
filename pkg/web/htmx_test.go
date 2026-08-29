package web

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTMXRequestMetadata(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/fleet", nil)
	request.Header.Set("HX-Request", "true")
	request.Header.Set("HX-Boosted", "true")
	request.Header.Set("HX-Current-URL", "https://example.test/operations")
	request.Header.Set("HX-Target", "fleet-panel")
	request.Header.Set("HX-Trigger-Name", "refresh")
	current := newContext(httptest.NewRecorder(), request)
	if !current.HTMX() || !current.HXBoosted() {
		t.Fatal("htmx request flags were not detected")
	}
	if current.HXCurrentURL() != "https://example.test/operations" || current.HXTarget() != "fleet-panel" || current.HXTriggerName() != "refresh" {
		t.Fatalf("unexpected htmx request metadata: %q, %q, %q", current.HXCurrentURL(), current.HXTarget(), current.HXTriggerName())
	}
}

func TestHTMXResponseHelpers(t *testing.T) {
	response := httptest.NewRecorder()
	current := newContext(response, httptest.NewRequest(http.MethodPost, "/jobs", nil))
	if err := current.HXRetarget("#dispatch-board"); err != nil {
		t.Fatal(err)
	}
	if err := current.HXReswap("outerHTML"); err != nil {
		t.Fatal(err)
	}
	if err := current.HXPushURL("/jobs/active"); err != nil {
		t.Fatal(err)
	}
	if err := current.HXTriggerEvents(map[string]any{"job-assigned": map[string]string{"id": "JOB-42"}}); err != nil {
		t.Fatal(err)
	}
	if got := response.Header().Get("HX-Retarget"); got != "#dispatch-board" {
		t.Fatalf("HX-Retarget = %q", got)
	}
	if got := response.Header().Get("HX-Reswap"); got != "outerHTML" {
		t.Fatalf("HX-Reswap = %q", got)
	}
	if got := response.Header().Get("HX-Push-Url"); got != "/jobs/active" {
		t.Fatalf("HX-Push-Url = %q", got)
	}
	if got := response.Header().Get("HX-Trigger"); got != `{"job-assigned":{"id":"JOB-42"}}` {
		t.Fatalf("HX-Trigger = %q", got)
	}
}

func TestHTMXResponseHelpersRequireWriter(t *testing.T) {
	current := newContext(nil, httptest.NewRequest(http.MethodGet, "/", nil))
	if err := current.HXRefresh(); err == nil {
		t.Fatal("HXRefresh accepted a loader context without a response writer")
	}
}
