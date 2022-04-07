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

package json

import (
	"bytes"
	stdjson "encoding/json"

	"github.com/elastic/beats/v8/libbeat/beat"
	"github.com/elastic/beats/v8/libbeat/common"
	"github.com/elastic/beats/v8/libbeat/outputs/codec"
	"github.com/elastic/go-structform/gotype"
	"github.com/elastic/go-structform/json"
)

// Encoder for serializing a beat.Event to json.
type Encoder struct {
	buf    bytes.Buffer
	folder *gotype.Iterator

	version string
	config  Config
}

// Config is used to pass encoding parameters to New.
type Config struct {
	Pretty     bool
	EscapeHTML bool
	LocalTime  bool
}

var defaultConfig = Config{
	Pretty:     false,
	EscapeHTML: false,
	LocalTime:  false,
}

func init() {
	codec.RegisterType("json", func(info beat.Info, cfg *common.Config) (codec.Codec, error) {
		config := defaultConfig
		if cfg != nil {
			if err := cfg.Unpack(&config); err != nil {
				return nil, err
			}
		}

		return New(info.Version, config), nil
	})
}

// New creates a new json Encoder.
func New(version string, config Config) *Encoder {
	e := &Encoder{version: version, config: config}
	e.reset()
	return e
}

func (e *Encoder) reset() {
	visitor := json.NewVisitor(&e.buf)
	visitor.SetEscapeHTML(e.config.EscapeHTML)
	visitor.SetIgnoreInvalidFloat(true)

	var err error

	// create new encoder with custom time.Time encoding
	e.folder, err = gotype.NewIterator(visitor,
		gotype.Folders(
			codec.MakeUTCOrLocalTimestampEncoder(e.config.LocalTime),
			codec.MakeBCTimestampEncoder(),
		),
	)
	if err != nil {
		panic(err)
	}
}

// Encode serializes a beat event to JSON. It adds additional metadata in the
// `@metadata` namespace.
func (e *Encoder) Encode(index string, event *beat.Event) ([]byte, error) {
	e.buf.Reset()
	err := e.folder.Fold(makeEvent(index, e.version, event))
	if err != nil {
		e.reset()
		return nil, err
	}

	json := e.buf.Bytes()
	if !e.config.Pretty {
		return json, nil
	}

	var buf bytes.Buffer
	if err = stdjson.Indent(&buf, json, "", "  "); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
