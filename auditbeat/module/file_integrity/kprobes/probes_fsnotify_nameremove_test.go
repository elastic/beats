package kprobes

import (
	"fmt"
	"github.com/stretchr/testify/require"
	"testing"
)

func Test_fsNotifyNameRemoveSymbol_buildProbes(t *testing.T) {
	specs, err := loadEmbeddedSpecs()
	require.NoError(t, err)
	require.NotEmpty(t, specs)

	s := &fsNotifyNameRemoveSymbol{}

	for _, spec := range specs {
		switch {
		case spec.ContainsSymbol("fsnotify_nameremove"):
			s.symbolName = "fsnotify_nameremove"
		default:
			continue
		}

		_, err := s.buildProbes(spec)
		require.NoError(t, err)
	}
}

func Test_fsNotifyNameRemoveSymbol_onErr(t *testing.T) {
	s := &fsNotifyNameRemoveSymbol{}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	repeat := s.onErr(testErr)
	require.False(t, repeat)

}
