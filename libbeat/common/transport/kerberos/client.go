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

package kerberos

import (
	"fmt"
	"net/http"

	krbclient "gopkg.in/jcmturner/gokrb5.v7/client"
	krbconfig "gopkg.in/jcmturner/gokrb5.v7/config"
	"gopkg.in/jcmturner/gokrb5.v7/keytab"
	"gopkg.in/jcmturner/gokrb5.v7/spnego"
)

type Client struct {
	spClient *spnego.Client
}

func NewClient(config *Config, httpClient *http.Client, esurl string) (*Client, error) {
	var krbClient *krbclient.Client
	krbConf, err := krbconfig.Load(config.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("error creating Kerberos client: %+v", err)
	}

	switch config.AuthType {
	case authKeytab:
		kTab, err := keytab.Load(config.KeyTabPath)
		if err != nil {
			return nil, fmt.Errorf("cannot load keytab file %s: %+v", config.KeyTabPath, err)
		}
		krbClient = krbclient.NewClientWithKeytab(config.Username, config.Realm, kTab, krbConf)
	case authPassword:
		krbClient = krbclient.NewClientWithPassword(config.Username, config.Realm, config.Password, krbConf)
	default:
		return nil, InvalidAuthType
	}

	return &Client{
		spClient: spnego.NewClient(krbClient, httpClient, ""),
	}, nil
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.spClient.Do(req)
}

func (c *Client) CloseIdleConnections() {
	c.spClient.CloseIdleConnections()
}
