package kprobes

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getSymbolInfoFromReader(t *testing.T) {

	content := `0000000000000000 t fsnotify_move
0000000000000000 T fsnotify
0000000000000000 T fsnotifyy
0000000000000000 t fsnotify_file.isra.0	[btrfs]
0000000000000000 t chmod_common.isra.0`

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
	}

	for _, tc := range cases {
		t.Run(tc.tName, func(t *testing.T) {
			symInfo, err := getSymbolInfoFromReader(strings.NewReader(content), tc.symbolName)
			require.IsType(t, err, tc.err)
			require.Equal(t, tc.symbolName, symInfo.symbolName)
			require.Equal(t, tc.isOptimised, symInfo.isOptimised)
			require.Equal(t, tc.optimisedSymbolName, symInfo.optimisedSymbolName)
		})
	}
}
