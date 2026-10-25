package claude

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/anthropics/anthropic-sdk-go"
)

// Describe turns an API or network error into a short message for the user.
// It never includes the API key.
func Describe(err error, model string) string {
	var apiErr *anthropic.Error
	if errors.As(err, &apiErr) {
		switch apiErr.StatusCode {
		case http.StatusUnauthorized:
			return "The API key was rejected. Use the tray menu: Set API key…"
		case http.StatusForbidden:
			return "The API key is not allowed to do this (permission denied)."
		case http.StatusPaymentRequired:
			return "The Anthropic account has a billing problem."
		case http.StatusNotFound:
			return fmt.Sprintf("Model %q was not found, or is not available to this account. Check \"model\" in the config.", model)
		case http.StatusRequestEntityTooLarge:
			return "The selected text is too large for the API."
		case http.StatusTooManyRequests:
			return "Rate limit reached. Try again in a moment."
		case http.StatusBadRequest:
			return "The API rejected the request: " + apiErr.Error()
		}
		if apiErr.StatusCode >= 500 {
			return fmt.Sprintf("The Claude API is having problems (HTTP %d). Try again later.", apiErr.StatusCode)
		}
		return fmt.Sprintf("The Claude API returned an error (HTTP %d).", apiErr.StatusCode)
	}
	var netErr net.Error
	if errors.As(err, &netErr) || errors.Is(err, context.DeadlineExceeded) {
		return "Cannot reach the Claude API. Check the network connection."
	}
	return "Request failed: " + err.Error()
}
