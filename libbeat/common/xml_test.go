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

package common

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUnmarshalXML(t *testing.T) {
	type xml struct {
		Input  []byte
		Output map[string]interface{}
		Error  error
	}

	tests := []xml{
		{
			Input: []byte(`
			<catalog>
				<book seq="1">
					<author>William H. Gaddis</author>
					<title>The Recognitions</title>
					<review>One of the great seminal American novels of the 20th century.</review>
				</book>
			</catalog>`),
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author": "William H. Gaddis",
						"review": "One of the great seminal American novels of the 20th century.",
						"seq":    "1",
						"title":  "The Recognitions"}}},
			Error: nil,
		},
		{
			Input: []byte(`
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications 
					with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, 
					an evil sorceress, and her own childhood to become queen 
					of the world.</description>
				</book>
			</catalog>`),
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{map[string]interface{}{
						"author":       "Gambardella, Matthew",
						"description":  "An in-depth look at creating applications \n\t\t\t\t\twith XML.",
						"genre":        "Computer",
						"id":           "bk101",
						"price":        "44.95",
						"publish_date": "2000-10-01",
						"title":        "XML Developer's Guide",
					},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, \n\t\t\t\t\tan evil sorceress, and her own childhood to become queen \n\t\t\t\t\tof the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
			Error: nil,
		},
		{
			Input: []byte(`
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications 
					with XML.</description>
				</book>
				<book id="bk102">
					<author>Ralls, Kim</author>
					<title>Midnight Rain</title>
					<genre>Fantasy</genre>
					<price>5.95</price>
					<publish_date>2000-12-16</publish_date>
					<description>A former architect battles corporate zombies, 
					an evil sorceress, and her own childhood to become queen 
					of the world.</description>
				</book>
			</catalog>`),
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": []interface{}{
						map[string]interface{}{
							"author":       "Gambardella, Matthew",
							"description":  "An in-depth look at creating applications \n\t\t\t\t\twith XML.",
							"genre":        "Computer",
							"id":           "bk101",
							"price":        "44.95",
							"publish_date": "2000-10-01",
							"title":        "XML Developer's Guide"},
						map[string]interface{}{
							"author":       "Ralls, Kim",
							"description":  "A former architect battles corporate zombies, \n\t\t\t\t\tan evil sorceress, and her own childhood to become queen \n\t\t\t\t\tof the world.",
							"genre":        "Fantasy",
							"id":           "bk102",
							"price":        "5.95",
							"publish_date": "2000-12-16",
							"title":        "Midnight Rain"}}}},
			Error: nil,
		},
		{
			Input: []byte(`
			<?xml version="1.0"?>
			<catalog>
				<book id="bk101">
					<author>Gambardella, Matthew</author>
					<title>XML Developer's Guide</title>
					<genre>Computer</genre>
					<price>44.95</price>
					<publish_date>2000-10-01</publish_date>
					<description>An in-depth look at creating applications 
					with XML.</description>
				</book>
				<secondcategory>
					<paper id="bk102">
						<test2>Ralls, Kim</test2>
						<description>A former architect battles corporate zombies, 
						an evil sorceress, and her own childhood to become queen 
						of the world.</description>
					</paper>
				</secondcategory>
			</catalog>`),
			Output: map[string]interface{}{
				"catalog": map[string]interface{}{
					"book": map[string]interface{}{
						"author":       "Gambardella, Matthew",
						"description":  "An in-depth look at creating applications \n\t\t\t\t\twith XML.",
						"genre":        "Computer",
						"id":           "bk101",
						"price":        "44.95",
						"publish_date": "2000-10-01",
						"title":        "XML Developer's Guide"},
					"secondcategory": map[string]interface{}{
						"paper": map[string]interface{}{
							"description": "A former architect battles corporate zombies, \n\t\t\t\t\t\tan evil sorceress, and her own childhood to become queen \n\t\t\t\t\t\tof the world.",
							"id":          "bk102",
							"test2":       "Ralls, Kim"}}}},
			Error: nil,
		},
	}

	for _, test := range tests {
		out, err := UnmarshalXML(test.Input, false, true)
		assert.Equal(t, test.Output, out)
		assert.Equal(t, test.Error, err)
	}
}
