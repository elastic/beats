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
	"github.com/elastic/beats/x-pack/agent/pkg/agent/application/info"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/storage"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

type store interface {
	Save(io.Reader) error
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
	log                  *logger.Logger
	enrollAPIKey         string
	client               clienter
	id                   string
	userProvidedMetadata map[string]interface{}
	configStore          store
	kibanaConfig         *kibana.Config
}

// NewEnrollCmd creates a new enroll command that will registers the current beats to the remote
// system.
func NewEnrollCmd(
	log *logger.Logger,
	url string,
	CAs []string,
	enrollAPIKey string,
	id string,
	userProvidedMetadata map[string]interface{},
	configPath string,
) (*EnrollCmd, error) {

	store := storage.NewReplaceOnSuccessStore(
		configPath,
		DefaultAgentFleetConfig,
		storage.NewEncryptedDiskStore(fleetAgentConfigPath(), []byte("")),
	)

	return NewEnrollCmdWithStore(
		log,
		url,
		CAs,
		enrollAPIKey,
		id,
		userProvidedMetadata,
		configPath,
		store,
	)
}

//NewEnrollCmdWithStore creates an new enrollment and accept a custom store.
func NewEnrollCmdWithStore(
	log *logger.Logger,
	url string,
	CAs []string,
	enrollAPIKey string,
	id string,
	userProvidedMetadata map[string]interface{},
	configPath string,
	store store,
) (*EnrollCmd, error) {
	cfg, err := kibana.NewConfigFromURL(url, CAs)
	if err != nil {
		return nil, errors.New(err,
			"invalid Kibana URL",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, url))
	}

	client, err := fleetapi.NewWithConfig(log, cfg)
	if err != nil {
		return nil, errors.New(err,
			"fail to create the API client",
			errors.TypeNetwork,
			errors.M(errors.MetaKeyURI, url))
	}

	if userProvidedMetadata == nil {
		userProvidedMetadata = make(map[string]interface{})
	}

	// Extract the token
	// Create the kibana client
	return &EnrollCmd{
		log:                  log,
		client:               client,
		enrollAPIKey:         enrollAPIKey,
		id:                   id,
		userProvidedMetadata: userProvidedMetadata,
		kibanaConfig:         cfg,
		configStore:          store,
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
		EnrollAPIKey: c.enrollAPIKey,
		SharedID:     c.id,
		Type:         fleetapi.PermanentEnroll,
		Metadata: fleetapi.Metadata{
			Local:        metadata,
			UserProvided: c.userProvidedMetadata,
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
		return errors.New(err, "could not save enroll credentials", errors.TypeFilesystem)
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
