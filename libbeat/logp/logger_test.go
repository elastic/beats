package logp_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"

	"github.com/elastic/beats/libbeat/logp"
)

func TestZap(t *testing.T) {
	logp.DevelopmentSetup()
	defer logp.Sync()
	log := logp.NewLogger("beats-test")
	log.Info("hello world", zap.Int("one", 2))
}

func TestCustomSetup(t *testing.T) {
	dir, err := ioutil.TempDir("", "file_rotator")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	c := logp.Config{
		ToStderr: true,
		ToFiles:  true,
		ToSyslog: true,
		Files: logp.FileConfig{
			Path:             dir,
			Name:             "test.log",
			KeepFiles:        1,
			RotateEveryBytes: 1024 * 1024,
			Permissions:      0600,
		},
		JSON: true,
	}
	if err = logp.CustomSetup(c); err != nil {
		t.Fatal(err)
	}

	log := logp.NewLogger("beats-test")
	log.Info("hello world", zap.Int("one", 3))
	log.Info("hello world", zap.Int("one", 4))
	logp.Sync()

	_, err = ioutil.ReadFile(filepath.Join(dir, "test.log"))
	if err != nil {
		t.Fatal(err)
	}
	//t.Log(string(fileContents))
}
