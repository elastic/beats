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
	"github.com/stretchr/testify/assert"
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
	// wrap mimics verify(), which never returns the bare sentinel.
	wrap := func(err error) error {
		return fmt.Errorf("events could not be verified: %w", err)
	}

	tests := []struct {
		name   string
		errs   []error
		repeat []bool
	}{
		{
			name:   "each sentinel retried once",
			errs:   []error{ErrVerifyOverlappingEvents, ErrVerifyMissingEvents, ErrVerifyMissingEvents, ErrVerifyUnexpectedEvent},
			repeat: []bool{true, true, false, false},
		},
		{
			name:   "wrapped repeated error is not retried again",
			errs:   []error{wrap(ErrVerifyMissingEvents), wrap(ErrVerifyMissingEvents)},
			repeat: []bool{true, false},
		},
		{
			name:   "wrapped alternating errors are bounded",
			errs:   []error{wrap(ErrVerifyMissingEvents), wrap(ErrVerifyOverlappingEvents), wrap(ErrVerifyMissingEvents), wrap(ErrVerifyOverlappingEvents)},
			repeat: []bool{true, true, false, false},
		},
		{
			name:   "unrelated error is not retried",
			errs:   []error{wrap(ErrAckTimeout)},
			repeat: []bool{false},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s := &fsNotifySymbol{
				symbolName: "fsnotify",
			}

			for i, err := range tc.errs {
				assert.Equal(t, tc.repeat[i], s.onErr(err), "unexpected retry decision for call %d with error %q", i, err)
			}
		})
	}
}
