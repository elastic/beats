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

package gzip

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/processors"
	"github.com/elastic/beats/libbeat/processors/checks"

	"bytes"
	"compress/gzip"
	"io"
)

type decompressGzipFields struct {
	decompressGzipFieldsConfig
	fields map[string]string
}

type decompressGzipFieldsConfig struct {
	Fields        common.MapStr `config:"fields"`
	IgnoreMissing bool          `config:"ignore_missing"`
	OverwriteKeys bool          `config:"overwrite_keys"`
	FailOnError   bool          `config:"fail_on_error"`
}

var (
	defaultDecompressGzipFieldsConfig = decompressGzipFieldsConfig{
		FailOnError: true,
	}
)

func init() {
	processors.RegisterPlugin("decompress_gzip_fields",
		checks.ConfigChecked(NewDecompressGzipFields,
			checks.RequireFields("fields"),
			checks.AllowedFields("fields", "ignore_missing", "overwrite_keys", "overwrite_keys", "fail_on_error", "when")))
}

// NewDecompressGzipFields construct a new decompress_gzip_fields processor.
func NewDecompressGzipFields(c *common.Config) (processors.Processor, error) {
	config := defaultDecompressGzipFieldsConfig

	err := c.Unpack(&config)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack the decompress_gzip_fields configuration: %s", err)
	}
	if len(config.Fields) == 0 {
		return nil, errors.New("no fields to decompress configured")
	}
	f := &decompressGzipFields{decompressGzipFieldsConfig: config}

	// Set fields as string -> string
	f.fields = make(map[string]string, len(config.Fields))
	for src, dstIf := range config.Fields.Flatten() {
		dst, ok := dstIf.(string)
		if !ok {
			return nil, errors.Errorf("bad destination mapping for %s: destination field must be string, not %T (got %v)", src, dstIf, dstIf)
		}
		f.fields[src] = dst
	}
	return f, nil
}

// Run applies the decompress_gzip_fields processor to an event.
func (f *decompressGzipFields) Run(event *beat.Event) (*beat.Event, error) {
	saved := *event
	if f.FailOnError {
		saved.Fields = event.Fields.Clone()
		saved.Meta = event.Meta.Clone()
	}
	for src, dest := range f.fields {
		if err := f.decompressGzipField(src, dest, event); err != nil && f.FailOnError {
			return &saved, err
		}
	}
	return event, nil
}

func (f *decompressGzipFields) decompressGzipField(src, dest string, event *beat.Event) error {
	// Check source value
	data, err := event.GetValue(src)
	if err != nil {
		if f.IgnoreMissing && errors.Cause(err) == common.ErrKeyNotFound {
			return nil
		}
		return errors.Wrapf(err, "could not fetch value for field %s", src)
	}

	text, ok := data.(string)
	if !ok {
		return errors.Errorf("field %s is not of string type", src)
	}

	//Uncompress gzip message
	b := bytes.NewBuffer([]byte(text))

	var r io.Reader
	r, err = gzip.NewReader(b)

	if err != nil {
		return errors.Wrapf(err, "error decompressing field %s", src)
	}

	var resB bytes.Buffer
	resB.ReadFrom(r)

	decompressed := string(resB.Bytes())

	// Write message out
	if src != dest && !f.OverwriteKeys {
		if _, err = event.GetValue(dest); err == nil {
			return errors.Errorf("target field %s already has a value. Set the overwrite_keys flag or drop/rename the field first", dest)
		}
	}
	if _, err = event.PutValue(dest, decompressed); err != nil {
		return errors.Wrapf(err, "failed setting field %s", dest)
	}
	return nil
}

// String returns a string representation of this processor.
func (f decompressGzipFields) String() string {
	json, _ := json.Marshal(f.decompressGzipFieldsConfig)
	return "decompress_gzip_fields=" + string(json)
}
