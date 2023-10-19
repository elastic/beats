package kprobes

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_fsNotifySymbol_buildProbes(t *testing.T) {
	specs, err := loadEmbeddedSpecs()
	require.NoError(t, err)
	require.NotEmpty(t, specs)

	s := &fsNotifySymbol{
		symbolName: "fsnotify",
		lastOnErr:  nil,
	}

	for _, spec := range specs {

		if !spec.ContainsSymbol("fsnotify") {
			t.FailNow()
		}

		_, err := s.buildProbes(spec)
		require.NoError(t, err)
	}
}

func Test_fsNotifySymbol_onErr(t *testing.T) {
	s := &fsNotifySymbol{
		symbolName: "fsnotify",
		lastOnErr:  nil,
	}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	require.True(t, s.onErr(testErr))

	testErr = fmt.Errorf("test: %w", ErrVerifyMissingEvents)
	require.True(t, s.onErr(testErr))

	testErr = fmt.Errorf("test: %w", ErrVerifyUnexpectedEvent)
	require.False(t, s.onErr(testErr))

}
