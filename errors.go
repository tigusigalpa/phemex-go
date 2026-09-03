package phemex

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// APIError is the common error type returned by the Phemex client.
type APIError struct {
	StatusCode int
	Code       int
	Message    string
	Body       []byte
}

func (e *APIError) Error() string {
	if e.Code != 0 && e.Message != "" {
		return fmt.Sprintf("phemex API error %d (code %d): %s", e.StatusCode, e.Code, e.Message)
	}
	if e.Code != 0 {
		return fmt.Sprintf("phemex API error %d (code %d)", e.StatusCode, e.Code)
	}
	if e.Message != "" {
		return fmt.Sprintf("phemex API error %d: %s", e.StatusCode, e.Message)
	}
	return fmt.Sprintf("phemex API error %d", e.StatusCode)
}

// AuthenticationError indicates an invalid or missing API key/secret.
type AuthenticationError struct {
	APIError
}

func (e *AuthenticationError) Error() string {
	return fmt.Sprintf("phemex authentication error %d: %s", e.StatusCode, e.Message)
}

// ValidationError indicates the request parameters were rejected.
type ValidationError struct {
	APIError
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("phemex validation error %d: %s", e.StatusCode, e.Message)
}

// NotFoundError indicates the requested resource was not found.
type NotFoundError struct {
	APIError
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("phemex not found error %d: %s", e.StatusCode, e.Message)
}

// RateLimitError indicates the request was throttled and exposes retry info.
type RateLimitError struct {
	APIError
	RetryAfter time.Duration
}

func (e *RateLimitError) Error() string {
	if e.RetryAfter > 0 {
		return fmt.Sprintf("phemex rate limit error %d: retry after %v", e.StatusCode, e.RetryAfter)
	}
	return fmt.Sprintf("phemex rate limit error %d", e.StatusCode)
}

func extractMessage(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	var envelope struct {
		Msg     string `json:"msg"`
		Message string `json:"message"`
		Error   string `json:"error"`
		Code    int    `json:"code"`
	}
	if err := json.Unmarshal(body, &envelope); err == nil {
		if envelope.Msg != "" {
			return envelope.Msg
		}
		if envelope.Message != "" {
			return envelope.Message
		}
		if envelope.Error != "" {
			return envelope.Error
		}
		if envelope.Code != 0 {
			return fmt.Sprintf("code %d", envelope.Code)
		}
	}
	return string(body)
}

func newAPIError(status int, message string) *APIError {
	return &APIError{StatusCode: status, Message: message}
}

func newHTTPError(status int, body []byte) *APIError {
	return newAPIErrorFromBody(status, 0, body)
}

func classifyHTTPError(status int, body []byte) error {
	e := newAPIErrorFromBody(status, 0, body)
	switch status {
	case http.StatusUnauthorized:
		return &AuthenticationError{*e}
	case http.StatusBadRequest, http.StatusForbidden:
		return &ValidationError{*e}
	case http.StatusNotFound:
		return &NotFoundError{*e}
	default:
		return e
	}
}

func newRateLimitError(status int, body []byte, retryAfter string) *RateLimitError {
	e := newAPIErrorFromBody(status, 0, body)
	return &RateLimitError{APIError: *e, RetryAfter: parseRetryAfter(retryAfter)}
}

func newAPIErrorFromBody(status, fallbackCode int, body []byte) *APIError {
	e := &APIError{StatusCode: status, Code: fallbackCode, Message: extractMessage(body), Body: body}
	var envelope struct {
		Code int `json:"code"`
	}
	if json.Unmarshal(body, &envelope) == nil {
		e.Code = envelope.Code
	}
	return e
}
