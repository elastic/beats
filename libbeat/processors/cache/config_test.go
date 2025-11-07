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

package cache

import (
	"errors"
	"testing"

	"github.com/elastic/go-ucfg/yaml"
)

var validateTests = []struct {
	name string
	cfg  string
	want error
}{
	{
		name: "put_file",
		cfg: `
backend:
  file:
    id: aidmaster
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: nil,
	},
	{
		name: "put_file_with_periodic_write_out",
		cfg: `
backend:
  file:
    id: aidmaster
    write_interval: 15m
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: nil,
	},
	{
		name: "put_memory",
		cfg: `
backend:
  memory:
    id: aidmaster
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: nil,
	},
	{
		name: "get",
		cfg: `
backend:
  file:
    id: aidmaster
get:
  key_field: crowdstrike.aid
  target_field: crowdstrike.metadata
`,
		want: nil,
	},
	{
		name: "delete",
		cfg: `
backend:
  file:
    id: aidmaster
delete:
  key_field: crowdstrike.aid
`,
		want: nil,
	},
	{
		name: "memory_no_id",
		cfg: `
backend:
  memory:
    id: ''
delete:
  key_field: crowdstrike.aid
`,
		want: errors.New("string value is not set accessing 'backend.memory.id'"),
	},
	{
		name: "file_no_id",
		cfg: `
backend:
  file:
    id: ''
delete:
  key_field: crowdstrike.aid
`,
		want: errors.New("string value is not set accessing 'backend.file.id'"),
	},
	{
		name: "no_op",
		cfg: `
backend:
  file:
    id: aidmaster
`,
		want: errors.New("no operation specified for cache processor accessing config"),
	},
	{

		name: "too_many_ops",
		cfg: `
backend:
  file:
    id: aidmaster
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
get:
  key_field: crowdstrike.aid
  target_field: crowdstrike.metadata
`,
		want: errors.New("cannot specify multiple operations together in a cache processor accessing config"),
	},
	{

		name: "no_backend",
		cfg: `
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: errors.New("missing required field accessing 'backend'"),
	},
	{

		name: "incomplete_backend",
		cfg: `
backend:
  file: ~
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: errors.New("must specify one of backend.memory.id or backend.file.id accessing 'backend'"),
	},
	{

		name: "too_many_backends",
		cfg: `
backend:
  file:
    id: aidmaster_f
  memory:
    id: aidmaster_m
put:
  ttl: 168h
  key_field: crowdstrike.aid
  value_field: crowdstrike.metadata
`,
		want: errors.New("must specify only one of backend.memory.id or backend.file.id accessing 'backend'"),
	},
}

func TestValidate(t *testing.T) {
	for _, test := range validateTests {
		t.Run(test.name, func(t *testing.T) {
			got := ucfgRigmarole(test.cfg)
			if !sameError(got, test.want) {
				t.Errorf("unexpected error: got:%v want:%v", got, test.want)
			}
		})
	}
}

func ucfgRigmarole(text string) error {
	c, err := yaml.NewConfig([]byte(text))
	if err != nil {
		return err
	}
	cfg := defaultConfig()
	err = c.Unpack(&cfg)
	if err != nil {
		return err
	}
	return cfg.Validate()
}

func sameError(a, b error) bool {
	switch {
	case a == nil && b == nil:
		return true
	case a == nil, b == nil:
		return false
	default:
		return a.Error() == b.Error()
	}
}
