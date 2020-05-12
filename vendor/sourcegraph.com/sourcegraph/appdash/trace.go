package appdash

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"
)

// A Trace is a tree of spans.
type Trace struct {
	Span          // Root span
	Sub  []*Trace // Children
}

// String returns the Trace as a formatted string.
func (t *Trace) String() string {
	b, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(b)
}

// FindSpan recursively searches for a span whose Span ID is spanID in
// t and its descendants. If no such span is found, nil is returned.
func (t *Trace) FindSpan(spanID ID) *Trace {
	if t.ID.Span == spanID {
		return t
	}
	for _, sub := range t.Sub {
		if s := sub.FindSpan(spanID); s != nil {
			return s
		}
	}
	return nil
}

// TreeString returns the Trace as a formatted string that visually
// represents the trace's tree.
func (t *Trace) TreeString() string {
	var buf bytes.Buffer
	t.treeString(&buf, 0)
	return buf.String()
}

func (t *Trace) TimespanEvent() (TimespanEvent, error) {
	var events []Event
	if err := UnmarshalEvents(t.Annotations, &events); err != nil {
		return timespanEvent{}, err
	}
	start, end, ok := findTraceTimes(events)
	if !ok {
		return timespanEvent{}, errors.New("time span event not found")
	}
	return timespanEvent{S: start, E: end}, nil
}

func (t *Trace) treeString(w io.Writer, depth int) {
	const indent1 = "    "
	indent := strings.Repeat(indent1, depth)

	if depth == 0 {
		fmt.Fprintf(w, "+ Trace %x\n", uint64(t.Span.ID.Trace))
	} else {
		if depth == 1 {
			fmt.Fprint(w, "|")
		} else {
			fmt.Fprint(w, "|", indent[len(indent1):])
		}
		fmt.Fprintf(w, "%s+ Span %x", strings.Repeat("-", len(indent1)), uint64(t.Span.ID.Span))
		if t.Span.ID.Parent != 0 {
			fmt.Fprintf(w, " (parent %x)", uint64(t.Span.ID.Parent))
		}
		fmt.Fprintln(w)
	}
	for _, a := range t.Span.Annotations {
		if depth == 0 {
			fmt.Fprint(w, "| ")
		} else {
			fmt.Fprint(w, "|", indent[1:], " | ")
		}
		fmt.Fprintf(w, "%s = %s\n", a.Key, a.Value)
	}
	for _, sub := range t.Sub {
		sub.treeString(w, depth+1)
	}
}

// findTraceTimes finds the minimum and maximum timespan event times for the
// given set of events, or returns ok == false if there are no such events.
func findTraceTimes(events []Event) (start, end time.Time, _ bool) {
	// Find the start and end time of the trace.
	for _, e := range events {
		e, ok := e.(TimespanEvent)
		if !ok {
			continue
		}
		if start.IsZero() {
			start = e.Start()
			end = e.End()
			continue
		}
		if v := e.Start(); v.UnixNano() < start.UnixNano() {
			start = v
		}
		if v := e.End(); v.UnixNano() > end.UnixNano() {
			end = v
		}
	}
	return start, end, !start.IsZero()
}

type tracesByIDSpan []*Trace

func (t tracesByIDSpan) Len() int           { return len(t) }
func (t tracesByIDSpan) Less(i, j int) bool { return t[i].Span.ID.Span < t[j].Span.ID.Span }
func (t tracesByIDSpan) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
