package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomePath(t *testing.T) {
	type io struct {
		Home       string // cli flag home setting
		Path       string // requested path
		ResultHome string // expected home path
		ResultData string // expected data path
	}

	binDir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		t.Fatal(err)
	}

	tests := []io{
		{
			Home:       binDir,
			Path:       "test",
			ResultHome: filepath.Join(binDir, "test"),
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			Home:       rootDir("/tmp"),
			Path:       "test",
			ResultHome: rootDir("/tmp/test"),
			ResultData: rootDir("/tmp/data/test"),
		},
		{
			Home:       rootDir("/home"),
			Path:       rootDir("/abc/test"),
			ResultHome: rootDir("/abc/test"),
			ResultData: rootDir("/abc/test"),
		},
	}

	for _, test := range tests {
		cfg := Path{Home: test.Home}
		if err := Paths.initPaths(&cfg); err != nil {
			t.Errorf("error on %+v: %v", test, err)
			continue
		}

		assert.Equal(t, test.ResultHome, Resolve(Home, test.Path), "failed on %+v", test)

		// config path same as home path
		assert.Equal(t, test.ResultHome, Resolve(Config, test.Path), "failed on %+v", test)

		// data path under home path
		assert.Equal(t, test.ResultData, Resolve(Data, test.Path), "failed on %+v", test)
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
	if err != nil {
		t.Fatal(err)
	}

	tests := []io{
		{
			Home:       binDir,
			Data:       "",
			Path:       "test",
			ResultData: filepath.Join(binDir, "data", "test"),
		},
		{
			Home:       rootDir("/tmp"),
			Data:       rootDir("/root"),
			Path:       "test",
			ResultData: rootDir("/root/test"),
		},
		{
			Home:       rootDir("/tmp"),
			Data:       rootDir("root"),
			Path:       rootDir("/var/data"),
			ResultData: rootDir("/var/data"),
		},
	}

	for _, test := range tests {
		cfg := Path{Home: test.Home, Data: test.Data}
		if err := Paths.initPaths(&cfg); err != nil {
			t.Errorf("error on %+v: %v", test, err)
			continue
		}

		assert.Equal(t, test.ResultData, Resolve(Data, test.Path), "failed on %+v", test)
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
	if err != nil {
		t.Fatal(err)
	}

	tests := []io{
		{
			Home:       binDir,
			Logs:       "",
			Path:       "test",
			ResultLogs: filepath.Join(binDir, "logs", "test"),
		},
		{
			Home:       rootDir("/tmp"),
			Logs:       rootDir("/var"),
			Path:       "log",
			ResultLogs: rootDir("/var/log"),
		},
		{
			Home:       rootDir("tmp"),
			Logs:       rootDir("root"),
			Path:       rootDir("/var/log"),
			ResultLogs: rootDir("/var/log"),
		},
	}

	for _, test := range tests {
		cfg := Path{Home: test.Home, Logs: test.Logs}
		if err := Paths.initPaths(&cfg); err != nil {
			t.Errorf("error on %+v: %v", test, err)
			continue
		}

		assert.Equal(t, test.ResultLogs, Resolve(Logs, test.Path))
	}
}

// rootDir builds an OS specific absolute root directory.
func rootDir(path string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(`c:\`, path)
	}
	return filepath.Join("/", path)
}
