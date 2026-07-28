// SPDX-License-Identifier: MPL-2.0

package chargebee_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/jhuiting/terraform-provider-chargebee/internal/chargebee"
)

func TestDeleteWithArchiveFallback(t *testing.T) {
	restore := chargebee.SetDeleteRetryForTest(5, func(context.Context, int) error { return nil })
	t.Cleanup(restore)

	inUse := &chargebee.APIError{HTTPStatus: 400, Code: "invalid_state_for_request"}
	notFound := &chargebee.APIError{HTTPStatus: 404, Code: "resource_not_found"}
	alreadyArchived := &chargebee.APIError{HTTPStatus: 400, Code: "invalid_request", Message: "Item price cannot be deleted because it is already archived."}
	boom := errors.New("boom")

	ok := func(context.Context) error { return nil }
	fail := func(err error) func(context.Context) error {
		return func(context.Context) error { return err }
	}
	// failThenOK refuses as in-use for the first n calls, then succeeds; it models
	// a parent whose children are still being deleted concurrently during destroy.
	failThenOK := func(n int) func(context.Context) error {
		calls := 0
		return func(context.Context) error {
			calls++
			if calls <= n {
				return inUse
			}
			return nil
		}
	}

	tests := []struct {
		name    string
		del     func(context.Context) error
		archive func(context.Context) error
		want    chargebee.DeleteOutcome
		wantErr bool
	}{
		{"delete succeeds", ok, ok, chargebee.Deleted, false},
		{"already gone", fail(notFound), ok, chargebee.Deleted, false},
		{"in use clears within retries", failThenOK(3), ok, chargebee.Deleted, false},
		{"in use persists, archived", fail(inUse), ok, chargebee.Archived, false},
		{"in use, no archive available", fail(inUse), nil, chargebee.LeftInPlace, false},
		{"in use, archive fails", fail(inUse), fail(boom), chargebee.LeftInPlace, true},
		{"already archived", fail(alreadyArchived), ok, chargebee.Archived, false},
		{"other delete error", fail(boom), ok, chargebee.LeftInPlace, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := chargebee.DeleteWithArchiveFallback(context.Background(), tt.del, tt.archive)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestDeleteWithArchiveFallbackContextCancelled(t *testing.T) {
	restore := chargebee.SetDeleteRetryForTest(5, func(ctx context.Context, _ int) error { return ctx.Err() })
	t.Cleanup(restore)

	inUse := &chargebee.APIError{HTTPStatus: 400, Code: "invalid_state_for_request"}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := chargebee.DeleteWithArchiveFallback(ctx,
		func(context.Context) error { return inUse },
		func(context.Context) error { return nil },
	)
	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, chargebee.LeftInPlace, got)
}
