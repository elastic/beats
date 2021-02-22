package pe

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBinaries(t *testing.T) {
	generate := os.Getenv("GENERATE") == "1"
	binaries := []string{
		"hello-windows",
	}
	for _, binary := range binaries {
		t.Run(binary, func(t *testing.T) {
			f, err := os.Open("../fixtures/pe/" + binary)
			require.NoError(t, err)
			defer f.Close()

			info, err := Parse(f)
			require.NoError(t, err)

			expectedFile := "../fixtures/pe/" + binary + ".fingerprint"
			if generate {
				data, err := json.MarshalIndent(info, "", "  ")
				require.NoError(t, err)
				require.NoError(t, ioutil.WriteFile(expectedFile, data, 0644))
			} else {
				fixture, err := os.Open(expectedFile)
				require.NoError(t, err)
				defer fixture.Close()
				expected, err := ioutil.ReadAll(fixture)
				require.NoError(t, err)

				data, err := json.Marshal(info)
				require.NoError(t, err)
				require.JSONEq(t, string(expected), string(data))
			}
		})
	}
}
