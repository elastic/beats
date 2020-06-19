package main

import (
	"context"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/elastic/beats/v7/libbeat/common"
)

// osSignalContext creates a context.Context that will be cancelled if the
// configured os signals are received. osSignalContext exits the process
// immediately with error code 3 if the signal is received a second time.
// Calling the cancel function triggers the context cancellation and stops
// the os signal listener.
func osSignalContext(sigs ...os.Signal) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	go func() {
		defer func() {
			signal.Stop(ch)
			cancel()
		}()

		select {
		case <-ctx.Done():
			return
		case <-ch:
			cancel()
			// force shutdown in case we receive another signal
			<-ch
			os.Exit(3)
		}
	}()

	signal.Notify(ch, sigs...)
	return ctx, cancel
}

func readConfigFiles(paths pathSettings, files []string, strictPerm bool) (*common.Config, error) {
	configFilePaths := make([]string, len(files))
	for i, path := range files {
		if !filepath.IsAbs(path) {
			path = filepath.Join(paths.Config, path)
		}
		configFilePaths[i] = path
	}

	if strictPerm {
		for _, path := range configFilePaths {
			if err := common.OwnerHasExclusiveWritePerms(path); err != nil {
				return nil, err
			}
		}
	}

	config := common.NewConfig()
	for _, path := range configFilePaths {
		contents, err := ioutil.ReadFile(path)
		if err != nil {
			return nil, err
		}

		fileConfig, err := common.NewConfigWithYAML(contents, path)
		if err != nil {
			return nil, err
		}

		if err = config.Merge(fileConfig); err != nil {
			return nil, err
		}
	}

	return config, nil
}
