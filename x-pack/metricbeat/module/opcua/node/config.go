// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package node

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gopcua/opcua/ua"
)

// Security modes accepted in the configuration, as named by the OPC UA
// specification.
const (
	securityModeNone           = "None"
	securityModeSign           = "Sign"
	securityModeSignAndEncrypt = "SignAndEncrypt"
)

// securityPolicies are the policy names accepted in the configuration. They are
// the short forms of the policy URIs defined by the OPC UA specification.
var securityPolicies = []string{
	"None",
	"Basic128Rsa15",
	"Basic256",
	"Basic256Sha256",
	"Aes128_Sha256_RsaOaep",
	"Aes256_Sha256_RsaPss",
}

type nodeConfig struct {
	ID    string `config:"id" validate:"required"`
	Label string `config:"label"`
}

type config struct {
	Nodes             []nodeConfig `config:"nodes" validate:"required"`
	SecurityPolicy    string       `config:"security_policy"`
	SecurityMode      string       `config:"security_mode"`
	Username          string       `config:"username"`
	Password          string       `config:"password"`
	ClientCertificate string       `config:"client_certificate"`
	ClientKey         string       `config:"client_key"`
	ApplicationURI    string       `config:"application_uri"`
}

func defaultConfig() config {
	return config{
		SecurityPolicy: "None",
		SecurityMode:   securityModeNone,
		ApplicationURI: "urn:elastic:metricbeat:opcua",
	}
}

// Validate checks the configuration before any connection is made.
func (c *config) Validate() error {
	if len(c.Nodes) == 0 {
		return errors.New("at least one node must be configured in nodes")
	}
	for i, n := range c.Nodes {
		if _, err := ua.ParseNodeID(n.ID); err != nil {
			return fmt.Errorf("nodes[%d]: invalid node id %q: %w", i, n.ID, err)
		}
	}

	if !containsFold(securityPolicies, c.SecurityPolicy) {
		return fmt.Errorf("unsupported security_policy %q, use one of %s", c.SecurityPolicy, strings.Join(securityPolicies, ", "))
	}
	switch {
	case strings.EqualFold(c.SecurityMode, securityModeNone):
	case strings.EqualFold(c.SecurityMode, securityModeSign):
	case strings.EqualFold(c.SecurityMode, securityModeSignAndEncrypt):
	default:
		return fmt.Errorf("unsupported security_mode %q, use one of None, Sign, SignAndEncrypt", c.SecurityMode)
	}

	secured := !strings.EqualFold(c.SecurityMode, securityModeNone)
	if secured != !strings.EqualFold(c.SecurityPolicy, "None") {
		return errors.New("security_policy and security_mode must either both be None or both be set")
	}
	if secured && (c.ClientCertificate == "" || c.ClientKey == "") {
		return errors.New("client_certificate and client_key are required when security_mode is Sign or SignAndEncrypt")
	}
	if (c.ClientCertificate == "") != (c.ClientKey == "") {
		return errors.New("client_certificate and client_key must be set together")
	}
	if c.Password != "" && c.Username == "" {
		return errors.New("password is set but username is empty")
	}
	return nil
}

func containsFold(values []string, s string) bool {
	for _, v := range values {
		if strings.EqualFold(v, s) {
			return true
		}
	}
	return false
}
