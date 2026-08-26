package mail

import (
	"context"
	"strings"
	"testing"
)

func TestMemoryOutboxAndEncoding(t *testing.T) {
	message := Message{From: "team@example.test", To: []string{"user@example.test"}, Subject: "Welcome", Text: "Hello"}
	sender := &Memory{}
	if err := sender.Send(context.Background(), message); err != nil {
		t.Fatal(err)
	}
	if len(sender.Outbox()) != 1 || !strings.Contains(string(encode(message)), "Subject: Welcome") {
		t.Fatalf("outbox = %#v", sender.Outbox())
	}
}
func TestValidateRejectsMissingRecipient(t *testing.T) {
	if err := Validate(Message{From: "team@example.test", Subject: "Hi", Text: "Body"}); err == nil {
		t.Fatal("missing recipient accepted")
	}
}

func TestValidateRejectsHeaderInjection(t *testing.T) {
	err := Validate(Message{From: "team@example.test", To: []string{"user@example.test"}, Subject: "Hello\r\nBcc: attacker@example.test", Text: "Body"})
	if err == nil {
		t.Fatal("header injection was accepted")
	}
}
