package lsp

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
)

type transport struct {
	reader *bufio.Reader
	writer io.Writer
	mu     sync.Mutex
}

func newTransport(reader io.Reader, writer io.Writer) *transport {
	return &transport{reader: bufio.NewReader(reader), writer: writer}
}

func (current *transport) read() (request, error) {
	contentLength := -1
	for {
		line, err := current.reader.ReadString('\n')
		if err != nil {
			return request{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		name, value, found := strings.Cut(line, ":")
		if found && strings.EqualFold(strings.TrimSpace(name), "Content-Length") {
			contentLength, err = strconv.Atoi(strings.TrimSpace(value))
			if err != nil {
				return request{}, fmt.Errorf("invalid Content-Length: %w", err)
			}
		}
	}
	if contentLength < 0 {
		return request{}, fmt.Errorf("missing Content-Length header")
	}
	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(current.reader, payload); err != nil {
		return request{}, err
	}
	var message request
	if err := json.Unmarshal(payload, &message); err != nil {
		return request{}, fmt.Errorf("decode JSON-RPC message: %w", err)
	}
	return message, nil
}

func (current *transport) write(value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	current.mu.Lock()
	defer current.mu.Unlock()
	_, err = fmt.Fprintf(current.writer, "Content-Length: %d\r\n\r\n%s", len(payload), payload)
	return err
}
