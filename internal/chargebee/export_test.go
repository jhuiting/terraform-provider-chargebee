// SPDX-License-Identifier: MPL-2.0

package chargebee

import "context"

// SetDeleteRetryForTest overrides the delete retry policy and returns a function
// that restores the previous values. It lets tests exercise the retry path
// without real backoff sleeps.
func SetDeleteRetryForTest(attempts int, wait func(context.Context, int) error) func() {
	prevAttempts, prevWait := deleteRetryAttempts, deleteRetryWait
	deleteRetryAttempts = attempts
	deleteRetryWait = wait
	return func() {
		deleteRetryAttempts = prevAttempts
		deleteRetryWait = prevWait
	}
}
