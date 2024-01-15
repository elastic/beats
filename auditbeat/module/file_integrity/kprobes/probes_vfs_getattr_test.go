package kprobes

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_vfsGetAttr_buildProbes(t *testing.T) {
	specs, err := loadEmbeddedSpecs()
	require.NoError(t, err)
	require.NotEmpty(t, specs)

	s := &vfsGetAttrSymbol{}

	for _, spec := range specs {
		switch {
		case spec.ContainsSymbol("vfs_getattr_nosec"):
			s.symbolName = "vfs_getattr_nosec"
		case spec.ContainsSymbol("vfs_getattr"):
			s.symbolName = "vfs_getattr"
		default:
			t.FailNow()
		}

		_, err := s.buildProbes(spec)
		require.NoError(t, err)

		if err != nil {
			t.FailNow()
		}
	}
}

func Test_vfsGetAttr_load(t *testing.T) {
	exec := newFixedThreadExecutor(context.TODO())

	prbMgr := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.ErrorIs(t, loadVFSGetAttrSymbol(prbMgr, exec), ErrSymbolNotFound)

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "vfs_getattr_nosec" {
			return runtimeSymbolInfo{
				symbolName:          "vfs_getattr_nosec",
				isOptimised:         true,
				optimisedSymbolName: "vfs_getattr_nosec.isra.0",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.Error(t, loadVFSGetAttrSymbol(prbMgr, exec))

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "vfs_getattr" {
			return runtimeSymbolInfo{
				symbolName:          "vfs_getattr",
				isOptimised:         false,
				optimisedSymbolName: "",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.NoError(t, loadVFSGetAttrSymbol(prbMgr, exec))
	require.NotEmpty(t, prbMgr.symbols)
	require.NotEmpty(t, prbMgr.buildChecks)
	require.IsType(t, &vfsGetAttrSymbol{}, prbMgr.symbols[0])
	require.Equal(t, prbMgr.symbols[0].(*vfsGetAttrSymbol).symbolName, "vfs_getattr")

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName == "vfs_getattr_nosec" {
			return runtimeSymbolInfo{
				symbolName:          "vfs_getattr_nosec",
				isOptimised:         false,
				optimisedSymbolName: "",
			}, nil
		}
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.NoError(t, loadVFSGetAttrSymbol(prbMgr, exec))
	require.NotEmpty(t, prbMgr.symbols)
	require.NotEmpty(t, prbMgr.buildChecks)
	require.IsType(t, &vfsGetAttrSymbol{}, prbMgr.symbols[0])
	require.Equal(t, prbMgr.symbols[0].(*vfsGetAttrSymbol).symbolName, "vfs_getattr_nosec")

	prbMgr = &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}
	unknownErr := errors.New("unknown error")
	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, unknownErr
	}
	require.Error(t, loadVFSGetAttrSymbol(prbMgr, exec))
}

func Test_vfsGetAttr_onErr(t *testing.T) {
	s := &vfsGetAttrSymbol{}

	testErr := fmt.Errorf("test: %w", ErrVerifyOverlappingEvents)
	repeat := s.onErr(testErr)
	require.False(t, repeat)

}
