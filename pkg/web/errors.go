package web

import (
	"errors"
	"log"
	"net/http"
	"strings"
)

// HTTPError carries a safe public message and HTTP response status.
type HTTPError struct {
	Status  int
	Message string
	Err     error
}

// RedirectError stops a loader and asks the router to send an HTTP redirect.
type RedirectError struct {
	Path   string
	Status int
}

func (err *RedirectError) Error() string { return "redirect to " + err.Path }

func (err *HTTPError) Error() string {
	if err.Err != nil {
		return err.Err.Error()
	}
	return err.Message
}

func (err *HTTPError) Unwrap() error { return err.Err }

func Error(status int, message string, err error) error {
	return &HTTPError{Status: status, Message: message, Err: err}
}

func NotFound(message string) error {
	return Error(http.StatusNotFound, message, nil)
}

func BadRequest(message string, err error) error {
	return Error(http.StatusBadRequest, message, err)
}

func Unauthorized(message string) error {
	return Error(http.StatusUnauthorized, message, nil)
}

func Forbidden(message string) error {
	return Error(http.StatusForbidden, message, nil)
}

// Redirect returns a loader-safe redirect error. Returning this from Page or
// Layout prevents rendering after the redirect response has been committed.
func Redirect(path string, status int) error {
	path = strings.TrimSpace(path)
	if status < 300 || status > 399 {
		status = http.StatusSeeOther
	}
	return &RedirectError{Path: path, Status: status}
}

func (app *App) defaultErrorHandler(writer http.ResponseWriter, request *http.Request, err error) {
	var redirect *RedirectError
	if errors.As(err, &redirect) {
		http.Redirect(writer, request, redirect.Path, redirect.Status)
		return
	}
	status, message := publicError(err)
	if status >= 500 {
		log.Printf("northframe: request failed: %v", err)
	}
	http.Error(writer, message, status)
}

func publicError(err error) (int, string) {
	status := http.StatusInternalServerError
	message := http.StatusText(status)
	var httpError *HTTPError
	if errors.As(err, &httpError) {
		status = httpError.Status
		if httpError.Message != "" {
			message = httpError.Message
		}
	}
	return status, message
}
