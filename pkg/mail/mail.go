// Package mail provides a small sender contract, an SMTP adapter, and an
// in-memory outbox suitable for tests and local development.
package mail

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/mail"
	"net/smtp"
	"strings"
	"sync"
)

type Message struct {
	From                string
	To, CC, BCC         []string
	Subject, Text, HTML string
	Headers             map[string]string
}
type Sender interface {
	Send(context.Context, Message) error
}
type SMTPConfig struct{ Address, Host, Username, Password, From string }
type SMTP struct{ config SMTPConfig }

func NewSMTP(config SMTPConfig) (*SMTP, error) {
	if strings.TrimSpace(config.Address) == "" || strings.TrimSpace(config.Host) == "" {
		return nil, errors.New("SMTP address and host are required")
	}
	return &SMTP{config: config}, nil
}
func (sender *SMTP) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if message.From == "" {
		message.From = sender.config.From
	}
	if err := Validate(message); err != nil {
		return err
	}
	recipients := append(append(append([]string{}, message.To...), message.CC...), message.BCC...)
	var auth smtp.Auth
	if sender.config.Username != "" {
		auth = smtp.PlainAuth("", sender.config.Username, sender.config.Password, sender.config.Host)
	}
	return smtp.SendMail(sender.config.Address, auth, message.From, recipients, encode(message))
}
func Validate(message Message) error {
	if unsafeHeader(message.Subject) {
		return errors.New("message subject contains a newline")
	}
	if _, err := mail.ParseAddress(message.From); err != nil {
		return fmt.Errorf("invalid from address: %w", err)
	}
	if len(message.To)+len(message.CC)+len(message.BCC) == 0 {
		return errors.New("at least one recipient is required")
	}
	for _, address := range append(append(append([]string{}, message.To...), message.CC...), message.BCC...) {
		if _, err := mail.ParseAddress(address); err != nil {
			return fmt.Errorf("invalid recipient %q: %w", address, err)
		}
	}
	if strings.TrimSpace(message.Subject) == "" {
		return errors.New("message subject is required")
	}
	if message.Text == "" && message.HTML == "" {
		return errors.New("message body is required")
	}
	for name, value := range message.Headers {
		if unsafeHeader(name) || unsafeHeader(value) || strings.Contains(name, ":") {
			return fmt.Errorf("invalid message header %q", name)
		}
	}
	return nil
}

func unsafeHeader(value string) bool { return strings.ContainsAny(value, "\r\n") }
func encode(message Message) []byte {
	var output bytes.Buffer
	write := func(name, value string) {
		if value != "" {
			fmt.Fprintf(&output, "%s: %s\r\n", name, value)
		}
	}
	write("From", message.From)
	write("To", strings.Join(message.To, ", "))
	write("Cc", strings.Join(message.CC, ", "))
	write("Subject", message.Subject)
	write("MIME-Version", "1.0")
	for name, value := range message.Headers {
		write(name, value)
	}
	if message.HTML != "" {
		write("Content-Type", `text/html; charset="UTF-8"`)
		output.WriteString("\r\n" + message.HTML)
	} else {
		write("Content-Type", `text/plain; charset="UTF-8"`)
		output.WriteString("\r\n" + message.Text)
	}
	return output.Bytes()
}

type Memory struct {
	mu       sync.Mutex
	Messages []Message
}

func (sender *Memory) Send(ctx context.Context, message Message) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := Validate(message); err != nil {
		return err
	}
	sender.mu.Lock()
	sender.Messages = append(sender.Messages, message)
	sender.mu.Unlock()
	return nil
}
func (sender *Memory) Outbox() []Message {
	sender.mu.Lock()
	defer sender.mu.Unlock()
	return append([]Message(nil), sender.Messages...)
}
