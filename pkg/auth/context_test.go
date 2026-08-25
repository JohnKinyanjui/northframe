package auth

import (
	"context"
	"testing"
)

func TestSessionStandardContextIsCloned(t *testing.T) {
	original := Session{Subject: "owner-1", Values: map[string]string{"store": "one"}, Permissions: []string{"admin.*"}}
	ctx := WithSession(context.Background(), original)
	original.Values["store"] = "changed"
	original.Permissions[0] = "changed"

	loaded, ok := SessionFromContext(ctx)
	if !ok || loaded.Subject != "owner-1" || loaded.Values["store"] != "one" || loaded.Permissions[0] != "admin.*" {
		t.Fatalf("loaded session = %#v, %v", loaded, ok)
	}
	loaded.Values["store"] = "mutated"
	reloaded, _ := SessionFromContext(ctx)
	if reloaded.Values["store"] != "one" {
		t.Fatalf("context session was mutated: %#v", reloaded)
	}
}
