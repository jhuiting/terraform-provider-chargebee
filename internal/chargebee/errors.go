// SPDX-License-Identifier: MPL-2.0

package chargebee

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
)

// APIError is a normalized Chargebee API error.
type APIError struct {
	HTTPStatus int
	Code       string
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("chargebee api error (http %d, code %q): %s", e.HTTPStatus, e.Code, e.Message)
}

// IsNotFound reports whether err means the requested object does not exist.
func IsNotFound(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.HTTPStatus == http.StatusNotFound || apiErr.Code == "resource_not_found"
}

// IsAlreadyExists reports whether err means an object with the requested id
// already exists. Chargebee returns a generic invalid_request for this, so the
// message ("Code <id> already exists") is the only reliable signal.
func IsAlreadyExists(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "already exists")
}

// IsAlreadyArchived reports whether err means deletion was refused because the
// object is already archived. Chargebee returns a generic invalid_request for
// this, so the message is the only reliable signal.
func IsAlreadyArchived(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "already archived")
}

// IsDraftState reports whether err means an operation was refused because the
// object is still in draft. Archiving a draft feature is rejected this way, but
// a draft feature can be deleted directly, so callers treat it as a no-op.
// Chargebee returns a generic invalid_request, so the message is the only
// reliable signal.
func IsDraftState(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return strings.Contains(strings.ToLower(apiErr.Message), "draft state")
}

// IsConflict reports whether err is a transient concurrency conflict that
// Chargebee asks the caller to retry. Concurrent writes to entitlements of the
// same entity come back as http 409 invalid_state_for_request.
func IsConflict(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	return apiErr.HTTPStatus == http.StatusConflict || apiErr.Code == "invalid_state_for_request"
}

// IsInUse reports whether err means the object is referenced by other data
// (e.g. subscriptions or invoices) and therefore cannot be deleted.
func IsInUse(err error) bool {
	var apiErr *APIError
	if !errors.As(err, &apiErr) {
		return false
	}
	switch apiErr.Code {
	case "invalid_state_for_request", "operation_not_allowed":
		return true
	}
	if apiErr.HTTPStatus == http.StatusConflict {
		return true
	}
	// Deleting an item family with items still attached comes back as a generic
	// invalid_request; the message is the only reliable in-use signal.
	return strings.Contains(strings.ToLower(apiErr.Message), "has items attached")
}
