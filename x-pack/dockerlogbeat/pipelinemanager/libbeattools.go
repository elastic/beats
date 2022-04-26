// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package pipelinemanager

import (
	"crypto/sha1"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"time"

	"github.com/gofrs/uuid"
	"github.com/pkg/errors"
	yaml "gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/cloudid"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/beats/v7/libbeat/publisher/pipeline"
	"github.com/elastic/beats/v7/libbeat/publisher/processing"
	"github.com/elastic/beats/v7/libbeat/version"
	"github.com/elastic/elastic-agent-libs/file"
)

// makeConfigHash is the helper function that turns a user config into a hash
func makeConfigHash(cfg map[string]string) string {
	var hashString string
	var orderedVal []string

	for _, val := range cfg {
		orderedVal = append(orderedVal, val)
	}

	sort.Strings(orderedVal)

	for _, val := range orderedVal {
		hashString = hashString + val
	}

	sum := sha1.Sum([]byte(hashString))

	return string(sum[:])
}

// load pipeline starts up a new pipeline with the given config
func loadNewPipeline(logOptsConfig ContainerOutputConfig, hostname string, log *logp.Logger) (*Pipeline, error) {
	cfg, err := logOptsConfig.CreateConfig()
	if err != nil {
		return nil, err
	}

	// Attach CloudID config if needed
	err = cloudid.OverwriteSettings(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "Error creating CloudID")
	}

	config := containerConfig{}
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("unpacking config failed: %v", err)
	}

	info, err := getBeatInfo(logOptsConfig, hostname)
	if err != nil {
		return nil, err
	}

	processing, err := processing.MakeDefaultBeatSupport(true)(info, log, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "error in MakeDefaultSupport")
	}

	pipelineCfg := pipeline.Config{}
	err = cfg.Unpack(&pipelineCfg)
	if err != nil {
		return nil, errors.Wrap(err, "error unpacking pipeline config")
	}

	idxMgr := newIndexSupporter(info)

	settings := pipeline.Settings{
		WaitClose:     time.Duration(time.Second * 10),
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
		return nil, errors.Wrap(err, "error in pipeline.Load")
	}

	return &Pipeline{pipeline: pipeline, refCount: 0}, nil
}

// parseCfgKeys helpfully parses the values in the map, so users can specify yml structures.
func parseCfgKeys(cfg map[string]string) (map[string]interface{}, error) {
	outMap := make(map[string]interface{})

	for cfgKey, strVal := range cfg {
		var parsed interface{}
		if err := yaml.Unmarshal([]byte(strVal), &parsed); err != nil {
			return nil, err
		}
		outMap[cfgKey] = parsed
	}

	return outMap, nil
}

// getBeatInfo returns the beat.Info type needed to start the pipeline
func getBeatInfo(pluginOpts ContainerOutputConfig, hostname string) (beat.Info, error) {
	vers := version.GetDefaultVersion()

	eid, err := uuid.NewV4()
	if err != nil {
		return beat.Info{}, errors.Wrap(err, "error creating ephemeral ID")
	}

	id, err := loadMeta("/tmp/meta.json")
	if err != nil {
		return beat.Info{}, errors.Wrap(err, "error loading UUID")
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
		return uuid.Nil, errors.Wrapf(err, "beat meta file %s failed to open", metaPath)
	}

	//return the UUID if it exists
	if err == nil {
		m := meta{}
		if err := json.NewDecoder(f).Decode(&m); err != nil && err != io.EOF {
			f.Close()
			return uuid.Nil, errors.Wrapf(err, "Error reading %s", metaPath)
		}

		f.Close()
		if m.UUID != uuid.Nil {
			return m.UUID, nil
		}
	}

	// file does not exist or ID is invalid, let's create a new one
	newID, err := uuid.NewV4()
	if err != nil {
		return uuid.Nil, errors.Wrap(err, "error creating ID")
	}
	// write temporary file first
	tempFile := metaPath + ".new"
	f, err = os.OpenFile(tempFile, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "failed to create Beat meta file at %s", tempFile)
	}

	encodeErr := json.NewEncoder(f).Encode(meta{UUID: newID})
	err = f.Sync()
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "beat meta file at %s failed to write", tempFile)
	}

	err = f.Close()
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "beat meta file at %s failed to close", tempFile)
	}

	if encodeErr != nil {
		return uuid.Nil, errors.Wrapf(err, "beat meta file at %s failed to write", tempFile)
	}

	// move temporary file into final location
	err = file.SafeFileRotate(metaPath, tempFile)
	if err != nil {
		return uuid.Nil, errors.Wrapf(err, "error rotating file to %s", metaPath)
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
