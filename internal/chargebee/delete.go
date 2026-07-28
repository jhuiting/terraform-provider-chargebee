// SPDX-License-Identifier: MPL-2.0

package chargebee

import (
	"context"
	"fmt"
	"time"
)

// DeleteOutcome describes what DeleteWithArchiveFallback did with the object.
type DeleteOutcome int

const (
	// LeftInPlace means the object still exists unchanged in Chargebee.
	LeftInPlace DeleteOutcome = iota
	// Deleted means the object was deleted (or was already gone).
	Deleted
	// Archived means deletion was refused and the object was archived instead.
	Archived
)

// deleteRetryAttempts bounds how many extra times a delete refused as in-use is
// retried before falling back to archiving. A refusal during `terraform destroy`
// is often transient: a parent (e.g. an item) is deleted concurrently with its
// children (its item prices), so the first attempt is refused only until the
// children finish deleting. Retrying a few times lets the concurrent deletes
// complete, so users do not have to wire explicit depends_on edges. A genuinely
// in-use object keeps being refused and is archived once the attempts run out.
var deleteRetryAttempts = 5

// deleteRetryWait blocks before the next retry, honoring context cancellation.
// Backoff grows 250ms, 500ms, 1s, 2s, 4s and is capped at 4s (worst-case ~7.75s
// before archiving a truly in-use object). It is a package var so tests can make
// retries instant.
var deleteRetryWait = func(ctx context.Context, attempt int) error {
	d := min(250*time.Millisecond<<attempt, 4*time.Second)
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// DeleteWithArchiveFallback deletes an object, falling back to archiving when
// Chargebee refuses because the object is in use. An in-use refusal is retried
// with backoff first (see deleteRetryAttempts): during destroy it is usually a
// transient race with the object's children being deleted concurrently, and the
// retry lets the real delete succeed instead of needlessly archiving. A nil
// archive means the object type cannot be archived; the in-use case then
// succeeds without touching the object (caller should warn). An object Chargebee
// reports as already archived is treated as an Archived success, since that is
// its terminal state.
func DeleteWithArchiveFallback(ctx context.Context, del, archive func(context.Context) error) (DeleteOutcome, error) {
	err := del(ctx)
	for attempt := 0; attempt < deleteRetryAttempts && IsInUse(err); attempt++ {
		if waitErr := deleteRetryWait(ctx, attempt); waitErr != nil {
			return LeftInPlace, waitErr
		}
		err = del(ctx)
	}

	switch {
	case err == nil, IsNotFound(err):
		return Deleted, nil
	case IsAlreadyArchived(err):
		return Archived, nil
	case IsInUse(err):
		if archive == nil {
			return LeftInPlace, nil
		}
		if archiveErr := archive(ctx); archiveErr != nil {
			return LeftInPlace, fmt.Errorf("delete refused (in use) and archive failed: %w", archiveErr)
		}
		return Archived, nil
	default:
		return LeftInPlace, fmt.Errorf("failed to delete: %w", err)
	}
}
