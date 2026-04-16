package fetch

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrTooManyRedirects is returned when the HTTP redirect chain exceeds the configured limit.
var ErrTooManyRedirects = errors.New("too many redirects")

// FetchError describes a failed fetch (upstream or transport).
type FetchError struct {
	HTTPStatus int    // response status from server, or 0 if none
	Message    string // brief explanation
}

func (e *FetchError) Error() string { return e.Message }

func mapDoError(err error) error {
	if errors.Is(err, ErrTooManyRedirects) {
		return &FetchError{HTTPStatus: 0, Message: "too many HTTP redirects"}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &FetchError{HTTPStatus: 0, Message: "request deadline exceeded"}
	}
	if errors.Is(err, context.Canceled) {
		return &FetchError{HTTPStatus: 0, Message: "request canceled"}
	}
	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return &FetchError{HTTPStatus: 0, Message: fmt.Sprintf("DNS error: %v", dnsErr)}
	}
	var ue *url.Error
	if errors.As(err, &ue) && ue.Timeout() {
		return &FetchError{HTTPStatus: 0, Message: fmt.Sprintf("timeout reaching %s", ue.URL)}
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Timeout() {
		return &FetchError{HTTPStatus: 0, Message: fmt.Sprintf("network timeout: %v", opErr)}
	}
	msg := err.Error()
	if strings.Contains(msg, "x509:") || strings.Contains(msg, "certificate") {
		return &FetchError{HTTPStatus: 0, Message: "TLS error: " + msg}
	}
	return &FetchError{HTTPStatus: 0, Message: msg}
}
