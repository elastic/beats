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
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getSymbolInfoFromReader(t *testing.T) {
	// fsnotify.cold precedes fsnotify on purpose: distribution kernels list the
	// cold fragment first, and the exact symbol must still win. Likewise
	// fsnotify_nameremove.isra.0 precedes fsnotify_nameremove: an exact match
	// must win over an earlier non-cold clone too.
	content := `0000000000000000 t fsnotify_move
0000000000000000 t fsnotify.cold
0000000000000000 t __fsnotify_parent
0000000000000000 T fsnotify
0000000000000000 T fsnotifyy
0000000000000000 t fsnotify_file.isra.0	[btrfs]
0000000000000000 t chmod_common.isra.0
0000000000000000 t chmod_common.isra.0.cold
0000000000000000 t vfs_getattr.constprop.0.cold
0000000000000000 t vfs_getattr.constprop.0
0000000000000000 t fsnotify_nameremove.isra.0
0000000000000000 T fsnotify_nameremove
0000000000000000 t fsnotify_parent.cold`

	cases := []struct {
		tName               string
		symbolName          string
		isOptimised         bool
		optimisedSymbolName string
		err                 error
	}{
		{
			tName:               "symbol_exists",
			symbolName:          "fsnotify",
			isOptimised:         false,
			optimisedSymbolName: "",
			err:                 nil,
		},
		{
			tName:               "symbol_exists_optimised",
			symbolName:          "chmod_common",
			isOptimised:         true,
			optimisedSymbolName: "chmod_common.isra.0",
			err:                 nil,
		},
		{
			tName:               "symbol_exists_optimised_with_space_at_end",
			symbolName:          "fsnotify_file",
			isOptimised:         true,
			optimisedSymbolName: "fsnotify_file.isra.0",
			err:                 nil,
		},
		{
			tName:               "cold_fragment_listed_before_clone",
			symbolName:          "vfs_getattr",
			isOptimised:         true,
			optimisedSymbolName: "vfs_getattr.constprop.0",
			err:                 nil,
		},
		{
			tName:               "exact_symbol_listed_after_clone",
			symbolName:          "fsnotify_nameremove",
			isOptimised:         false,
			optimisedSymbolName: "",
			err:                 nil,
		},
		{
			tName:               "only_cold_fragment_is_not_found",
			symbolName:          "fsnotify_parent",
			isOptimised:         false,
			optimisedSymbolName: "",
			err:                 ErrSymbolNotFound,
		},
		{
			tName:               "symbol_missing",
			symbolName:          "vfs_getattr_nosec",
			isOptimised:         false,
			optimisedSymbolName: "",
			err:                 ErrSymbolNotFound,
		},
	}

	for _, tc := range cases {
		t.Run(tc.tName, func(t *testing.T) {
			symInfo, err := getSymbolInfoFromReader(strings.NewReader(content), tc.symbolName)
			if tc.err != nil {
				require.ErrorIs(t, err, tc.err, "expected lookup of %q to fail", tc.symbolName)
				return
			}
			require.NoError(t, err, "lookup of %q", tc.symbolName)
			require.Equal(t, tc.symbolName, symInfo.symbolName)
			require.Equal(t, tc.isOptimised, symInfo.isOptimised)
			require.Equal(t, tc.optimisedSymbolName, symInfo.optimisedSymbolName)
		})
	}
}
