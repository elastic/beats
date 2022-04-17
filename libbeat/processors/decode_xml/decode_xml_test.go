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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/menderesk/beats/v7/libbeat/beat"
	"github.com/menderesk/beats/v7/libbeat/common"
)

var (
	testXMLTargetField  = "xml"
	testRootTargetField = ""
)

func TestDecodeXML(t *testing.T) {
	var testCases = []struct {
		description  string
		config       decodeXMLConfig
		Input        common.MapStr
		Output       common.MapStr
		error        bool
		errorMessage string
	}{
		{
			description: "Simple xml decode with target field set",
			config: decodeXMLConfig{
				Field:  "message",
				Target: &testXMLTargetField,
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
				"xml": common.MapStr{
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
		},
		{
			description: "Test with target set to root",
			config: decodeXMLConfig{
				Field:  "message",
				Target: &testRootTargetField,
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
				"catalog": common.MapStr{
					"book": map[string]interface{}{
						"author": "William H. Gaddis",
						"review": "One of the great seminal American novels of the 20th century.",
						"seq":    "1",
						"title":  "The Recognitions",
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
		},
		{
			description: "Simple xml decode with xml string to same field name when Target is null",
			config: decodeXMLConfig{
				Field: "message",
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
				"message": common.MapStr{
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
		},
		{
			description: "Decoding with array input",
			config: decodeXMLConfig{
				Field: "message",
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
				"message": common.MapStr{
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
		},
		{
			description: "Decoding with an array and mixed-case keys",
			config: decodeXMLConfig{
				Field:   "message",
				ToLower: true,
			},
			Input: common.MapStr{
				"message": `<AuditBase>
				  <ContextComponents>
					<Component>
					  <RelyingParty>N/A</RelyingParty>
					</Component>
					<Component>
					  <PrimaryAuth>N/A</PrimaryAuth>
					</Component>
				  </ContextComponents>
				</AuditBase>`,
			},
			Output: common.MapStr{
				"message": common.MapStr{
					"auditbase": map[string]interface{}{
						"contextcomponents": map[string]interface{}{
							"component": []interface{}{
								map[string]interface{}{
									"relyingparty": "N/A",
								},
								map[string]interface{}{
									"primaryauth": "N/A",
								},
							},
						},
					},
				},
			},
		},
		{
			description: "Decoding with multiple xml objects",
			config: decodeXMLConfig{
				Field: "message",
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
						<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
					</paper>
				</secondcategory>
				</catalog>`,
			},
			Output: common.MapStr{
				"message": common.MapStr{
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
								"description": "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
								"id":          "bk102",
								"test2":       "Ralls, Kim",
							},
						},
					},
				},
			},
		},
		{
			description: "Decoding with broken XML format, with IgnoreFailure false",
			config: decodeXMLConfig{
				Field:         "message",
				IgnoreFailure: false,
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
				"message": `<?xml version="1.0"?>
				<catalog>
					<book>
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
				</ook>
				catalog>`,
				"error": common.MapStr{"message": "failed in decode_xml on the \"message\" field: error decoding XML field: XML syntax error on line 7: element <book> closed by </ook>"},
			},
			error:        true,
			errorMessage: "error decoding XML field:",
		},
		{
			description: "Decoding with broken XML format, with IgnoreFailure true",
			config: decodeXMLConfig{
				Field:         "message",
				IgnoreFailure: true,
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
				"message": `<?xml version="1.0"?>
				<catalog>
					<book>
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
				</ook>
				catalog>`,
			},
		},
		{
			description: "Test when the XML field is empty, IgnoreMissing false",
			config: decodeXMLConfig{
				Field:         "message2",
				IgnoreMissing: false,
			},
			Input: common.MapStr{
				"message": "testing message",
			},
			Output: common.MapStr{
				"message": "testing message",
				"error":   common.MapStr{"message": "failed in decode_xml on the \"message2\" field: key not found"},
			},
			error:        true,
			errorMessage: "key not found",
		},
		{
			description: "Test when the XML field is empty IgnoreMissing true",
			config: decodeXMLConfig{
				Field:         "message2",
				IgnoreMissing: true,
			},
			Input: common.MapStr{
				"message": "testing message",
			},
			Output: common.MapStr{
				"message": "testing message",
			},
		},
		{
			description: "Test when the XML field not a string, IgnoreFailure false",
			config: decodeXMLConfig{
				Field:         "message",
				IgnoreFailure: false,
			},
			Input: common.MapStr{
				"message": 1,
			},
			Output: common.MapStr{
				"message": 1,
				"error":   common.MapStr{"message": "failed in decode_xml on the \"message\" field: field value is not a string"},
			},
			error:        true,
			errorMessage: "field value is not a string",
		},
		{
			description: "Test when the XML field not a string, IgnoreFailure true",
			config: decodeXMLConfig{
				Field:         "message",
				IgnoreFailure: true,
			},
			Input: common.MapStr{
				"message": 1,
			},
			Output: common.MapStr{
				"message": 1,
			},
		},
	}

	for _, test := range testCases {
		test := test
		t.Run(test.description, func(t *testing.T) {
			t.Parallel()

			f, err := newDecodeXML(test.config)
			require.NoError(t, err)

			event := &beat.Event{
				Fields: test.Input,
			}
			newEvent, err := f.Run(event)
			if !test.error {
				assert.NoError(t, err)
			} else {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), test.errorMessage)
				}
			}
			assert.Equal(t, test.Output, newEvent.Fields)
		})
	}

	t.Run("supports metadata as a target", func(t *testing.T) {
		t.Parallel()
		target := "@metadata.xml"
		config := decodeXMLConfig{
			Field:  "@metadata.message",
			Target: &target,
		}

		f, err := newDecodeXML(config)
		require.NoError(t, err)

		event := &beat.Event{
			Meta: common.MapStr{
				"message": `<catalog>
					<book seq="1">
						<author>William H. Gaddis</author>
						<title>The Recognitions</title>
						<review>One of the great seminal American novels of the 20th century.</review>
					</book>
				</catalog>`,
			},
		}
		expMeta := common.MapStr{
			"xml": common.MapStr{
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
		}

		newEvent, err := f.Run(event)
		assert.NoError(t, err)
		assert.Equal(t, expMeta, newEvent.Meta)
		assert.Equal(t, event.Fields, newEvent.Fields)
	})
}

func BenchmarkProcessor_Run(b *testing.B) {
	c := defaultConfig()
	target := "xml"
	c.Target = &target
	p, err := newDecodeXML(c)
	require.NoError(b, err)

	b.Run("single_object", func(b *testing.B) {
		evt := &beat.Event{Fields: map[string]interface{}{
			"message": `<?xml version="1.0"?>
				<catalog>
					<book>
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</book>
				</catalog>`,
		}}

		for i := 0; i < b.N; i++ {
			_, err = p.Run(evt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("nested_and_array_object", func(b *testing.B) {
		evt := &beat.Event{Fields: map[string]interface{}{
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
						<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
					</paper>
				</secondcategory>
				</catalog>`,
		}}

		for i := 0; i < b.N; i++ {
			_, err = p.Run(evt)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestXMLToDocumentID(t *testing.T) {
	p, err := newDecodeXML(decodeXMLConfig{
		Field:      "message",
		DocumentID: "catalog.book.seq",
	})
	require.NoError(t, err)

	input := common.MapStr{
		"message": `<catalog>
						<book seq="10">
							<author>William H. Gaddis</author>
							<title>The Recognitions</title>
							<review>One of the great seminal American novels of the 20th century.</review>
						</book>
					</catalog>`,
	}
	actual, err := p.Run(&beat.Event{Fields: input})
	require.NoError(t, err)

	wantFields := common.MapStr{
		"message": common.MapStr{
			"catalog": map[string]interface{}{
				"book": map[string]interface{}{
					"author": "William H. Gaddis",
					"review": "One of the great seminal American novels of the 20th century.",
					"title":  "The Recognitions",
				},
			},
		},
	}
	wantMeta := common.MapStr{
		"_id": "10",
	}

	assert.Equal(t, wantFields, actual.Fields)
	assert.Equal(t, wantMeta, actual.Meta)
}
