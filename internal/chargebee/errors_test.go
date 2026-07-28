// SPDX-License-Identifier: MPL-2.0

package chargebee_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"http 404", &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}, true},
		{"code only", &chargebee.APIError{HTTPStatus: 400, Code: "resource_not_found"}, true},
		{"wrapped", fmt.Errorf("get item: %w", &chargebee.APIError{HTTPStatus: 404}), true},
		{"other api error", &chargebee.APIError{HTTPStatus: 400, Code: "param_wrong_value"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chargebee.IsNotFound(tt.err))
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"code already exists", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Code v3-test already exists"}, true},
		{"wrapped", fmt.Errorf("create item family: %w", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Code v3-test already exists"}), true},
		{"unrelated invalid_request", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "some other problem"}, false},
		{"not found", &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chargebee.IsAlreadyExists(tt.err))
		})
	}
}

func TestIsAlreadyArchived(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"already archived", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Item price cannot be deleted because it is already archived."}, true},
		{"wrapped", fmt.Errorf("delete item price: %w", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Item price cannot be deleted because it is already archived."}), true},
		{"unrelated invalid_request", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "some other problem"}, false},
		{"not found", &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chargebee.IsAlreadyArchived(tt.err))
		})
	}
}

func TestIsDraftState(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"draft state", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Feature awareness cannot be archived because it is in a draft state."}, true},
		{"wrapped", fmt.Errorf("archive feature: %w", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Feature awareness cannot be archived because it is in a draft state."}), true},
		{"unrelated invalid_request", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "some other problem"}, false},
		{"not found", &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chargebee.IsDraftState(tt.err))
		})
	}
}

func TestIsInUse(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"plain error", errors.New("boom"), false},
		{"invalid_state_for_request", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_state_for_request"}, true},
		{"operation_not_allowed", &chargebee.APIError{HTTPStatus: 400, Code: "operation_not_allowed"}, true},
		{"http 409", &chargebee.APIError{HTTPStatus: 409, Code: ""}, true},
		{"family has items attached", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Delete operation for v3-test item family is not supported since it has items attached to it. Delete the family last"}, true},
		{"wrapped", fmt.Errorf("delete item: %w", &chargebee.APIError{Code: "invalid_state_for_request"}), true},
		{"not found is not in-use", &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}, false},
		{"unrelated invalid_request", &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "some other problem"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, chargebee.IsInUse(tt.err))
		})
	}
}

func TestAPIErrorMessage(t *testing.T) {
	err := &chargebee.APIError{HTTPStatus: 400, Code: "invalid_state_for_request", Message: "cannot delete"}
	assert.Contains(t, err.Error(), "invalid_state_for_request")
	assert.Contains(t, err.Error(), "cannot delete")
}
