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
	"testing"

	"github.com/stretchr/testify/require"
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
