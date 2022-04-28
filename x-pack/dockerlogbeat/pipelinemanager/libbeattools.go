// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/gofrs/uuid"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/file"
)

// load pipeline starts up a new pipeline with the given config
func loadNewPipeline(logOptsConfig ContainerOutputConfig, hostname string, log *logp.Logger) (*Pipeline, error) {
	cfg, err := logOptsConfig.CreateConfig()
	if err != nil {
		return nil, err
	}

	// Attach CloudID config if needed
	err = cloudid.OverwriteSettings(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating CloudID: %w", err)
	}

	config := containerConfig{}
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("unpacking config failed: %w", err)
	}

	info, err := getBeatInfo(logOptsConfig, hostname)
	if err != nil {
		return nil, err
	}

	processing, err := processing.MakeDefaultBeatSupport(true)(info, log, cfg)
	if err != nil {
		return nil, fmt.Errorf("error in MakeDefaultSupport: %w", err)
	}

	pipelineCfg := pipeline.Config{}
	err = cfg.Unpack(&pipelineCfg)
	if err != nil {
		return nil, fmt.Errorf("error unpacking pipeline config: %w", err)
	}

	idxMgr := newIndexSupporter(info)

	settings := pipeline.Settings{
		WaitClose:     time.Second * 10,
		WaitCloseMode: pipeline.WaitOnPipelineClose,
		Processors:    processing,
	}

	pipeline, err := pipeline.LoadWithSettings(
		info,
		pipeline.Monitors{
			Metrics:   nil,
			Telemetry: nil,
			Logger:    log,
		},
		pipelineCfg,
		func(stat outputs.Observer) (string, outputs.Group, error) {
			cfg := config.Output
			out, err := outputs.Load(idxMgr, info, stat, cfg.Name(), cfg.Config())
			return cfg.Name(), out, err
		},
		settings,
	)

	if err != nil {
		return nil, fmt.Errorf("error in pipeline.Load")
	}

	return &Pipeline{pipeline: pipeline, refCount: 0}, nil
}

// getBeatInfo returns the beat.Info type needed to start the pipeline
func getBeatInfo(pluginOpts ContainerOutputConfig, hostname string) (beat.Info, error) {
	vers := version.GetDefaultVersion()

	eid, err := uuid.NewV4()
	if err != nil {
		return beat.Info{}, fmt.Errorf("error creating ephemeral ID: %w", err)
	}

	id, err := loadMeta("/tmp/meta.json")
	if err != nil {
		return beat.Info{}, fmt.Errorf("error loading UUID: %w", err)
	}

	beatName := "elastic-log-driver"

	info := beat.Info{
		Beat:        beatName,
		Name:        pluginOpts.BeatName,
		IndexPrefix: "logs-docker",
		Hostname:    hostname,
		Version:     vers,
		EphemeralID: eid,
		ID:          id,
	}

	return info, nil

}

// loadMeta loads the metadata file that contains the UUID
func loadMeta(metaPath string) (uuid.UUID, error) {
	type meta struct {
		UUID uuid.UUID `json:"uuid"`
	}
	// check for an existing file
	f, err := openRegular(metaPath)
	if err != nil && !os.IsNotExist(err) {
		return uuid.Nil, fmt.Errorf("beat meta file %s failed to open: %w", metaPath, err)
	}

	//return the UUID if it exists
	if err == nil {
		m := meta{}
		if err := json.NewDecoder(f).Decode(&m); err != nil && err != io.EOF {
			f.Close()
			return uuid.Nil, fmt.Errorf("error reading %s: %w", metaPath, err)
		}

		f.Close()
		if m.UUID != uuid.Nil {
			return m.UUID, nil
		}
	}

	// file does not exist or ID is invalid, let's create a new one
	newID, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, fmt.Errorf("error creating ID: %w", err)
	}
	// write temporary file first
	tempFile := metaPath + ".new"
	f, err = os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return uuid.Nil, fmt.Errorf("failed to create Beat meta file at %s: %w", tempFile, err)
	}

	encodeErr := json.NewEncoder(f).Encode(meta{UUID: newID})
	err = f.Sync()
	if err != nil {
		return uuid.Nil, fmt.Errorf("beat meta file at %s failed to write: %w", tempFile, err)
	}

	err = f.Close()
	if err != nil {
		return uuid.Nil, fmt.Errorf("beat meta file at %s failed to close: %w", tempFile, err)
	}

	if encodeErr != nil {
		return uuid.Nil, fmt.Errorf("beat meta file at %s failed to write: %w", tempFile, err)
	}

	// move temporary file into final location
	err = file.SafeFileRotate(metaPath, tempFile)
	if err != nil {
		return uuid.Nil, fmt.Errorf("error rotating file to %s: %w", metaPath, err)
	}
	return newID, nil
}

// openRegular is a wrapper to handle a file based on a path
func openRegular(filename string) (*os.File, error) {
	f, err := os.Open(filename)
	if err != nil {
		return f, err
	}

	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}

	if !info.Mode().IsRegular() {
		f.Close()
		if info.IsDir() {
			return nil, err
		}
		return nil, err
	}

	return f, nil
}
