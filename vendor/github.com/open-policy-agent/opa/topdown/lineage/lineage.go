// Copyright 2019 The OPA Authors.  All rights reserved.
// Use of this source code is governed by an Apache2
// license that can be found in the LICENSE file.

package lineage

import (
	"github.com/open-policy-agent/opa/topdown"
)

// Notes returns a filtered trace that contains Note events and context to
// understand where the Note was emitted.
func Notes(trace []*topdown.Event) []*topdown.Event {
	return Filter(trace, func(event *topdown.Event) bool {
		return event.Op == topdown.NoteOp
	})
}

// Fails returns a filtered trace that contains Fail events and context to
// understand where the Fail occurred.
func Fails(trace []*topdown.Event) []*topdown.Event {
	return Filter(trace, func(event *topdown.Event) bool {
		return event.Op == topdown.FailOp
	})
}

// Filter will filter a given trace using the specified filter function. The
// filtering function should return true for events that should be kept, false
// for events that should be filtered out.
func Filter(trace []*topdown.Event, filter func(*topdown.Event) bool) (result []*topdown.Event) {

	qids := map[uint64]*topdown.Event{}

	for _, event := range trace {

		if filter(event) {
			// Path will end with the Note event.
			path := []*topdown.Event{event}

			// Construct path of recorded Enter/Redo events that lead to the
			// Note event. The path is constructed in reverse order by iterating
			// backwards through the Enter/Redo events from the Note event.
			curr := qids[event.QueryID]
			var prev *topdown.Event

			for curr != nil && curr != prev {
				path = append(path, curr)
				prev = curr
				curr = qids[curr.ParentID]
			}

			// Add the path to the result, reversing it in the process.
			for i := len(path) - 1; i >= 0; i-- {
				result = append(result, path[i])
			}

			qids = map[uint64]*topdown.Event{}
		}

		if event.Op == topdown.EnterOp || event.Op == topdown.RedoOp {
			if event.HasRule() || event.HasBody() {
				qids[event.QueryID] = event
			}
		}
	}

	return result
}
