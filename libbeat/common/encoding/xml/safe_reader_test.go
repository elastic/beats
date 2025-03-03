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

//go:build !integration

package xml

import (
	"encoding/xml"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInvalidXMLIsSanitized(t *testing.T) {
	tests := []struct {
		name  string
		input []byte
		want  map[string]interface{}
		err   error
	}{
		{
			name: "control space",
			input: []byte(`<person><Name ID="123">John` + "\t" + `
</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John",
						"ID":    "123",
					},
				},
			},
		},
		{
			name:  "crlf",
			input: []byte(`<person><Name ID="123">John` + "\r\n" + `</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John",
						"ID":    "123",
					},
				},
			},
		},
		{
			name:  "single invalid",
			input: []byte(`<person><Name ID="123">John` + "\x80" + `</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John\\ufffd",
						"ID":    "123",
					},
				},
			},
		},
		{
			name:  "double invalid",
			input: []byte(`<person><Name ID="123">` + "\x80" + `John` + "\x80" + `</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "\\ufffdJohn\\ufffd",
						"ID":    "123",
					},
				},
			},
		},
		{
			name:  "happy single invalid",
			input: []byte(`<person><Name ID="123">😊John` + "\x80" + `</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "😊John\\ufffd",
						"ID":    "123",
					},
				},
			},
		},
		{
			name:  "invalid tag",
			input: []byte(`<person><Name ID` + "\x80" + `="123">John</Name></person>`),
			want:  nil,
			err:   &xml.SyntaxError{Msg: "attribute name without = in element", Line: 1},
		},
		{
			name:  "invalid tag value",
			input: []byte(`<person><Name ID="` + "\x80" + `123">John</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John",
						"ID":    "\\ufffd123",
					},
				},
			},
		},
		{
			name:  "unhappy",
			input: []byte(`<person><Name ID="123">John is` + strings.Repeat(" ", 223) + ` 😞</Name></person>`),
			want: map[string]interface{}{
				"person": map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John is" + strings.Repeat(" ", 223) + " 😞",
						"ID":    "123",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			d := NewDecoder(NewSafeReader(test.input))
			out, err := d.Decode()
			assert.Equal(t, test.err, err)
			assert.Equal(t, test.want, out)
		})
	}
}
