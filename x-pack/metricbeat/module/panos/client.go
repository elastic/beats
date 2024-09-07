// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panos

import (
	"flag"

	"github.com/PaloAltoNetworks/pango"
)

// PanosClient interface with an Op function
type PanosClient interface {
	Op(req interface{}, vsys string, extras interface{}, ans interface{}) ([]byte, error)
}

type PanosFirewallClient struct {
	pango.Firewall
}

type PanosTestClient struct {
}

// Op is a mock function for testing that returns sample XML output based on the initial req parameter
// XML output is stored in the testdata directory, one file per query string
func (c *PanosTestClient) Op(req interface{}, vsys string, extras, ans interface{}) ([]byte, error) {
	return nil, nil
}

func GetPanosClient(config *Config) (PanosClient, error) {
	// If running tests, return a test client
	if flag.Lookup("test.v") != nil {
		return &PanosTestClient{}, nil
	}

	firewall := pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey}}
	err := firewall.Initialize()
	if err != nil {
		return nil, err
	}
	// Instantiate PanosFirewallClient
	return &PanosFirewallClient{Firewall: firewall}, nil

}
