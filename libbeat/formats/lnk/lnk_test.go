package lnk

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
		"local.directory.seven.lnk",
		"local.directory.xp.lnk",
		"local.file.darwin.lnk",
		"local.file.env.lnk",
		"local.file.exec.lnk",
		"local.file.icoset.lnk",
		"local.file.seven.lnk",
		"local.file.xp.lnk",
		"local_cmd.lnk",
		"local_unicode.lnk",
		"local_win31j.lnk",
		"microsoft.lnk",
		"native.2008srv.01.lnk",
		"native.2008srv.02.lnk",
		"native.2008srv.03.lnk",
		"native.2008srv.04.lnk",
		"native.2008srv.05.lnk",
		"native.2008srv.06.lnk",
		"native.2008srv.07.lnk",
		"native.2008srv.08.lnk",
		"native.2008srv.09.lnk",
		"native.2008srv.10.lnk",
		"native.2008srv.11.lnk",
		"native.2008srv.12.lnk",
		"native.2008srv.13.lnk",
		"native.2008srv.14.lnk",
		"native.2008srv.15.lnk",
		"native.2008srv.16.lnk",
		"native.2008srv.17.lnk",
		"native.2008srv.18.lnk",
		"native.2008srv.19.lnk",
		"native.2008srv.20.lnk",
		"native.seven.01.lnk",
		"native.seven.02.lnk",
		"native.seven.03.lnk",
		"native.seven.04.lnk",
		"native.seven.05.lnk",
		"native.seven.06.lnk",
		"native.seven.07.lnk",
		"native.seven.08.lnk",
		"native.seven.09.lnk",
		"native.seven.10.lnk",
		"native.seven.11.lnk",
		"native.seven.12.lnk",
		"native.seven.13.lnk",
		"native.seven.14.lnk",
		"native.seven.15.lnk",
		"native.seven.16.lnk",
		"native.seven.17.lnk",
		"native.seven.18.lnk",
		"native.seven.19.lnk",
		"native.seven.20.lnk",
		"native.xp.01.lnk",
		"native.xp.02.lnk",
		"native.xp.03.lnk",
		"native.xp.04.lnk",
		"native.xp.05.lnk",
		"native.xp.06.lnk",
		"native.xp.07.lnk",
		"native.xp.08.lnk",
		"native.xp.09.lnk",
		"native.xp.10.lnk",
		"native.xp.11.lnk",
		"native.xp.12.lnk",
		"native.xp.13.lnk",
		"native.xp.14.lnk",
		"native.xp.15.lnk",
		"native.xp.16.lnk",
		"native.xp.17.lnk",
		"native.xp.18.lnk",
		"native.xp.19.lnk",
		"native.xp.20.lnk",
		"net_unicode.lnk",
		"net_unicode2.lnk",
		"net_win31j.lnk",
		"remote.directory.xp.lnk",
		"remote.file.aidlist.lnk",
		"remote.file.xp.lnk",
	}
	for _, binary := range binaries {
		t.Run(binary, func(t *testing.T) {
			f, err := os.Open("../fixtures/lnk/" + binary)
			require.NoError(t, err)
			defer f.Close()

			info, err := Parse(f)
			require.NoError(t, err)

			expectedFile := "../fixtures/lnk/" + binary + ".fingerprint"
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
