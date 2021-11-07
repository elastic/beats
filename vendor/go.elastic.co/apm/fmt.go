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

package apm // import "go.elastic.co/apm"

import (
	"context"
	"fmt"
	"io"
)

// TraceFormatter returns a fmt.Formatter that can be used to
// format the identifiers of the transaction and span in ctx.
//
// The returned Formatter understands the following verbs:
//
//   %v: trace ID, transaction ID, and span ID (if existing), space-separated
//       the plus flag (%+v) adds field names, e.g. "trace.id=... transaction.id=..."
//   %t: trace ID (hex-encoded, or empty string if non-existent)
//       the plus flag (%+T) adds the field name, e.g. "trace.id=..."
//   %x: transaction ID (hex-encoded, or empty string if non-existent)
//       the plus flag (%+t) adds the field name, e.g. "transaction.id=..."
//   %s: span ID (hex-encoded, or empty string if non-existent)
//       the plus flag (%+s) adds the field name, e.g. "span.id=..."
func TraceFormatter(ctx context.Context) fmt.Formatter {
	f := traceFormatter{tx: TransactionFromContext(ctx)}
	if f.tx != nil {
		f.span = SpanFromContext(ctx)
	}
	return f
}

type traceFormatter struct {
	tx   *Transaction
	span *Span
}

func (t traceFormatter) Format(f fmt.State, c rune) {
	switch c {
	case 'v':
		if t.tx != nil {
			t.writeField(f, "trace.id", t.tx.TraceContext().Trace.String())
			f.Write([]byte{' '})
			t.writeField(f, "transaction.id", t.tx.TraceContext().Span.String())
			if t.span != nil {
				f.Write([]byte{' '})
				t.writeField(f, "span.id", t.span.TraceContext().Span.String())
			}
		}
	case 't':
		if t.tx != nil {
			t.writeField(f, "trace.id", t.tx.TraceContext().Trace.String())
		}
	case 'x':
		if t.tx != nil {
			t.writeField(f, "transaction.id", t.tx.TraceContext().Span.String())
		}
	case 's':
		if t.span != nil {
			t.writeField(f, "span.id", t.span.TraceContext().Span.String())
		}
	}
}

func (t traceFormatter) writeField(f fmt.State, name, value string) {
	if f.Flag('+') {
		io.WriteString(f, name)
		f.Write([]byte{'='})
	}
	io.WriteString(f, value)
}
