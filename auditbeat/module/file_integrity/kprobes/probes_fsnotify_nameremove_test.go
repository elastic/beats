package kprobes

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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

func Test_fsNotifyNameRemoveSymbol_load(t *testing.T) {
	prbMgr := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.NoError(t, loadFsNotifyNameRemoveSymbol(prbMgr))
	require.Equal(t, len(prbMgr.symbols), 0)
	require.Equal(t, len(prbMgr.buildChecks), 1)

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName != "fsnotify_nameremove" {
			return runtimeSymbolInfo{}, ErrSymbolNotFound
		}

		return runtimeSymbolInfo{
			symbolName:          "fsnotify_nameremove",
			isOptimised:         true,
			optimisedSymbolName: "fsnotify_nameremove.isra.0",
		}, nil
	}
	require.Error(t, loadFsNotifyNameRemoveSymbol(prbMgr))

	unknownErr := errors.New("unknown error")
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, unknownErr
	}
	require.Error(t, loadFsNotifyNameRemoveSymbol(prbMgr))

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{
			symbolName:          "fsnotify_nameremove",
			isOptimised:         false,
			optimisedSymbolName: "",
		}, nil
	}

	require.NoError(t, loadFsNotifyNameRemoveSymbol(prbMgr))
	require.Equal(t, len(prbMgr.symbols), 1)
	require.Equal(t, len(prbMgr.buildChecks), 1)
}

func Test_fsNotifyNameRemoveSymbol_onErr(t *testing.T) {
	s := &fsNotifyNameRemoveSymbol{}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	repeat := s.onErr(testErr)
	require.False(t, repeat)

}
