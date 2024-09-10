// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package panw

import (
	"flag"
	"fmt"

	"github.com/PaloAltoNetworks/pango"
)

// Vsys is the virtual system to query. If empty, the default vsys is used. This is a placeholder for future use,
// as the module currently only supports the default vsys.
const Vsys = ""

// PanwClient interface with an Op function
type PanwClient interface {
	Op(req interface{}, vsys string, extras interface{}, ans interface{}) ([]byte, error)
}

type PanwFirewallClient struct {
	pango.Firewall
}

type PanwTestClient struct {
}

// Op is a mock function for testing that returns sample XML output based on the initial req parameter
// XML output is stored in the testdata directory, one file per query string
func (c *PanwTestClient) Op(req interface{}, vsys string, extras, ans interface{}) ([]byte, error) {
	return nil, nil
}

func GetPanwClient(config *Config) (PanwClient, error) {
	// If running tests, return a test client
	if flag.Lookup("test.v") != nil {
		return &PanwTestClient{}, nil
	}

	firewall := pango.Firewall{Client: pango.Client{Hostname: config.HostIp, ApiKey: config.ApiKey, Port: config.Port}}
	err := firewall.Initialize()
	if err != nil {
		return nil, fmt.Errorf("error initializing firewall client: %w", err)
	}
	// Instantiate panwFirewallClient
	return &PanwFirewallClient{Firewall: firewall}, nil

}
