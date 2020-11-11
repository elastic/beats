// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package atomic

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
)

func TestOKInstall(t *testing.T) {
	sig := make(chan int)
	ti := &testInstaller{sig}
	var wg sync.WaitGroup
	i, err := NewInstaller(ti)
	s := program.Spec{Name: "a", Cmd: "a"}

	assert.NoError(t, err)

	ctx := context.Background()
	installDir := filepath.Join(paths.TempDir(), "install_dir")

	wg.Add(1)
	go func() {
		err := i.Install(ctx, s, "b", installDir)
		assert.NoError(t, err)
		wg.Done()
	}()

	// signal to process next files
	close(sig)

	wg.Wait()

	assert.DirExists(t, installDir)
	files := getFiles()

	for name := range files {
		path := filepath.Join(installDir, name)
		assert.FileExists(t, path)
	}

	os.RemoveAll(installDir)
}

func TestContextCancelledInstall(t *testing.T) {
	sig := make(chan int)
	ti := &testInstaller{sig}
	var wg sync.WaitGroup
	i, err := NewInstaller(ti)
	s := program.Spec{Name: "a", Cmd: "a"}

	assert.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	installDir := filepath.Join(paths.TempDir(), "install_dir")

	wg.Add(1)
	go func() {
		err := i.Install(ctx, s, "b", installDir)
		assert.Error(t, err)
		wg.Done()
	}()

	// cancel before signaling
	cancel()
	close(sig)

	wg.Wait()

	assert.NoDirExists(t, installDir)
}

type testInstaller struct {
	signal chan int
}

func (ti *testInstaller) Install(ctx context.Context, _ program.Spec, _, installDir string) error {
	files := getFiles()
	if err := os.MkdirAll(installDir, 0777); err != nil {
		return err
	}

	for name, content := range files {
		if err := ctx.Err(); err != nil {
			return err
		}

		filename := filepath.Join(installDir, name)
		if err := ioutil.WriteFile(filename, content, 0666); err != nil {
			return err
		}

		// wait for all but last
		<-ti.signal
	}

	return nil
}

func getFiles() map[string][]byte {
	files := make(map[string][]byte)
	fileCount := 3
	for i := 1; i <= fileCount; i++ {
		files[fmt.Sprintf("file_%d", i)] = []byte(fmt.Sprintf("content of file %d", i))
	}

	return files
}
