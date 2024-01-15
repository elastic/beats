package kprobes

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
	"golang.org/x/sys/unix"
)

func Test_EventsVerifier(t *testing.T) {

	type verifierEvents struct {
		path string
		op   uint32
	}

	cases := []struct {
		name           string
		emitErr        error
		verifyErr      error
		expectedEvents []verifierEvents
		emittedEvents  []verifierEvents
	}{
		{
			"no_error",
			nil,
			nil,
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
				{
					path: "test",
					op:   unix.IN_MOVED_FROM,
				},
				{
					path: "test",
					op:   unix.IN_MOVED_TO,
				},
				{
					path: "test",
					op:   unix.IN_MODIFY,
				},
				{
					path: "test",
					op:   unix.IN_CREATE,
				},
				{
					path: "test",
					op:   unix.IN_DELETE,
				},
			},
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
				{
					path: "test",
					op:   unix.IN_MOVED_FROM,
				},
				{
					path: "test",
					op:   unix.IN_MOVED_TO,
				},
				{
					path: "test",
					op:   unix.IN_MODIFY,
				},
				{
					path: "test",
					op:   unix.IN_CREATE,
				},
				{
					path: "test",
					op:   unix.IN_DELETE,
				},
			},
		}, {
			"overlapping_events",
			nil,
			ErrVerifyOverlappingEvents,
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
			},
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
			},
		}, {
			"missing_events",
			nil,
			ErrVerifyMissingEvents,
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
			},
			nil,
		}, {
			"unexpected_events",
			ErrVerifyUnexpectedEvent,
			nil,
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_ATTRIB,
				},
			},
			[]verifierEvents{
				{
					path: "test",
					op:   unix.IN_DELETE,
				},
			},
		}, {
			"no_events_to_expect",
			nil,
			ErrVerifyNoEventsToExpect,
			nil,
			nil,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			e, err := newEventsVerifier("")
			require.NoError(t, err)

			for _, ev := range c.expectedEvents {
				e.addEventToExpect(ev.path, ev.op)
			}

			for _, ev := range c.emittedEvents {
				require.ErrorIs(t, e.validateEvent(ev.path, 0, ev.op), c.emitErr)
				if c.emitErr != nil {
					return
				}
			}

			require.ErrorIs(t, e.Verified(), c.verifyErr)
		})
	}
}

func Test_EventsVerifier_GenerateEvents(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("skipping on non-linux")
	}

	tmpDir, err := os.MkdirTemp("", "kprobe_unit_test")
	require.NoError(t, err)

	defer func() {
		rmErr := os.RemoveAll(tmpDir)
		require.NoError(t, rmErr)
	}()

	e, err := newEventsVerifier(tmpDir)
	require.NoError(t, err)

	err = e.GenerateEvents()
	require.NoError(t, err)

	require.NotEmpty(t, e.eventsToExpect)
}
