package cache

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMemoryCopiesValuesExpiresAndSupportsJSON(t *testing.T) {
	store := NewMemory()
	now := time.Unix(100, 0)
	store.now = func() time.Time { return now }
	input := []byte("safe")
	if err := store.Set(context.Background(), "key", input, time.Minute); err != nil {
		t.Fatal(err)
	}
	input[0] = 'x'
	value, err := store.Get(context.Background(), "key")
	if err != nil || string(value) != "safe" {
		t.Fatalf("value = %q, %v", value, err)
	}
	now = now.Add(time.Minute)
	if _, err := store.Get(context.Background(), "key"); !errors.Is(err, ErrMiss) {
		t.Fatalf("expiry error = %v", err)
	}
	type item struct{ Name string }
	if err := SetJSON(context.Background(), store, "json", item{Name: "North"}, 0); err != nil {
		t.Fatal(err)
	}
	decoded, err := GetJSON[item](context.Background(), store, "json")
	if err != nil || decoded.Name != "North" {
		t.Fatalf("decoded = %#v, %v", decoded, err)
	}
}
