// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

// DefaultCheckTimeout is the default timeout used to check if any host information has changed.
const DefaultCheckTimeout = 5 * time.Minute

func init() {
	composable.Providers.AddContextProvider("host", ContextProviderBuilder)
}

type infoFetcher func() (map[string]interface{}, error)

type contextProvider struct {
	logger *logger.Logger

	// used by testing
	checkTimeout time.Duration
	fetcher      infoFetcher
}

// Run runs the environment context provider.
func (c *contextProvider) Run(comm composable.ContextProviderComm) error {
	current, err := c.fetcher()
	if err != nil {
		return err
	}
	comm.Set(current)

	// Update context when any host information changes.
	go func() {
		for {
			select {
			case <-comm.Done():
				return
			case <-time.After(c.checkTimeout):
				break
			}

			updated, err := c.fetcher()
			if err != nil {
				c.logger.Warnf("Failed fetching latest host information: %s", err)
				continue
			}
			if reflect.DeepEqual(current, updated) {
				// nothing to do
				continue
			}
			current = updated
			comm.Set(updated)
		}
	}()

	return nil
}

// ContextProviderBuilder builds the context provider.
func ContextProviderBuilder(c *config.Config) (composable.ContextProvider, error) {
	logger, err := logger.New("dynamic.providers.host")
	if err != nil {
		return nil, err
	}
	return &contextProvider{logger, DefaultCheckTimeout, getHostInfo}, nil
}

func getHostInfo() (map[string]interface{}, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return nil, err
	}
	sysInfo, err := sysinfo.Host()
	if err != nil {
		return nil, err
	}
	info := sysInfo.Info()
	return map[string]interface{}{
		"id":           info.UniqueID,
		"name":         hostname,
		"platform":     runtime.GOOS,
		"architecture": info.Architecture,
		"ip":           info.IPs,
		"mac":          info.MACs,
	}, nil
}
