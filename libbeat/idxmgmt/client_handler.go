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
	ILMClient() ilm.ClientHandler
	TemplateClient() template.ClientHandler
}

// ESClientHandler implements the ClientHandler interface for talking to ES.
type ESClientHandler struct {
	client ESClient
}

// ILMClient returns ESClientHandler for ILM handling.
func (c *ESClientHandler) ILMClient() ilm.ClientHandler {
	return ilm.NewESClientHandler(c.client)
}

// TemplateClient returns ESClientHandler for template handling.
func (c *ESClientHandler) TemplateClient() template.ClientHandler {
	return template.NewESClientHandler(c.client)
}

// NewESClientHandler returns a new ESClientHandler instance.
func NewESClientHandler(c ESClient) *ESClientHandler {
	return &ESClientHandler{client: c}
}

// ESClient defines the minimal interface required for the index manager to
// prepare an index.
type ESClient interface {
	Request(method, path string, pipeline string, params map[string]string, body interface{}) (int, []byte, error)
	GetVersion() common.Version
}

// FileClientHandler implements the ClientHandler interface for loading to a file.
type FileClientHandler struct {
	client FileClient
}

// FileClient defines the minimal interface required for the ClientHandler to
// prepare a policy and write alias.
type FileClient interface {
	GetVersion() common.Version
	Write(name string, body string) error
}

// NewFileClientHandler returns a new instance of a FileClientHandler
func NewFileClientHandler(c FileClient) *FileClientHandler {
	return &FileClientHandler{client: c}
}

// ILMClient returns FileClientHandler for ILM handling.
func (c *FileClientHandler) ILMClient() ilm.ClientHandler {
	return ilm.NewFileClientHandler(c.client)
}

// TemplateClient returns FileClientHandler for template handling.
func (c *FileClientHandler) TemplateClient() template.ClientHandler {
	return template.NewFileClientHandler(c.client)
}
