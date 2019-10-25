// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package application

import (
	"io"
	"net/http"
	"net/url"
	"runtime"

	"github.com/pkg/errors"

	"github.com/elastic/beats/agent/kibana"
	"github.com/elastic/beats/agent/release"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/fleetapi"
)

type clienter interface {
	Send(
		method string,
		path string,
		params url.Values,
		headers http.Header,
		body io.Reader,
	) (*http.Response, error)
}

// EnrollCmd is an enroll subcommand that interacts between the Kibana API and the Agent.
type EnrollCmd struct {
	log                  *logger.Logger
	enrollmentToken      string
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
	enrollmentToken string,
	id string,
	userProvidedMetadata map[string]interface{},
	configStore store,
) (*EnrollCmd, error) {

	cfg, err := kibana.NewConfigFromURL(url, CAs)
	if err != nil {
		return nil, errors.Wrap(err, "invalid Kibana URL")
	}

	client, err := fleetapi.NewWithConfig(log, cfg)
	if err != nil {
		return nil, errors.Wrap(err, "fail to create the API client")
	}

	if userProvidedMetadata == nil {
		userProvidedMetadata = make(map[string]interface{})
	}

	// Extract the token
	// Create the kibana client
	return &EnrollCmd{
		log:                  log,
		client:               client,
		enrollmentToken:      enrollmentToken,
		id:                   id,
		userProvidedMetadata: userProvidedMetadata,
		kibanaConfig:         cfg,
		configStore:          configStore,
	}, nil
}

// Execute tries to enroll the agent into Fleet.
func (c *EnrollCmd) Execute() error {
	cmd := fleetapi.NewEnrollCmd(c.client)

	r := &fleetapi.EnrollRequest{
		EnrollmentToken: c.enrollmentToken,
		SharedID:        c.id,
		Type:            fleetapi.PermanentEnroll,
		Metadata: fleetapi.Metadata{
			Local:        metadata(),
			UserProvided: c.userProvidedMetadata,
		},
	}

	resp, err := cmd.Execute(r)
	if err != nil {
		return errors.Wrap(err, "fail to execute request to Kibana")
	}

	if err := c.configStore.Save(fleetConfig{
		AccessToken: resp.Item.AccessToken,
		Kibana:      c.kibanaConfig,
	}); err != nil {
		return errors.Wrap(err, "could not save credentials")
	}

	return nil
}

func metadata() map[string]interface{} {
	return map[string]interface{}{
		"platform": runtime.GOOS,
		"version":  release.Version(),
	}
}

// info Enrollment token eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ0eXBlIjoiRU5ST0xNRU5UX1RPS0VOIiwicG9saWN5Ijp7ImlkIjoiNjlmM2Y1YTAtZWM1Mi0xMWU5LTkzYzQtZDcyYWI4YTY5MzkxIn0sImlhdCI6MTU3MDgxNzQzMn0.tVxm4JY9gAcd14YlQTmi_y-8AqbGtKyS_PXXI2gdLBY
