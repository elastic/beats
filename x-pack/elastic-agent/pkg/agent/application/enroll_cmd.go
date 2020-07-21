// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/storage"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/fleetapi"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/kibana"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/release"
)

type store interface {
	Save(io.Reader) error
}

type storeLoad interface {
	store
	Load() (io.ReadCloser, error)
}

type clienter interface {
	Send(
		ctx context.Context,
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)

	URI() string
}

// EnrollCmd is an enroll subcommand that interacts between the Kibana API and the Agent.
type EnrollCmd struct {
	log          *logger.Logger
	options      *EnrollCmdOption
	client       clienter
	configStore  store
	kibanaConfig *kibana.Config
}

// EnrollCmdOption define all the supported enrollment option.
type EnrollCmdOption struct {
	ID                   string
	URL                  string
	CAs                  []string
	CASha256             []string
	Insecure             bool
	UserProvidedMetadata map[string]interface{}
	EnrollAPIKey         string
	Staging              string
}

func (e *EnrollCmdOption) kibanaConfig() (*kibana.Config, error) {
	cfg, err := kibana.NewConfigFromURL(e.URL)
	if err != nil {
		return nil, err
	}
	if cfg.Protocol == kibana.ProtocolHTTP && !e.Insecure {
		return nil, fmt.Errorf("connection to Kibana is insecure, strongly recommended to use a secure connection (override with --insecure)")
	}

	// Add any SSL options from the CLI.
	if len(e.CAs) > 0 || len(e.CASha256) > 0 {
		cfg.TLS = &tlscommon.Config{
			CAs:      e.CAs,
			CASha256: e.CASha256,
		}
	}
	if e.Insecure {
		cfg.TLS = &tlscommon.Config{
			VerificationMode: tlscommon.VerifyNone,
		}
	}

	return cfg, nil
}

// NewEnrollCmd creates a new enroll command that will registers the current beats to the remote
// system.
func NewEnrollCmd(
	log *logger.Logger,
	options *EnrollCmdOption,
	configPath string,
) (*EnrollCmd, error) {

	store := storage.NewReplaceOnSuccessStore(
		configPath,
		DefaultAgentFleetConfig,
		storage.NewDiskStore(info.AgentConfigFile()),
	)

	return NewEnrollCmdWithStore(
		log,
		options,
		configPath,
		store,
	)
}

//NewEnrollCmdWithStore creates an new enrollment and accept a custom store.
func NewEnrollCmdWithStore(
	log *logger.Logger,
	options *EnrollCmdOption,
	configPath string,
	store store,
) (*EnrollCmd, error) {

	cfg, err := options.kibanaConfig()
	if err != nil {
		return nil, errors.New(
			err, "Error",
			errors.TypeConfig,
			errors.M(errors.MetaKeyURI, options.URL))
	}

	client, err := fleetapi.NewWithConfig(log, cfg)
	if err != nil {
		return nil, errors.New(
			err, "Error",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, options.URL))
	}

	return &EnrollCmd{
		log:          log,
		client:       client,
		options:      options,
		kibanaConfig: cfg,
		configStore:  store,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *EnrollCmd) Execute() error {
	cmd := fleetapi.NewEnrollCmd(c.client)

	metadata, err := metadata()
	if err != nil {
		return errors.New(err, "acquiring metadata failed")
	}

	r := &fleetapi.EnrollRequest{
		EnrollAPIKey: c.options.EnrollAPIKey,
		SharedID:     c.options.ID,
		Type:         fleetapi.PermanentEnroll,
		Metadata: fleetapi.Metadata{
			Local:        metadata,
			UserProvided: c.options.UserProvidedMetadata,
		},
	}

	resp, err := cmd.Execute(context.Background(), r)
	if err != nil {
		return errors.New(err,
			"fail to execute request to Kibana",
			errors.TypeNetwork)
	}

	fleetConfig, err := createFleetConfigFromEnroll(resp.Item.AccessAPIKey, c.kibanaConfig)
	agentConfig := map[string]interface{}{
		"id": resp.Item.ID,
	}
	if c.options.Staging != "" {
		staging := fmt.Sprintf("https://staging.elastic.co/%s-%s/downloads/", release.Version(), c.options.Staging[:8])
		agentConfig["download"] = map[string]interface{}{
			"sourceURI": staging,
		}
	}

	configToStore := map[string]interface{}{
		"fleet": fleetConfig,
		"agent": agentConfig,
	}

	reader, err := yamlToReader(configToStore)
	if err != nil {
		return err
	}

	if err := c.configStore.Save(reader); err != nil {
		return errors.New(err, "could not save enrollment information", errors.TypeFilesystem)
	}

	if _, err := info.NewAgentInfo(); err != nil {
		return err
	}

	// clear action store
	// fail only if file exists and there was a failure
	if err := os.Remove(info.AgentActionStoreFile()); !os.IsNotExist(err) {
		return err
	}

	return nil
}

func yamlToReader(in interface{}) (io.Reader, error) {
	data, err := yaml.Marshal(in)
	if err != nil {
		return nil, errors.New(err, "could not marshal to YAML")
	}
	return bytes.NewReader(data), nil
}
