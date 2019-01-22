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

package readcbor

import (
	"fmt"
	"io"
	"reflect"
	"time"

	gocbor "github.com/ugorji/go/codec"

	"github.com/elastic/beats/libbeat/beat"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cbortransform"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/reader"
)

type CBOR struct {
	reader    io.Reader
	cfg       *Config
	bytesRead int
}

// New creates a new reader that can decode CBOR.
func New(r io.Reader, cfg *Config) *CBOR {
	return &CBOR{reader: r, cfg: cfg}
}

// Next decodes CBOR and returns the filled Line object.
func (r *CBOR) Next() (reader.Message, error) {
	message, err := r.GetMsg()

	message.Content = r.decode(message)
	return message, err
}

func createCBORError(message string) common.MapStr {
	return common.MapStr{"message": message, "type": "cbor"}
}

// decodeCBOR unmarshals the text parameter into a MapStr and
// returns the new text column if one was requested.
func (r *CBOR) decode(msg reader.Message) []byte {
	cborFields := msg.Fields
	if cborFields == nil {
		if !r.cfg.IgnoreDecodingError {
			logp.Err("Error decoding CBOR: empty fields")
		}
		if r.cfg.AddErrorKey {
			msg.AddFields(common.MapStr{"error": createCBORError("Error decoding CBOR: empty fields")})
		}
		return []byte("")
	}

	if len(r.cfg.MessageKey) == 0 {
		return []byte("")
	}

	textValue, ok := cborFields[r.cfg.MessageKey]
	if !ok {
		if r.cfg.AddErrorKey {
			msg.AddFields(common.MapStr{"error": createCBORError(fmt.Sprintf("Key '%s' not found", r.cfg.MessageKey))})
		}
		return []byte("")
	}

	textString, ok := textValue.(string)
	if !ok {
		if r.cfg.AddErrorKey {
			msg.AddFields(common.MapStr{"error": createCBORError(fmt.Sprintf("Value of key '%s' is not a string", r.cfg.MessageKey))})
		}
		return []byte("")
	}
	return []byte(textString)
}

// MergeCBORFields writes the CBOR fields in the event map,
// respecting the KeysUnderRoot and OverwriteKeys configuration options.
// If MessageKey is defined, the Text value from the event always
// takes precedence.
func MergeCBORFields(data common.MapStr, cborFields common.MapStr, text *string, config Config) time.Time {

	// handle the case in which r.cfg.AddErrorKey is set and len(cborFields) == 1
	// and only thing it contains is `error` key due to error in cbor decoding
	// which results in loss of message key in the main beat event
	if len(cborFields) == 1 && cborFields["error"] != nil {
		data["message"] = *text
	}

	if config.KeysUnderRoot {
		// Delete existing cbor key
		delete(data, "cbor")

		var ts time.Time
		if v, ok := data["@timestamp"]; ok {
			switch t := v.(type) {
			case time.Time:
				ts = t
			case common.Time:
				ts = time.Time(ts)
			}
			delete(data, "@timestamp")
		}
		event := &beat.Event{
			Timestamp: ts,
			Fields:    data,
		}
		cbortransform.WriteCBORKeys(event, cborFields, config.OverwriteKeys)

		return event.Timestamp
	}
	return time.Time{}
}

//GetMsg is retrieving the next message in cbor reader
func (r *CBOR) GetMsg() (reader.Message, error) {
	var cborFields common.MapStr
	var ch = &gocbor.CborHandle{}
	ch.MapType = reflect.TypeOf(map[string]interface{}(nil))

	dec := gocbor.NewDecoder(r.reader, ch)
	err := dec.Decode(&cborFields)
	n := dec.NumBytesRead()

	message := reader.Message{
		Ts:    time.Now(),
		Bytes: n,
	}

	message.AddFields(common.MapStr{"cbor": cborFields})
	return message, err
}
