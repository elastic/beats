package cfgfile

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/stretchr/testify/assert"
)

type mockRunner struct{}
type mockRunnerFactory struct{}

func TestReloader(t *testing.T) {
	// Create random temp directory
	dir, err := ioutil.TempDir("", "libbeat-reloader")
	defer os.RemoveAll(dir)
	if err != nil {
		t.Fatal(err)
	}
	glob := dir + "/*.yml"

	config := common.MustNewConfigFrom(common.MapStr{
		"path": glob,
		"reload": common.MapStr{
			"period":  "1s",
			"enabled": true,
		},
	})
	// common.Config{}
	reloader := NewReloader(nil, config)
	retryCount := 5

	go reloader.Run(nil)
	defer reloader.Stop()

	// wait until configScans >= 2 (which should happen after ~1 second)
	for i := 0; i < retryCount; i++ {
		if configScans.Get() >= 2 {
			break
		}
		// time interval is slightly more than a second so we don't slightly
		// undershoot the first iteration and wait a whole extra second.
		time.Sleep(1100 * time.Millisecond)
	}
	if configScans.Get() < 2 {
		assert.Fail(t, "Timed out waiting for configScans >= 2")
	}

	// The first scan should cause a reload, but additional ones should not,
	// so configReloads should still be 1.
	assert.Equal(t, int64(1), configReloads.Get())

	// Write a file to the reloader path to trigger a real reload
	content := []byte("test\n")
	err = ioutil.WriteFile(dir+"/config1.yml", content, 0644)
	assert.NoError(t, err)

	// Wait for the number of scans to increase at least twice. This is somewhat
	// pedantic, but if we just wait for the next scan, it's possible to wake up
	// during the brief interval after configScans is updated but before
	// configReloads is, giving a false negative. Waiting two iterations
	// guarantees that the change from the first one has taken effect.
	targetScans := configScans.Get() + 2
	for i := 0; i < retryCount; i++ {
		time.Sleep(time.Second)
		if configScans.Get() >= targetScans {
			break
		}
	}
	if configScans.Get() < targetScans {
		assert.Fail(t,
			fmt.Sprintf("Timed out waiting for configScans >= %d", targetScans))
	}

	// The number of reloads should now have increased. It would be nicer to
	// check if the value is exactly 2, but we can't guarantee this: the glob
	// watcher includes an extra 1-second margin around the real modification
	// time, so changes that fall too close to a scan interval can be detected
	// twice.
	if configReloads.Get() < 2 {
		assert.Fail(t,
			fmt.Sprintf(
				"Reloader performed %d scans but only reloaded once",
				configScans.Get()))
	}
}

func (rf *mockRunnerFactory) Create(
	p beat.Pipeline, config *common.Config, meta *common.MapStrPointer,
) (Runner, error) {
	return &mockRunner{}, nil
}

func (r *mockRunner) String() string {
	return "mockRunner{}"
}
func (r *mockRunner) Start() {}
func (r *mockRunner) Stop()  {}
