// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package host

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"time"

	"github.com/elastic/go-sysinfo"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

// DefaultCheckInterval is the default timeout used to check if any host information has changed.
const DefaultCheckInterval = 5 * time.Minute

func init() {
	composable.Providers.AddContextProvider("host", ContextProviderBuilder)
}

type infoFetcher func() (map[string]interface{}, error)

type contextProvider struct {
	logger *logger.Logger

	CheckInterval time.Duration `config:"check_interval"`

	// used by testing
	fetcher infoFetcher
}

// Run runs the environment context provider.
func (c *contextProvider) Run(comm composable.ContextProviderComm) error {
	current, err := c.fetcher()
	if err != nil {
		return err
	}
	err = comm.Set(current)
	if err != nil {
		return errors.New(err, "failed to set mapping", errors.TypeUnexpected)
	}

	// Update context when any host information changes.
	go func() {
		for {
			select {
			case <-comm.Done():
				return
			case <-time.After(c.CheckInterval):
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
			err = comm.Set(updated)
			if err != nil {
				c.logger.Errorf("Failed updating mapping to latest host information: %s", err)
			}
		}
	}()

	return nil
}

// ContextProviderBuilder builds the context provider.
func ContextProviderBuilder(c *config.Config) (composable.ContextProvider, error) {
	logger, err := logger.New("composable.providers.host")
	if err != nil {
		return nil, err
	}
	p := &contextProvider{
		logger:  logger,
		fetcher: getHostInfo,
	}
	if c != nil {
		err := c.Unpack(p)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack config: %s", err)
		}
	}
	if p.CheckInterval <= 0 {
		p.CheckInterval = DefaultCheckInterval
	}
	return p, nil
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
