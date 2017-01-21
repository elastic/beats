package paths

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomePath(t *testing.T) {
	type io struct {
		Home       string // cli flag home setting
		Path       string // requested path
		Result     string // expected result
		ResultData string // expected data path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	assert.NoError(t, err)

	tests := []io{
		{
			Home:       binDir,
			Path:       "test",
			Result:     filepath.Join(binDir, "test"),
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			Home:       "/tmp",
			Path:       "test",
			Result:     "/tmp/test",
			ResultData: "/tmp/data/test",
		},
		{
			Home:       "/home/",
			Path:       "/abc/test",
			Result:     "/abc/test",
			ResultData: "/abc/test",
		},
	}

	for _, test := range tests {
		t.Log("Executing test", test)
		cfg := Path{Home: test.Home}
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
		Home       string // cli flag home setting
		Data       string // cli flag for data setting
		Path       string // requested path
		ResultData string // expected data path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	assert.NoError(t, err)

	tests := []io{
		{
			Home:       binDir,
			Data:       "",
			Path:       "test",
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			Home:       "/tmp/",
			Data:       "/root/",
			Path:       "test",
			ResultData: "/root/test",
		},
		{
			Home:       "/tmp/",
			Data:       "/root/",
			Path:       "/var/data",
			ResultData: "/var/data",
		},
	}

	for _, test := range tests {
		t.Log("Executing test", test)
		cfg := Path{Home: test.Home, Data: test.Data}
		assert.NoError(t, Paths.initPaths(&cfg))

		assert.Equal(t, test.ResultData, Resolve(Data, test.Path))
	}

}

func TestLogsPath(t *testing.T) {
	type io struct {
		Home       string // cli flag home setting
		Logs       string // cli flag for data setting
		Path       string // requested path
		ResultLogs string // expected logs path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	assert.NoError(t, err)

	tests := []io{
		{
			Home:       binDir,
			Logs:       "",
			Path:       "test",
			ResultLogs: filepath.Join(binDir, "logs", "test"),
		},
		{
			Home:       "/tmp/",
			Logs:       "/var/",
			Path:       "log",
			ResultLogs: "/var/log",
		},
		{
			Home:       "/tmp/",
			Logs:       "/root/",
			Path:       "/var/log",
			ResultLogs: "/var/log",
		},
	}

	for _, test := range tests {
		t.Log("Executing test", test)
		cfg := Path{Home: test.Home, Logs: test.Logs}
		assert.NoError(t, Paths.initPaths(&cfg))

		assert.Equal(t, test.ResultLogs, Resolve(Logs, test.Path))
	}

}
