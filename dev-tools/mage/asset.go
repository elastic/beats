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

package mage

import (
	"io/ioutil"

	"github.com/elastic/beats/libbeat/asset"
	"github.com/elastic/beats/licenses"
)

func Asset(license, name, input, output string) error {
	licenseHeader, err := licenses.Find(license)
	if err != nil {
		return err
	}

	data, err := ioutil.ReadFile(input)
	if err != nil {
		return err
	}

	bs, err := asset.CreateAsset(licenseHeader, name, "fields.yml", "include", data, "asset.BeatFieldsPri", input)
	if err != nil {
		panic(err)
	}

	return ioutil.WriteFile(output, bs, 0640)
}
