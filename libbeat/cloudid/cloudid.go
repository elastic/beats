// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

// Package cloudid contains functions for parsing the cloud.id and cloud.auth
// settings and modifying the configuration to take them into account.
package cloudid

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

const defaultCloudPort = "443"

// CloudID encapsulates the encoded (i.e. raw) and decoded parts of Elastic Cloud ID.
type CloudID struct {
	id     string
	esURL  string
	kibURL string

	auth     string
	username string
	password string
}

// NewCloudID constructs a new CloudID object by decoding the given cloud ID and cloud auth.
func NewCloudID(cloudID string, cloudAuth string) (*CloudID, error) {
	cid := CloudID{
		id:   cloudID,
		auth: cloudAuth,
	}

	if err := cid.decode(); err != nil {
		return nil, err
	}

	return &cid, nil
}

// ElasticsearchURL returns the Elasticsearch URL decoded from the cloud ID.
func (c *CloudID) ElasticsearchURL() string {
	return c.esURL
}

// KibanaURL returns the Kibana URL decoded from the cloud ID.
func (c *CloudID) KibanaURL() string {
	return c.kibURL
}

// Username returns the username decoded from the cloud auth.
func (c *CloudID) Username() string {
	return c.username
}

// Password returns the password decoded from the cloud auth.
func (c *CloudID) Password() string {
	return c.password
}

func (c *CloudID) decode() error {
	var err error
	if err = c.decodeCloudID(); err != nil {
		return errors.Wrapf(err, "invalid cloud id '%v'", c.id)
	}

	if c.auth != "" {
		if err = c.decodeCloudAuth(); err != nil {
			return errors.Wrap(err, "invalid cloud auth")
		}
	}

	return nil
}

// decodeCloudID decodes the c.id into c.esURL and c.kibURL
func (c *CloudID) decodeCloudID() error {
	cloudID := c.id

	// 1. Ignore anything before `:`.
	idx := strings.LastIndex(cloudID, ":")
	if idx >= 0 {
		cloudID = cloudID[idx+1:]
	}

	// 2. base64 decode
	decoded, err := base64.StdEncoding.DecodeString(cloudID)
	if err != nil {
		return errors.Wrapf(err, "base64 decoding failed on %s", cloudID)
	}

	// 3. separate based on `$`
	words := strings.Split(string(decoded), "$")
	if len(words) < 3 {
		return errors.Errorf("Expected at least 3 parts in %s", string(decoded))
	}

	// 4. extract port from the ES and Kibana host, or use 443 as the default
	host, port := extractPortFromName(words[0], defaultCloudPort)
	esID, esPort := extractPortFromName(words[1], port)
	kbID, kbPort := extractPortFromName(words[2], port)

	// 5. form the URLs
	esURL := url.URL{Scheme: "https", Host: fmt.Sprintf("%s.%s:%s", esID, host, esPort)}
	kibanaURL := url.URL{Scheme: "https", Host: fmt.Sprintf("%s.%s:%s", kbID, host, kbPort)}

	c.esURL = esURL.String()
	c.kibURL = kibanaURL.String()

	return nil
}

// decodeCloudAuth splits the c.auth into c.username and c.password.
func (c *CloudID) decodeCloudAuth() error {
	cloudAuth := c.auth
	idx := strings.Index(cloudAuth, ":")
	if idx < 0 {
		return errors.New("cloud.auth setting doesn't contain `:` to split between username and password")
	}

	c.username = cloudAuth[0:idx]
	c.password = cloudAuth[idx+1:]
	return nil
}

// OverwriteSettings modifies the received config object by overwriting the
// output.elasticsearch.hosts, output.elasticsearch.username, output.elasticsearch.password,
// setup.kibana.host settings based on values derived from the cloud.id and cloud.auth
// settings.
func OverwriteSettings(cfg *config.C) error {

	logger := logp.NewLogger("cloudid")
	cloudID, _ := cfg.String("cloud.id", -1)
	cloudAuth, _ := cfg.String("cloud.auth", -1)

	if cloudID == "" && cloudAuth == "" {
		// nothing to hack
		return nil
	}

	logger.Debugf("cloud.id: %s, cloud.auth: %s", cloudID, cloudAuth)
	if cloudID == "" {
		return errors.New("cloud.auth specified but cloud.id is empty. Please specify both")
	}

	// cloudID overwrites
	cid, err := NewCloudID(cloudID, cloudAuth)
	if err != nil {
		return errors.Errorf("Error decoding cloud.id: %v", err)
	}

	logger.Infof("Setting Elasticsearch and Kibana URLs based on the cloud id: output.elasticsearch.hosts=%s and setup.kibana.host=%s", cid.esURL, cid.kibURL)

	esURLConfig, err := config.NewConfigFrom([]string{cid.ElasticsearchURL()})
	if err != nil {
		return err
	}

	// Before enabling the ES output, check that no other output is enabled
	tmp := struct {
		Output config.Namespace `config:"output"`
	}{}
	if err := cfg.Unpack(&tmp); err != nil {
		return err
	}
	if out := tmp.Output; out.IsSet() && out.Name() != "elasticsearch" {
		return errors.Errorf("The cloud.id setting enables the Elasticsearch output, but you already have the %s output enabled in the config", out.Name())
	}

	err = cfg.SetChild("output.elasticsearch.hosts", -1, esURLConfig)
	if err != nil {
		return err
	}

	err = cfg.SetString("setup.kibana.host", -1, cid.KibanaURL())
	if err != nil {
		return err
	}

	if cloudAuth != "" {
		// cloudAuth overwrites
		err = cfg.SetString("output.elasticsearch.username", -1, cid.Username())
		if err != nil {
			return err
		}

		err = cfg.SetString("output.elasticsearch.password", -1, cid.Password())
		if err != nil {
			return err
		}
	}

	return nil
}

// extractPortFromName takes a string in the form `id:port` and returns the
// ID and the port. If there's no `:`, the default port is returned
func extractPortFromName(word string, defaultPort string) (id, port string) {
	idx := strings.LastIndex(word, ":")
	if idx >= 0 {
		return word[:idx], word[idx+1:]
	}
	return word, defaultPort
}
