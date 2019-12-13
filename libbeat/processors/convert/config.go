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

package convert

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
)

func defaultConfig() config {
	return config{
		IgnoreMissing: false,
		FailOnError:   true,
		Mode:          copyMode,
	}
}

type config struct {
	Fields        []field `config:"fields" validate:"required"` // List of fields to convert.
	Tag           string  `config:"tag"`                        // Processor ID for debug and metrics.
	IgnoreMissing bool    `config:"ignore_missing"`             // Skip field when From field is missing.
	FailOnError   bool    `config:"fail_on_error"`              // Ignore errors (missing fields / conversion failures).
	Mode          mode    `config:"mode"`                       // Mode (copy vs rename).
}

type field struct {
	From string   `config:"from" validate:"required"`
	To   string   `config:"to"`
	Type dataType `config:"type"`
}

func (f field) Validate() error {
	if f.To == "" && f.Type == unset {
		return errors.New("each field must have a 'to' or a 'type'")
	}
	return nil
}

func (f field) String() string {
	return fmt.Sprintf("{from=%v, to=%v, type=%v}", f.From, f.To, f.Type)
}

type dataType uint8

// List of dataTypes.
const (
	unset dataType = iota
	Integer
	Long
	Float
	Double
	String
	Boolean
	IP
)

var dataTypeNames = map[dataType]string{
	unset:   "[unset]",
	Integer: "integer",
	Long:    "long",
	Float:   "float",
	Double:  "double",
	String:  "string",
	Boolean: "boolean",
	IP:      "ip",
}

func (dt dataType) String() string {
	return dataTypeNames[dt]
}

func (dt dataType) MarshalText() ([]byte, error) {
	return []byte(dt.String()), nil
}

func (dt *dataType) Unpack(s string) error {
	s = strings.ToLower(s)
	for typ, name := range dataTypeNames {
		if s == name {
			*dt = typ
			return nil
		}
	}
	return errors.Errorf("invalid data type: %v", s)
}

type mode uint8

// List of modes.
const (
	copyMode mode = iota
	renameMode
)

var modeNames = map[mode]string{
	copyMode:   "copy",
	renameMode: "rename",
}

func (m mode) String() string {
	return modeNames[m]
}

func (m mode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

func (m *mode) Unpack(s string) error {
	s = strings.ToLower(s)
	for md, name := range modeNames {
		if s == name {
			*m = md
			return nil
		}
	}
	return errors.Errorf("invalid mode: %v", s)
}
