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

package client

import (
	"encoding/json"
)

// License
type License struct {
	Status string
	Type   string
}

func (c *Client) GetLicense() (*License, error) {
	var rs struct {
		License License `json:"license"`
	}

	switch {
	case c.e6 != nil:
		r, err := c.e6.XPack.LicenseGet()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	case c.e7 != nil:
		r, err := c.e7.License.Get()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	case c.e8 != nil:
		r, err := c.e8.License.Get()
		if err != nil {
			return nil, err
		}
		defer r.Body.Close()

		if r.IsError() {
			return nil, errorFromBody(r.Body)
		}

		d := json.NewDecoder(r.Body)
		if err := d.Decode(&rs); err != nil {
			return nil, err
		}

	default:
		return nil, ErrUnsupportedVersion
	}

	return &rs.License, nil
}
