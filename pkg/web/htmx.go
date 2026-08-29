package web

import (
	"encoding/json"
	"errors"
	"strings"
)

const (
	htmxRequestHeader      = "HX-Request"
	htmxBoostedHeader      = "HX-Boosted"
	htmxCurrentURLHeader   = "HX-Current-URL"
	htmxPromptHeader       = "HX-Prompt"
	htmxTargetHeader       = "HX-Target"
	htmxTriggerNameHeader  = "HX-Trigger-Name"
	htmxTriggerHeader      = "HX-Trigger"
	htmxLocationHeader     = "HX-Location"
	htmxPushURLHeader      = "HX-Push-Url"
	htmxRedirectHeader     = "HX-Redirect"
	htmxRefreshHeader      = "HX-Refresh"
	htmxReplaceURLHeader   = "HX-Replace-Url"
	htmxReswapHeader       = "HX-Reswap"
	htmxRetargetHeader     = "HX-Retarget"
	htmxReselectHeader     = "HX-Reselect"
	htmxTriggerAfterSwap   = "HX-Trigger-After-Swap"
	htmxTriggerAfterSettle = "HX-Trigger-After-Settle"
)

// HTMX reports whether the request was issued by htmx. Normal browser
// navigation and form submission remain the fallback contract.
func (current *Context) HTMX() bool {
	return current != nil && current.Request != nil && headerTrue(current.Request.Header.Get(htmxRequestHeader))
}

func (current *Context) HXBoosted() bool {
	return current != nil && current.Request != nil && headerTrue(current.Request.Header.Get(htmxBoostedHeader))
}

func (current *Context) HXCurrentURL() string {
	return requestHeader(current, htmxCurrentURLHeader)
}

func (current *Context) HXPrompt() string {
	return requestHeader(current, htmxPromptHeader)
}

func (current *Context) HXTarget() string {
	return requestHeader(current, htmxTargetHeader)
}

func (current *Context) HXTriggerName() string {
	return requestHeader(current, htmxTriggerNameHeader)
}

// HXLocation performs an htmx navigation without forcing a full-page reload.
func (current *Context) HXLocation(path string) error {
	return current.setHXHeader(htmxLocationHeader, path)
}

// HXRedirect asks htmx to perform a full browser navigation.
func (current *Context) HXRedirect(path string) error {
	return current.setHXHeader(htmxRedirectHeader, path)
}

func (current *Context) HXRefresh() error {
	return current.setHXHeader(htmxRefreshHeader, "true")
}

func (current *Context) HXPushURL(path string) error {
	return current.setHXHeader(htmxPushURLHeader, path)
}

func (current *Context) HXReplaceURL(path string) error {
	return current.setHXHeader(htmxReplaceURLHeader, path)
}

func (current *Context) HXReswap(strategy string) error {
	return current.setHXHeader(htmxReswapHeader, strategy)
}

func (current *Context) HXRetarget(selector string) error {
	return current.setHXHeader(htmxRetargetHeader, selector)
}

func (current *Context) HXReselect(selector string) error {
	return current.setHXHeader(htmxReselectHeader, selector)
}

// HXTrigger dispatches a browser event immediately after htmx receives the
// response. Use HXTriggerEvents when an event needs structured detail.
func (current *Context) HXTrigger(name string) error {
	return current.setHXHeader(htmxTriggerHeader, name)
}

func (current *Context) HXTriggerEvents(events map[string]any) error {
	return current.setHXJSONHeader(htmxTriggerHeader, events)
}

func (current *Context) HXTriggerEventsAfterSwap(events map[string]any) error {
	return current.setHXJSONHeader(htmxTriggerAfterSwap, events)
}

func (current *Context) HXTriggerEventsAfterSettle(events map[string]any) error {
	return current.setHXJSONHeader(htmxTriggerAfterSettle, events)
}

func (current *Context) setHXJSONHeader(name string, value map[string]any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return current.setHXHeader(name, string(encoded))
}

func (current *Context) setHXHeader(name, value string) error {
	if current == nil || current.Response == nil {
		return errors.New("cannot set an htmx response header from a page or layout loader")
	}
	current.Response.Header().Set(name, value)
	return nil
}

func requestHeader(current *Context, name string) string {
	if current == nil || current.Request == nil {
		return ""
	}
	return current.Request.Header.Get(name)
}

func headerTrue(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}
