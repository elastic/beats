// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package provider

import (
	"context"
	"errors"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
)

// SyncInterruptedMsg is the log message to use when SyncInterrupted
// reports true, so the event is worded identically across providers.
const SyncInterruptedMsg = "Sync interrupted by input shutdown or reconfiguration; will retry on next interval"

// SyncInterrupted reports whether err is the benign result of the input's
// cancellation context being cancelled while a sync was in flight — the
// agent stopping, restarting, or reconfiguring the input — rather than a
// failure of the sync itself. Callers should log this condition at info
// level and leave the input's status untouched: the interrupted sync has
// been rolled back and collection resumes on the next interval, or in the
// replacement input.
func SyncInterrupted(inputCtx v2.Context, err error) bool {
	return errors.Is(err, context.Canceled) && inputCtx.Cancelation.Err() != nil
}
