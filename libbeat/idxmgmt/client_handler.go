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
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/idxmgmt/ilm"
	"github.com/elastic/beats/libbeat/template"
)

// ClientHandler defines the interface between a remote service and the Manager for ILM and templates.
type ClientHandler interface {
	ilm.ClientHandler
	template.Loader
}

type clientHandler struct {
	ilm.ClientHandler
	template.Loader
}

// ESClient defines the minimal interface required for the index manager to
// prepare an index.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

// FileClient defines the minimal interface required for the Loader to
// prepare a policy and write alias.
type FileClient interface {
	GetVersion() common.Version
	Write(name string, body string) error
}

// NewClientHandler initializes and returns a new instance of ClientHandler
func NewClientHandler(ilm ilm.ClientHandler, template template.Loader) ClientHandler {
	return &clientHandler{ilm, template}
}

// NewESClientHandler returns a new ESLoader instance,
// initialized with an ilm and template client handler based on the passed in client.
func NewESClientHandler(c ESClient) ClientHandler {
	return NewClientHandler(ilm.NewESClientHandler(c), template.NewESLoader(c))
}

// NewFileClientHandler returns a new ESLoader instance,
// initialized with an ilm and template client handler based on the passed in client.
func NewFileClientHandler(c FileClient) ClientHandler {
	return NewClientHandler(ilm.NewFileClientHandler(c), template.NewFileLoader(c))
}
