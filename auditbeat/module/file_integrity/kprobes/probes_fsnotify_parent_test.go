package kprobes

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
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

func Test_fsNotifyParentSymbol_load(t *testing.T) {
	prbMgr := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.ErrorIs(t, loadFsNotifyParentSymbol(prbMgr), ErrSymbolNotFound)

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "fsnotify_parent" {
			return runtimeSymbolInfo{
				symbolName:          "fsnotify_parent",
				isOptimised:         true,
				optimisedSymbolName: "fsnotify_parent.isra.0",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.Error(t, loadFsNotifyParentSymbol(prbMgr))

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "fsnotify_parent" {
			return runtimeSymbolInfo{
				symbolName:          "fsnotify_parent",
				isOptimised:         false,
				optimisedSymbolName: "",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.NoError(t, loadFsNotifyParentSymbol(prbMgr))
	require.NotEmpty(t, prbMgr.symbols)
	require.NotEmpty(t, prbMgr.buildChecks)
	require.IsType(t, &fsNotifyParentSymbol{}, prbMgr.symbols[0])
	require.Equal(t, prbMgr.symbols[0].(*fsNotifyParentSymbol).symbolName, "fsnotify_parent")

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "__fsnotify_parent" {
			return runtimeSymbolInfo{
				symbolName:          "__fsnotify_parent",
				isOptimised:         false,
				optimisedSymbolName: "",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.NoError(t, loadFsNotifyParentSymbol(prbMgr))
	require.NotEmpty(t, prbMgr.symbols)
	require.NotEmpty(t, prbMgr.buildChecks)
	require.IsType(t, &fsNotifyParentSymbol{}, prbMgr.symbols[0])
	require.Equal(t, prbMgr.symbols[0].(*fsNotifyParentSymbol).symbolName, "__fsnotify_parent")

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	unknownErr := errors.New("unknown error")
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, unknownErr
	}
	require.Error(t, loadFsNotifyParentSymbol(prbMgr))
}

func Test_fsNotifyParentSymbol_onErr(t *testing.T) {
	s := &fsNotifyParentSymbol{}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	repeat := s.onErr(testErr)
	require.False(t, repeat)

}
