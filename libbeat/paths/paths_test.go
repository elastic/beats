package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomePath(t *testing.T) {
	type io struct {
		CLIHome    *string // cli flag home setting
		CfgHome    string  // config file home setting
		Path       string  // requested path
		Result     string  // expected result
		ResultData string  // expected data path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	assert.NoError(t, err)
	tmp := "/tmp/"

	tests := []io{
		{
			CLIHome:    nil,
			CfgHome:    "",
			Path:       "test",
			Result:     filepath.Join(binDir, "test"),
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			CLIHome:    &tmp,
			CfgHome:    "",
			Path:       "test",
			Result:     "/tmp/test",
			ResultData: "/tmp/data/test",
		},
		{
			CLIHome:    &tmp,
			CfgHome:    "/root/",
			Path:       "test",
			Result:     "/tmp/test",
			ResultData: "/tmp/data/test",
		},
		{
			CLIHome:    nil,
			CfgHome:    "/root/",
			Path:       "test",
			Result:     "/root/test",
			ResultData: "/root/data/test",
		},
		{
			CLIHome:    nil,
			CfgHome:    "/root/",
			Path:       "/home/test",
			Result:     "/home/test",
			ResultData: "/home/test",
		},
	}

	for _, test := range tests {
		homePath = test.CLIHome
		cfg := Path{Home: test.CfgHome}
		assert.NoError(t, Paths.initPaths(&cfg))

		assert.Equal(t, test.Result, Resolve(Home, test.Path))

		// config path same as home path
		assert.Equal(t, test.Result, Resolve(Config, test.Path))

		// data path under home path
		assert.Equal(t, test.ResultData, Resolve(Data, test.Path))
	}

}

func TestDataPath(t *testing.T) {
	type io struct {
		CLIHome    *string // cli flag home setting
		CfgHome    string  // config file home setting
		CLIData    *string // cli flag for data setting
		CfgData    string  // config file data setting
		Path       string  // requested path
		ResultData string  // expected data path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	assert.NoError(t, err)
	tmp := "/tmp/"
	root := "/root/"

	tests := []io{
		{
			CLIHome:    nil,
			CfgHome:    "",
			CLIData:    nil,
			CfgData:    "",
			Path:       "test",
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			CLIHome:    nil,
			CfgHome:    "/tmp/",
			CLIData:    nil,
			CfgData:    "/root/",
			Path:       "test",
			ResultData: "/root/test",
		},
		{
			CLIHome:    &tmp,
			CfgHome:    "",
			CLIData:    nil,
			CfgData:    "/root/",
			Path:       "test",
			ResultData: "/root/test",
		},
		{
			CLIHome:    &tmp,
			CfgHome:    "",
			CLIData:    &root,
			CfgData:    "/root/data",
			Path:       "test",
			ResultData: "/root/test",
		},
	}

	for _, test := range tests {
		homePath = test.CLIHome
		dataPath = test.CLIData
		cfg := Path{Home: test.CfgHome, Data: test.CfgData}
		assert.NoError(t, Paths.initPaths(&cfg))

		assert.Equal(t, test.ResultData, Resolve(Data, test.Path))
	}

}
