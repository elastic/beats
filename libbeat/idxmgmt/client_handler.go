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

package idxmgmt

import (
	"fmt"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/idxmgmt/lifecycle"
	"github.com/elastic/beats/v7/libbeat/template"
	"github.com/elastic/elastic-agent-libs/version"
)

// ClientHandler defines the interface between a remote service and the Manager for ILM and templates.
type ClientHandler interface {
	lifecycle.ClientHandler
	template.Loader
}

type clientHandler struct {
	lifecycle.ClientHandler
	template.Loader
}

// ESClient defines the minimal interface required for the index manager to
// prepare an index.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() version.V
	IsServerless() bool
}

// FileClient defines the minimal interface required for the Loader to
// prepare a policy and write alias.
type FileClient interface {
	GetVersion() version.V
	Write(component string, name string, body string) error
}

// NewClientHandler initializes and returns a new instance of ClientHandler
func NewClientHandler(ilm lifecycle.ClientHandler, template template.Loader) ClientHandler {
	return &clientHandler{ilm, template}
}

// NewESClientHandler returns a new ESLoader instance,
// initialized with an ilm and template client handler based on the passed in client.
func NewESClientHandler(client ESClient, info beat.Info, cfg lifecycle.RawConfig) (ClientHandler, error) {
	esHandler, err := lifecycle.NewESClientHandler(client, info, cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating ES handler: %w", err)
	}
	loader, err := template.NewESLoader(client, esHandler)
	if err != nil {
		return nil, fmt.Errorf("error creating ES loader: %w", err)
	}
	return NewClientHandler(esHandler, loader), nil
}

// NewFileClientHandler returns a new ESLoader instance,
// initialized with an ilm and template client handler based on the passed in client.
func NewFileClientHandler(client FileClient, info beat.Info, cfg lifecycle.RawConfig) (ClientHandler, error) {
	mgmt, err := lifecycle.NewFileClientHandler(client, info, cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating client handler: %w", err)
	}
	return NewClientHandler(mgmt, template.NewFileLoader(client, mgmt.Mode() == lifecycle.DSL)), nil
}
