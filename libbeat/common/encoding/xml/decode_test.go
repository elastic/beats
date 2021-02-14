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

// +build !integration

package xml

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncompleteXML(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
`

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	assert.Nil(t, out)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected EOF")
}

func TestLowercaseKeys(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
</person>
`

	expected := map[string]interface{}{
		"person": map[string]interface{}{
			"name": map[string]interface{}{
				"#text": "John",
				"id":    "123",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	d.LowercaseKeys()
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestPrependHyphenToAttr(t *testing.T) {
	const xml = `
<person>
  <Name ID="123">John</Name>
</person>
`

	expected := map[string]interface{}{
		"person": map[string]interface{}{
			"Name": map[string]interface{}{
				"#text": "John",
				"-ID":   "123",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	d.PrependHyphenToAttr()
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestDecodeList(t *testing.T) {
	const xml = `
<people>
	<person>
	  <Name ID="123">John</Name>
	</person>
	<person>
	  <Name ID="456">Jane</Name>
	</person>
    <person>Foo</person>
</people>
`

	expected := map[string]interface{}{
		"people": map[string]interface{}{
			"person": []interface{}{
				map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "John",
						"ID":    "123",
					},
				},
				map[string]interface{}{
					"Name": map[string]interface{}{
						"#text": "Jane",
						"ID":    "456",
					},
				},
				"Foo",
			},
		},
	}

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestEmptyElement(t *testing.T) {
	const xml = `
<people>
</people>
`

	expected := map[string]interface{}{
		"people": "",
	}

	d := NewDecoder(strings.NewReader(xml))
	out, err := d.Decode()
	require.NoError(t, err)
	assert.Equal(t, expected, out)
}

func TestDecode(t *testing.T) {
	type testCase struct {
		XML    string
		Output map[string]interface{}
	}

	tests := []testCase{
		{
			XML: `
			<catalog>
				<book seq="1">
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author": "William H. Gaddis",
						"review": "One of the great seminal American novels of the 20th century.",
						"seq":    "1",
						"title":  "The Recognitions"}}},
		},
		{
			XML: `
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{
							"author":       "Gambardella, Matthew",
							"description":  "An in-depth look at creating applications with XML.",
							"genre":        "Computer",
							"id":           "bk101",
							"price":        "44.95",
							"publish_date": "2000-10-01",
							"title":        "XML Developer's Guide",
						},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
		},
		{
			XML: `
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
				</book>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{
							"author":       "Gambardella, Matthew",
							"description":  "An in-depth look at creating applications with XML.",
							"genre":        "Computer",
							"id":           "bk101",
							"price":        "44.95",
							"publish_date": "2000-10-01",
							"title":        "XML Developer's Guide"},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
		},
		{
			XML: `
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications with XML.</description>
				</book>
				<secondcategory>
					<paper id="bk102">
						<test2>Ralls, Kim</test2>
						<description>A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.</description>
					</paper>
				</secondcategory>
			</catalog>`,
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author":       "Gambardella, Matthew",
						"description":  "An in-depth look at creating applications with XML.",
						"genre":        "Computer",
						"id":           "bk101",
						"price":        "44.95",
						"publish_date": "2000-10-01",
						"title":        "XML Developer's Guide"},
					"secondcategory": map[string]interface{}{
						"paper": map[string]interface{}{
							"description": "A former architect battles corporate zombies, an evil sorceress, and her own childhood to become queen of the world.",
							"id":          "bk102",
							"test2":       "Ralls, Kim"}}}},
		},
	}

	for i, test := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			d := NewDecoder(strings.NewReader(test.XML))
			d.LowercaseKeys()

			out, err := d.Decode()
			require.NoError(t, err)
			assert.EqualValues(t, test.Output, out)
		})
	}
}
