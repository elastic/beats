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

package http

import (
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
)

type contentEncoder interface {
	AddHeaders(*http.Header)
	Encode(to io.Writer, from io.Reader) error
}

type nilEncoder struct{}

type gzipEncoder struct {
	gz *gzip.Writer
}

var plainEncoder = nilEncoder{}

func getContentEncoder(name string, level int) (contentEncoder, error) {
	name = strings.ToLower(name)
	if name == "gzip" {
		return newGZIPEncoder(level)
	}
	if name != "" {
		return nil, errors.New("invalid content encoder")
	}

	return plainEncoder, nil
}

func (nilEncoder) AddHeaders(_ *http.Header) {}

func (nilEncoder) Encode(to io.Writer, from io.Reader) error {
	_, err := io.Copy(to, from)
	return err
}

func newGZIPEncoder(level int) (*gzipEncoder, error) {
	w, err := gzip.NewWriterLevel(nil, level)
	if err != nil {
		return nil, err
	}

	return &gzipEncoder{w}, nil
}

func (e *gzipEncoder) AddHeaders(h *http.Header) {
	h.Add("Content-Type", "application/json; charset=UTF-8")
	h.Add("Content-Encoding", "gzip")
}

func (e *gzipEncoder) Encode(to io.Writer, from io.Reader) error {
	e.gz.Reset(to)
	if _, err := io.Copy(e.gz, from); err != nil {
		return err
	}
	if err := e.gz.Flush(); err != nil {
		return err
	}
	return nil
}
