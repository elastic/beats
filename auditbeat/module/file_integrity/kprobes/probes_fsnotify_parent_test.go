package kprobes

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_fsNotifyParentSymbol_buildProbes(t *testing.T) {
	specs, err := loadEmbeddedSpecs()
	require.NoError(t, err)
	require.NotEmpty(t, specs)

	s := &fsNotifyParentSymbol{}

	for _, spec := range specs {
		switch {
		case spec.ContainsSymbol("__fsnotify_parent"):
			s.symbolName = "__fsnotify_parent"
		case spec.ContainsSymbol("fsnotify_parent"):
			s.symbolName = "fsnotify_parent"
		default:
			t.FailNow()
		}

		_, err := s.buildProbes(spec)
		require.NoError(t, err)
	}
}

func Test_fsNotifyParentSymbol_onErr(t *testing.T) {
	s := &fsNotifyParentSymbol{}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	repeat := s.onErr(testErr)
	require.False(t, repeat)

}
