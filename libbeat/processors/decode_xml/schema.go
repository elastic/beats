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

package decode_xml

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/encoding/xml"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/winlogbeat/sys/winevent"
)

const wineventlogSchema = "wineventlog"

type newDecoderFunc func(cfg decodeXMLConfig) decoder
type decoder func(p []byte) (common.MapStr, error)

var (
	registeredDecoders                = map[string]newDecoderFunc{}
	newDefaultDecoder  newDecoderFunc = newSchemaLessDecoder
)

func registerDecoder(schema string, dec newDecoderFunc) error {
	if schema == "" {
		return errors.New("schema can't be empty")
	}

	if dec == nil {
		return errors.New("decoder can't be nil")
	}

	if _, found := registeredDecoders[schema]; found {
		return errors.New("already registered")
	}

	registeredDecoders[schema] = dec

	return nil
}

func newDecoder(cfg decodeXMLConfig) decoder {
	newDec, found := registeredDecoders[cfg.Schema]
	if !found {
		return newDefaultDecoder(cfg)
	}
	return newDec(cfg)
}

func registerDecoders() {
	log := logp.L().Named(logName)
	log.Debug(registerDecoder(wineventlogSchema, newWineventlogDecoder))
}

func newSchemaLessDecoder(cfg decodeXMLConfig) decoder {
	return func(p []byte) (common.MapStr, error) {
		dec := xml.NewDecoder(bytes.NewReader(p))
		if cfg.ToLower {
			dec.LowercaseKeys()
		}

		out, err := dec.Decode()
		if err != nil {
			return nil, fmt.Errorf("error decoding XML field: %w", err)
		}

		return common.MapStr(out), nil
	}
}

func newWineventlogDecoder(decodeXMLConfig) decoder {
	return func(p []byte) (common.MapStr, error) {
		evt, err := winevent.UnmarshalXML(p)
		if err != nil {
			return nil, err
		}
		return evt.Fields(), nil
	}
}
