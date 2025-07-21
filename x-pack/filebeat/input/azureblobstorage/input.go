// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azureblobstorage

import (
	"context"
	"fmt"
	"net/url"
	"reflect"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"

	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type azurebsInput struct {
	config     config
	serviceURL string
}

// defines the valid range for Unix timestamps for 64 bit integers
var (
	minTimestamp = time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	maxTimestamp = time.Date(3000, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
)

const (
	inputName string = "azure-blob-storage"
)

func Plugin(log *logp.Logger, store statestore.States) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "Azure Blob Storage logs",
		Doc:        "Collect logs from Azure Blob Storage Service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       inputName,
			Configure:  configure,
		},
	}
}

func configure(cfg *conf.C, _ *logp.Logger) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	//nolint:prealloc // No need to preallocate the slice here
	var sources []cursor.Source
	// This is to maintain backward compatibility with the old config.
	if config.BatchSize == 0 {
		config.BatchSize = *config.MaxWorkers
	}
	for _, c := range config.Containers {
		container := tryOverrideOrDefault(config, c)
		if container.TimeStampEpoch != nil && !isValidUnixTimestamp(*container.TimeStampEpoch) {
			return nil, nil, fmt.Errorf("invalid timestamp epoch: %d", *container.TimeStampEpoch)
		}
		sources = append(sources, &Source{
			AccountName:              config.AccountName,
			ContainerName:            c.Name,
			BatchSize:                *container.BatchSize,
			MaxWorkers:               *container.MaxWorkers,
			Poll:                     *container.Poll,
			PollInterval:             *container.PollInterval,
			TimeStampEpoch:           container.TimeStampEpoch,
			ExpandEventListFromField: container.ExpandEventListFromField,
			FileSelectors:            container.FileSelectors,
			ReaderConfig:             container.ReaderConfig,
			PathPrefix:               container.PathPrefix,
		})
	}

	var urL string
	if len(config.StorageURL) != 0 {
		if _, err := url.ParseRequestURI(config.StorageURL); err != nil {
			return nil, nil, fmt.Errorf("error parsing url : %w", err)
		}
		urL = config.StorageURL
	} else {
		urL = "https://" + config.AccountName + ".blob.core.windows.net/"
	}
	return sources, &azurebsInput{config: config, serviceURL: urL}, nil
}

// tryOverrideOrDefault, overrides global values with local
// container level values present. If both global & local values
// are absent, assigns default values
func tryOverrideOrDefault(cfg config, c container) container {
	if c.MaxWorkers == nil {
		maxWorkers := 1
		if cfg.MaxWorkers != nil {
			maxWorkers = *cfg.MaxWorkers
		}
		c.MaxWorkers = &maxWorkers
	}

	if c.BatchSize == nil {
		if cfg.BatchSize != 0 {
			// If the global batch size is set, use it
			c.BatchSize = &cfg.BatchSize
		} else {
			// If the global batch size is not set, use the local max_workers as the batch size
			// since at this point we know `c.MaxWorkers` will be set to a non-nil value.
			// This is to maintain backward compatibility with the old config.
			c.BatchSize = c.MaxWorkers
		}
	}

	if c.Poll == nil {
		var poll bool
		if cfg.Poll != nil {
			poll = *cfg.Poll
		}
		c.Poll = &poll
	}

	if c.PollInterval == nil {
		interval := time.Second * 300
		if cfg.PollInterval != nil {
			interval = *cfg.PollInterval
		}
		c.PollInterval = &interval
	}

	if c.TimeStampEpoch == nil {
		c.TimeStampEpoch = cfg.TimeStampEpoch
	}

	if c.ExpandEventListFromField == "" {
		c.ExpandEventListFromField = cfg.ExpandEventListFromField
	}

	if len(c.FileSelectors) == 0 && len(cfg.FileSelectors) != 0 {
		c.FileSelectors = cfg.FileSelectors
	}
	// If the container level ReaderConfig matches the default config ReaderConfig state,
	// use the global ReaderConfig. Matching the default ReaderConfig state
	// means that the container level ReaderConfig is not set, and we should use the
	// global ReaderConfig. Partial definition of ReaderConfig at both the global
	// and container level is not supported, it's an either or scenario.
	if reflect.DeepEqual(c.ReaderConfig, defaultReaderConfig) {
		c.ReaderConfig = cfg.ReaderConfig
	}

	// If the container level PathPrefix is empty, use the global PathPrefix.
	if c.PathPrefix == "" {
		c.PathPrefix = cfg.PathPrefix
	}

	return c
}

// isValidUnixTimestamp checks if the timestamp is a valid Unix timestamp
func isValidUnixTimestamp(timestamp int64) bool {
	// checks if the timestamp is within the valid range
	return minTimestamp <= timestamp && timestamp <= maxTimestamp
}

func (input *azurebsInput) Name() string {
	return inputName
}

func (input *azurebsInput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (input *azurebsInput) Run(inputCtx v2.Context, src cursor.Source, cursor cursor.Cursor, publisher cursor.Publisher) error {
	var cp *Checkpoint
	st := newState()
	if !cursor.IsNew() {
		if err := cursor.Unpack(&cp); err != nil {
			return err
		}
		st.setCheckpoint(cp)
	}
	return input.run(inputCtx, src, st, publisher)
}

func (input *azurebsInput) run(inputCtx v2.Context, src cursor.Source, st *state, publisher cursor.Publisher) error {
	currentSource := src.(*Source)

	stat := inputCtx.StatusReporter
	if stat == nil {
		stat = noopReporter{}
	}
	stat.UpdateStatus(status.Starting, "")
	stat.UpdateStatus(status.Configuring, "")

	log := inputCtx.Logger.With("account_name", currentSource.AccountName).With("container_name", currentSource.ContainerName)
	log.Infof("Running azure blob storage for account: %s", input.config.AccountName)
	// create a new inputMetrics instance
	metrics := newInputMetrics(inputCtx.ID+":"+currentSource.ContainerName, nil)
	metrics.url.Set(input.serviceURL + currentSource.ContainerName)
	defer metrics.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-inputCtx.Cancelation.Done()
		stat.UpdateStatus(status.Stopping, "")
		cancel()
	}()

	serviceClient, credential, err := fetchServiceClientAndCreds(input.config, input.serviceURL, log)
	if err != nil {
		metrics.errorsTotal.Inc()
		stat.UpdateStatus(status.Failed, "failed to get service client: "+err.Error())
		return err
	}
	containerClient, err := fetchContainerClient(serviceClient, currentSource.ContainerName, log)
	if err != nil {
		metrics.errorsTotal.Inc()
		stat.UpdateStatus(status.Failed, "failed to get container client: "+err.Error())
		return err
	}

	scheduler := newScheduler(publisher, containerClient, credential, currentSource, &input.config, st, input.serviceURL, stat, metrics, log)
	return scheduler.schedule(ctx)
}

type noopReporter struct{}

func (noopReporter) UpdateStatus(status.Status, string) {}
