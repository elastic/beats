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

package mime

import (
	"encoding/json"
	"encoding/xml"
	"net/http"
	"strings"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/processors"
	"github.com/h2non/filetype"
	"github.com/pkg/errors"
)

const (
	processorName = "mime"
	// size for mime detection, office file
	// detection requires ~8kb to detect properly
	headerSize = 8192
)

func init() {
	processors.RegisterPlugin(processorName, New)
}

type mimeType struct {
	from string
	to   string
	log  *logp.Logger
}

// New constructs a new mime processor.
func New(cfg *common.Config) (processors.Processor, error) {
	var config config
	if err := cfg.Unpack(&config); err != nil {
		return nil, errors.Wrapf(err, "fail to unpack the %v configuration", processorName)
	}

	log := logp.NewLogger(processorName)

	return &mimeType{
		from: config.FromOrDefault(),
		to:   config.ToOrDefault(),
		log:  log,
	}, nil
}

func (p *mimeType) Run(event *beat.Event) (*beat.Event, error) {
	valI, err := event.GetValue(p.from)
	if err != nil {
		// doesn't have the required from value to analyze
		return event, nil
	}
	val, _ := valI.(string)
	if val == "" {
		// wrong type or not set
		return event, nil
	}
	data := []byte(val)
	mimeType := p.analyze(data)
	if mimeType != "" {
		event.Fields.DeepUpdate(common.MapStr{
			p.to: mimeType,
		})
	}
	return event, nil
}

func (p *mimeType) analyze(data []byte) string {
	header := data
	if len(data) > headerSize {
		header = data[:headerSize]
	}
	kind, err := filetype.Match(header)
	if err == nil && kind != filetype.Unknown {
		// we have a known filetype, return
		return kind.MIME.Value
	}
	// if the above fails, try and sniff with http sniffing
	netType := http.DetectContentType(header)
	if netType == "application/octet-stream" {
		return ""
	}
	// try and parse any sort of text as json or xml
	if strings.HasPrefix(netType, "text/plain") {
		if detected := p.detectEncodedText(data); detected != "" {
			return detected
		}
	}
	return netType
}

func (p *mimeType) detectEncodedText(data []byte) string {
	// figure out how to optimize this so we don't have to try and parse the whole payload
	// every time
	if json.Valid(data) {
		return "application/json"
	}
	if xml.Unmarshal(data, new(interface{})) == nil {
		return "text/xml"
	}
	return ""
}

func (p *mimeType) String() string {
	return processorName
}
