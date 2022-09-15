// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"context"
	"fmt"
	"net/url"
	"time"

	v2 "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/state"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/azureblobstorage/types"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

type azurebsInput struct {
	config     config
	serviceURL string
}

const (
	inputName string = "azureblobstorage"
)

func Plugin(log *logp.Logger, store cursor.StateStore) v2.Plugin {
	return v2.Plugin{
		Name:       inputName,
		Stability:  feature.Experimental,
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

func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	var sources []cursor.Source
	for _, c := range config.Containers {
		container := tryOverrideOrDefault(config, c)
		sources = append(sources, &types.Source{
			AccountName:   config.AccountName,
			ContainerName: c.Name,
			MaxWorkers:    *container.MaxWorkers,
			Poll:          *container.Poll,
			PollInterval:  *container.PollInterval,
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

// tryOverrideOrDefault , overrides global values with local
// container level values present. If both global & local values
// are absent , assigns default values
func tryOverrideOrDefault(cfg config, c container) container {
	if c.MaxWorkers == nil && cfg.MaxWorkers != nil {
		c.MaxWorkers = cfg.MaxWorkers
	} else if c.MaxWorkers == nil && cfg.MaxWorkers == nil {
		workers := 1
		c.MaxWorkers = &workers
	}

	if c.Poll == nil && cfg.Poll != nil {
		c.Poll = cfg.Poll
	} else if c.Poll == nil && cfg.Poll == nil {
		poll := false
		c.Poll = &poll
	}

	if c.PollInterval == nil && cfg.PollInterval != nil {
		c.PollInterval = cfg.PollInterval
	} else if c.PollInterval == nil && cfg.PollInterval == nil {
		interval := time.Second * 120
		c.PollInterval = &interval
	}

	return c
}

func (input *azurebsInput) Name() string {
	return inputName
}

func (input *azurebsInput) Test(src cursor.Source, ctx v2.TestContext) error {
	return nil
}

func (input *azurebsInput) Run(inputCtx v2.Context, src cursor.Source, cursor cursor.Cursor, publisher cursor.Publisher) error {
	var cp *state.Checkpoint
	st := state.NewState()
	currentSource := src.(*types.Source)

	log := inputCtx.Logger.With("account_name", currentSource.AccountName).With("container", currentSource.ContainerName)
	log.Infof("Running azure blob storage for account: %s", input.config.AccountName)

	if !cursor.IsNew() {
		if err := cursor.Unpack(&cp); err != nil {
			return err
		}

		st.SetCheckpoint(cp)
	}

	ctx, cancelInputCtx := context.WithCancel(context.Background())
	go func() {
		defer cancelInputCtx()
		select {
		case <-inputCtx.Cancelation.Done():
		case <-ctx.Done():
		}
	}()

	serviceClient, credential, err := fetchServiceClientAndCreds(input.config, input.serviceURL, log)
	if err != nil {
		return err
	}
	containerClient, err := fetchContainerClient(serviceClient, currentSource.ContainerName, log)
	if err != nil {
		return err
	}

	scheduler := NewAzureInputScheduler(publisher, containerClient, credential, currentSource, &input.config, st, input.serviceURL, log)
	err = scheduler.Schedule(ctx)
	if err != nil {
		return err
	}

	return nil
}
