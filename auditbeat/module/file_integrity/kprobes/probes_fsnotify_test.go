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

//go:build linux

package kprobes

import (
	"fmt"
	"testing"

	"github.com/cilium/ebpf/btf"
	"github.com/stretchr/testify/require"
)

func Test_fsNotifyDataTypeBTF(t *testing.T) {
	// fetch the data types ourselves for comparison
	dentry := ""
	inode := ""
	path := ""
	rawSpec, err := btf.LoadKernelSpec()
	require.NoError(t, err)

	var knownBtf *btf.Enum
	err = rawSpec.TypeByName("fsnotify_data_type", &knownBtf)
	require.NoError(t, err)
	for _, enumType := range knownBtf.Values {
		switch enumType.Name {
		case "FSNOTIFY_EVENT_PATH":
			path = fmt.Sprintf("dt==%d", enumType.Value)
		case "FSNOTIFY_EVENT_INODE":
			inode = fmt.Sprintf("dt==%d", enumType.Value)
		case "FSNOTIFY_EVENT_DENTRY":
			dentry = fmt.Sprintf("dt==%d", enumType.Value)
		}
	}

	// now the actual test
	rawBtf, err := loadAllSpecs()
	require.NoError(t, err)

	fsNotify := fsNotifySymbol{}

	err = fsNotify.setKprobeFiltersFromBTF(rawBtf[0])
	require.NoError(t, err)

	require.Contains(t, fsNotify.dentryProbeFilter, dentry, "could not verify dentry filter")
	require.Contains(t, fsNotify.inodeProbeFilter, inode, "could not verify inode filter")
	require.Contains(t, fsNotify.pathProbeFilter, path, "could not verify path filter")

}

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

func Test_fsNotifySymbol_load(t *testing.T) {
	prbMgr := &probeManager{
		symbols:              nil,
		buildChecks:          nil,
		getSymbolInfoRuntime: nil,
	}

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{}, ErrSymbolNotFound
	}
	require.ErrorIs(t, loadFsNotifySymbol(prbMgr), ErrSymbolNotFound)

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		if symbolName != "fsnotify" {
			return runtimeSymbolInfo{}, ErrSymbolNotFound
		}

		return runtimeSymbolInfo{
			symbolName:          "fsnotify",
			isOptimised:         true,
			optimisedSymbolName: "fsnotify.isra.0",
		}, nil
	}

	require.Error(t, loadFsNotifySymbol(prbMgr))

	prbMgr.getSymbolInfoRuntime = func(symbolName string) (runtimeSymbolInfo, error) {
		return runtimeSymbolInfo{
			symbolName:          "fsnotify",
			isOptimised:         false,
			optimisedSymbolName: "",
		}, nil
	}

	require.NoError(t, loadFsNotifySymbol(prbMgr))
	require.Equal(t, len(prbMgr.symbols), 1)
	require.Equal(t, len(prbMgr.buildChecks), 1)
}

func Test_fsNotifySymbol_onErr(t *testing.T) {
	s := &fsNotifySymbol{
		symbolName: "fsnotify",
		lastOnErr:  nil,
	}

	require.True(t, s.onErr(ErrVerifyOverlappingEvents))

	require.True(t, s.onErr(ErrVerifyMissingEvents))

	require.False(t, s.onErr(ErrVerifyMissingEvents))

	require.False(t, s.onErr(ErrVerifyUnexpectedEvent))
}
