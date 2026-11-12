// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package input_logfile

import (
	"runtime"
	"testing"
	"weak"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/acker"
)

type ackedPrivate int

const (
	privateCursorOp ackedPrivate = iota
	privateNil
	privateOther
)

func TestCursorACKHandler(t *testing.T) {
	cases := []struct {
		name    string
		entries []ackedPrivate
	}{
		{
			name:    "only cursor ops",
			entries: []ackedPrivate{privateCursorOp, privateCursorOp, privateCursorOp},
		},
		{
			name:    "nil and non-op entries are skipped",
			entries: []ackedPrivate{privateCursorOp, privateNil, privateOther, privateCursorOp, privateNil},
		},
		{
			name:    "no cursor ops",
			entries: []ackedPrivate{privateNil, privateOther},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := testOpenStore(t, "test", nil)
			defer store.Release()
			res := store.Get("test::key")

			listener, ops := publishAndACK(t, store, res, tc.entries)
			res.Release()

			runtime.GC()
			for i, op := range ops {
				assert.Nilf(t, op.Value(), "cursor op %d is still reachable after it was ACKed and executed", i)
			}
			runtime.KeepAlive(listener)
		})
	}
}

// publishAndACK publishes one event per entry through the cursor ACK handler,
// ACKs and executes them, and returns weak pointers to the cursor ops it created.
func publishAndACK(t *testing.T, store *store, res *resource, entries []ackedPrivate) (beat.EventListener, []weak.Pointer[updateOp]) {
	ch := newUpdateChan()
	handle := cursorACKHandler(ch)
	var got []any
	listener := acker.EventPrivateReporter(func(acked int, private []any) {
		handle(acked, private)
		got = private
	})

	var ops []*updateOp
	for i, entry := range entries {
		var private any
		switch entry {
		case privateCursorOp:
			op := mustCreateUpdateOp(t, res, i)
			ops = append(ops, op)
			private = op
		case privateOther:
			private = "not a cursor op"
		}
		listener.AddEvent(beat.Event{Private: private}, true)
	}
	listener.ACKEvents(len(entries))

	assert.Equal(t, make([]any, len(entries)), got, "the handler must clear every ACKed entry")

	var want []scheduledUpdate
	if len(ops) > 0 {
		want = []scheduledUpdate{{op: ops[len(ops)-1], n: uint(len(ops))}}
	}
	updates := ch.TryRecv()
	assert.Equal(t, want, updates, "the handler must schedule the last cursor op with the count of all ops")
	for _, upd := range updates {
		upd.op.Execute(store, upd.n)
	}

	weakOps := make([]weak.Pointer[updateOp], len(ops))
	for i, op := range ops {
		weakOps[i] = weak.Make(op)
	}
	return listener, weakOps
}
