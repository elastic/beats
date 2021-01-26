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

package xmldecode

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
)

var (
	targetField = "xml"
)

func TestXMLDecode(t *testing.T) {
	var testCases = []struct {
		description  string
		config       xmlDecodeConfig
		Input        common.MapStr
		Output       common.MapStr
		error        bool
		errorMessage string
	}{
		{
			description: "Simple xml decode",
			config: xmlDecodeConfig{
				Fields: []string{"message"},
				Target: &targetField,
			},
			Input: common.MapStr{
				"message": `<catalog>
					<book seq="1">
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
					</book>
				</catalog>`,
			},
			Output: common.MapStr{
				"xml": map[string]interface{}{
					"catalog": map[string]interface{}{
						"book": map[string]interface{}{
							"author": "William H. Gaddis",
							"review": "One of the great seminal American novels of the 20th century.",
							"seq":    "1",
							"title":  "The Recognitions",
						},
					},
				},
				"message": `<catalog>
					<book seq="1">
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
					</book>
				</catalog>`,
			},
			error:        false,
			errorMessage: "",
		},
		{
			description: "Simple xml decode with xml string to same field name when Target is null",
			config: xmlDecodeConfig{
				Fields: []string{"message"},
			},
			Input: common.MapStr{
				"message": `<?xml version="1.0"?>
				<catalog>
					<book seq="1">
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
					</book>
				</catalog>`,
			},
			Output: common.MapStr{
				"message": map[string]interface{}{
					"catalog": map[string]interface{}{
						"book": map[string]interface{}{
							"author": "William H. Gaddis",
							"review": "One of the great seminal American novels of the 20th century.",
							"seq":    "1",
							"title":  "The Recognitions",
						},
					},
				},
			},
			error:        false,
			errorMessage: "",
		},
		{
			description: "Decoding with array input",
			config: xmlDecodeConfig{
				Fields: []string{"message"},
			},
			Input: common.MapStr{
				"message": `<?xml version="1.0"?>
				<catalog>
					<book>
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
					</book>
					<book>
						<author>Ralls, Kim</author>
						<title>Midnight Rain</title>
						<review>Some review.</review>
					</book>
				</catalog>`,
			},
			Output: common.MapStr{
				"message": map[string]interface{}{
					"catalog": map[string]interface{}{
						"book": []interface{}{
							map[string]interface{}{
								"author": "William H. Gaddis",
								"review": "One of the great seminal American novels of the 20th century.",
								"title":  "The Recognitions",
							},
							map[string]interface{}{
								"author": "Ralls, Kim",
								"review": "Some review.",
								"title":  "Midnight Rain",
							},
						},
					},
				},
			},
			error:        false,
			errorMessage: "",
		},
		{
			description: "Decoding with multiple xml objects",
			config: xmlDecodeConfig{
				Fields: []string{"message"},
			},
			Input: common.MapStr{
				"message": `<?xml version="1.0"?>
				<catalog>
					<book>
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</book>
				<book>
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<review>Some review.</review>
				</book>
				<secondcategory>
					<paper id="bk102">
						<test2>Ralls, Kim</test2>
						<description>A former architect battles corporate zombies, 
						an evil sorceress, and her own childhood to become queen 
						of the world.</description>
					</paper>
				</secondcategory>
				</catalog>`,
			},
			Output: common.MapStr{
				"message": map[string]interface{}{
					"catalog": map[string]interface{}{
						"book": []interface{}{
							map[string]interface{}{
								"author": "William H. Gaddis",
								"review": "One of the great seminal American novels of the 20th century.",
								"title":  "The Recognitions",
							},
							map[string]interface{}{
								"author": "Ralls, Kim",
								"review": "Some review.",
								"title":  "Midnight Rain",
							},
						},
						"secondcategory": map[string]interface{}{
							"paper": map[string]interface{}{
								"description": "A former architect battles corporate zombies, \n\t\t\t\t\t\tan evil sorceress, and her own childhood to become queen \n\t\t\t\t\t\tof the world.",
								"id":          "bk102",
								"test2":       "Ralls, Kim",
							},
						},
					},
				},
			},
			error:        false,
			errorMessage: "",
		},
		{
			description: "Decoding with broken XML format",
			config: xmlDecodeConfig{
				Fields:      []string{"message"},
				AddErrorKey: true,
			},
			Input: common.MapStr{
				"message": `<?xml version="1.0"?>
				<catalog>
					<book>
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</ook>
				catalog>`,
			},
			Output: common.MapStr{
				"message": map[string]interface{}{},
				"error":   []string{"error trying to decode XML field xml.Decoder.Token() - XML syntax error on line 7: element <book> closed by </ook>"},
			},
			error:        true,
			errorMessage: "error trying to decode XML field xml.Decoder.Token() - XML syntax error on line 7: element <book> closed by </ook>",
		},
	}

	for _, test := range testCases {
		test := test
		t.Log("testing")
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()
			f := &xmlDecode{
				logger: logp.NewLogger("xmldecode"),
				config: test.config,
			}

			event := &beat.Event{
				Fields: test.Input,
			}
			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.EqualError(t, err, test.errorMessage)
			}
			assert.Equal(t, test.Output, newEvent.Fields)
		})
	}

}
