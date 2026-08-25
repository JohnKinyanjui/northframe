package web

import (
	"errors"
	"net/http"
	"strings"
)

const enhanceHeader = "X-Northframe-Enhance"

// ActionResult is the stable server-to-browser contract for typed actions.
// Data is deliberately application-defined while validation errors remain
// predictable for forms and TypeScript event handlers.
type ActionResult struct {
	Success  bool        `json:"success"`
	Message  string      `json:"message,omitempty"`
	Errors   FieldErrors `json:"errors,omitempty"`
	Data     any         `json:"data,omitempty"`
	Redirect string      `json:"redirect,omitempty"`
	status   int
}

// ActionSuccess creates a successful result with an optional serializable value.
func ActionSuccess(message string, data any) ActionResult {
	return ActionResult{Success: true, Message: message, Data: data, status: http.StatusOK}
}

// ActionInvalid creates a field-addressable validation result.
func ActionInvalid(message string, fields FieldErrors) ActionResult {
	return ActionResult{Success: false, Message: message, Errors: fields, status: http.StatusUnprocessableEntity}
}

// ActionRedirect redirects native and progressively-enhanced form submissions.
func ActionRedirect(path string, status int) ActionResult {
	if status < 300 || status > 399 {
		status = http.StatusSeeOther
	}
	return ActionResult{Success: true, Redirect: path, status: status}
}

func writeActionResult(ctx *Context, result ActionResult) error {
	if ctx == nil || ctx.Request == nil || ctx.Response == nil {
		return errors.New("northframe: an action result requires a writable request context")
	}
	if result.Redirect != "" {
		return ctx.Redirect(result.Redirect, result.status)
	}
	status := result.status
	if status == 0 {
		if result.Success {
			status = http.StatusOK
		} else {
			status = http.StatusUnprocessableEntity
		}
	}
	if isEnhancedAction(ctx.Request) || acceptsJSON(ctx.Request) {
		return ctx.JSON(status, result)
	}
	if !result.Success {
		message := result.Message
		if message == "" {
			message = "The submitted form is invalid"
		}
		return Error(status, message, nil)
	}
	return ctx.NoContent()
}

func isEnhancedAction(request *http.Request) bool {
	return request.Header.Get(enhanceHeader) == "true"
}

func acceptsJSON(request *http.Request) bool {
	for _, value := range request.Header.Values("Accept") {
		if strings.Contains(value, "application/json") || strings.Contains(value, "application/*") || strings.Contains(value, "*/*") {
			return true
		}
	}
	return false
}
