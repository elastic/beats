// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"bytes"
	"io"
	"net/http"
	"net/url"

	"gopkg.in/yaml.v2"

	"github.com/elastic/beats/agent/kibana"
	"github.com/elastic/beats/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/application/info"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
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
	UserProvidedMetadata map[string]interface{}
	EnrollAPIKey         string
}

func (e *EnrollCmdOption) KibanaConfig() (*kibana.Config, error) {
	cfg, err := kibana.NewConfigFromURL(e.URL)
	if err != nil {
		return nil, err
	}

	// Add any SSL options from the CLI.
	if len(e.CAs) > 0 || len(e.CASha256) > 0 {
		cfg.TLS = &tlscommon.Config{
			CAs:      e.CAs,
			CASha256: e.CASha256,
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
		storage.NewEncryptedDiskStore(fleetAgentConfigPath(), []byte("")),
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

	cfg, err := options.KibanaConfig()
	if err != nil {
		return nil, errors.New(err,
			"invalid Kibana configuration",
			errors.TypeConfig,
			errors.M(errors.MetaKeyURI, options.URL))
	}

	client, err := fleetapi.NewWithConfig(log, cfg)
	if err != nil {
		return nil, errors.New(err,
			"fail to create the API client",
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
		return errors.New(err, "acquiring hostname")
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

	resp, err := cmd.Execute(r)
	if err != nil {
		return errors.New(err,
			"fail to execute request to Kibana",
			errors.TypeNetwork)
	}

	fleetConfig, err := createFleetConfigFromEnroll(&APIAccess{
		AccessAPIKey: resp.Item.AccessAPIKey,
		Kibana:       c.kibanaConfig,
	})

	reader, err := yamlToReader(fleetConfig)
	if err != nil {
		return err
	}

	if err := c.configStore.Save(reader); err != nil {
		return errors.New(err, "could not save enrollment information", errors.TypeFilesystem)
	}

	if _, err := info.NewAgentInfo(); err != nil {
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
