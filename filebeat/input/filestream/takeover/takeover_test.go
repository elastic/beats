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

package takeover

import (
	"testing"

	"github.com/elastic/beats/v7/filebeat/backup"
	cfg "github.com/elastic/beats/v7/filebeat/config"
	"github.com/elastic/beats/v7/libbeat/statestore/backend"

	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"

	"github.com/stretchr/testify/require"
)

func TestTakeOverLogInputStates(t *testing.T) {
	empty, err := conf.NewConfigFrom(``)
	require.NoError(t, err)

	noTakeOver, err := conf.NewConfigFrom(`
inputs:
  - type: log
    paths:
      - "/path/log*.log"
  - type: filestream
    id: filestream-id-1
    enabled: true
    paths:
      - "/path/filestream1-*.log"
`)
	require.NoError(t, err)

	takeOver, err := conf.NewConfigFrom(`
inputs:
  - type: filestream
    id: filestream-id-1
    enabled: true
    paths:
      - "/path/filestream1-*.log"
  - type: filestream
    id: filestream-id-2
    take_over: true
    enabled: true
    paths:
      - "/path/filestream2-*.log"
      - "/path/log*.log" # taking over from the log input

`)
	require.NoError(t, err)

	noUniqueID, err := conf.NewConfigFrom(`
inputs:
  - type: filestream
    id: filestream-id-2
    take_over: true
    enabled: true
    paths:
      - "/path/filestream2-*.log"
  - type: filestream
    id: filestream-id-2 # not unique
    take_over: true
    enabled: true
    paths:
      - "/path/filestream3-*.log"
  - type: filestream
    take_over: true # no ID
    enabled: true
    paths:
      - "/path/filestream-*.log"
`)
	require.NoError(t, err)

	states := []state{
		// this state is to make sure the filestreams without `take_over` remain untouched
		{
			key: "filestream::filestream-id-1::native::11111111-22222222",
			value: mapstr.M{
				"meta": mapstr.M{
					"source":          "/path/filestream1-1.log",
					"identifier_name": "native",
				},
				"ttl":     1800000000000,
				"updated": []int{257795329760, 1671033739},
				"cursor": mapstr.M{
					"offset": 42,
				},
			},
		},
		{
			key: "filebeat::logs::native::92938222-16777232",
			value: mapstr.M{
				"source":    "/path/log1.log",
				"timestamp": []int{258139663760, 1671033742},
				"ttl":       -1,
				"id":        "native::92938222-16777232",
				"offset":    392012100,
				"type":      "log",
				"FileStateOS": mapstr.M{
					"inode":  92938222,
					"device": 16777232,
				},
				"identifier_name": "native",
				"prev_id":         "",
			},
		},
		// second file to test it works on multiple log input states
		{
			key: "filebeat::logs::native::92938223-16777233",
			value: mapstr.M{
				"source":    "/path/log2.log",
				"timestamp": []int{258139663761, 1671033743},
				"ttl":       -1,
				"id":        "native::92938223-16777233",
				"offset":    64625356,
				"type":      "log",
				"FileStateOS": mapstr.M{
					"inode":  92938223,
					"device": 16777233,
				},
				"identifier_name": "native",
				"prev_id":         "",
			},
		},
		{
			// this is to make sure that a state that does not match a filestream path
			// remains untouched
			key: "filebeat::logs::native::33333333-44444444",
			value: mapstr.M{
				"source":    "/path/NOMATCH.log",
				"timestamp": []int{258139663760, 1671033742},
				"ttl":       -1,
				"id":        "native::33333333-44444444",
				"offset":    1,
				"type":      "log",
				"FileStateOS": mapstr.M{
					"inode":  33333333,
					"device": 44444444,
				},
				"identifier_name": "native",
				"prev_id":         "",
			},
		},
	}

	cases := []struct {
		name       string
		cfg        *conf.C
		states     []state
		mustBackup bool
		mustRemove []string
		mustSet    []setOp
		expErr     string
	}{
		{
			name:   "does nothing when default config",
			cfg:    empty,
			states: states,
		},
		{
			name:   "does nothing when there is no filestream with `take_over`",
			cfg:    noTakeOver,
			states: states,
		},
		{
			name:   "returns error if filestreams don't have unique IDs",
			cfg:    noUniqueID,
			states: states,
			expErr: "failed to read input configuration: filestream with ID `filestream-id-2` in `take over` mode requires a unique ID. Add the `id:` key with a unique value",
		},
		{
			name:       "filestream takes over when there is `take_over: true`",
			cfg:        takeOver,
			states:     states,
			mustBackup: true,
			mustRemove: []string{
				states[1].key,
				states[2].key,
			},
			mustSet: []setOp{
				{
					key: "filestream::filestream-id-2::native::92938222-16777232",
					value: mapstr.M{
						"meta": mapstr.M{
							"source":          "/path/log1.log",
							"identifier_name": "native",
						},
						"ttl":     -1,
						"updated": []int{258139663760, 1671033742},
						"cursor": mapstr.M{
							"offset": 392012100,
						},
					},
				},
				{
					key: "filestream::filestream-id-2::native::92938223-16777233",
					value: mapstr.M{
						"meta": mapstr.M{
							"source":          "/path/log2.log",
							"identifier_name": "native",
						},
						"ttl":     -1,
						"updated": []int{258139663761, 1671033743},
						"cursor": mapstr.M{
							"offset": 64625356,
						},
					},
				},
			},
		},
		{
			name:   "does nothing when there is no matching loginput entry",
			cfg:    takeOver,
			states: states[3:],
		},
	}

	log := logp.NewLogger("takeover-test")

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			config := cfg.DefaultConfig
			err := tc.cfg.Unpack(&config)
			require.NoError(t, err)

			store := storeMock{
				states: tc.states,
			}
			backuper := backuperMock{}
			err = TakeOverLogInputStates(log, &store, &backuper, &config)
			if tc.expErr != "" {
				require.Error(t, err)
				require.Contains(t, err.Error(), tc.expErr)
				return
			}
			require.NoError(t, err)

			if tc.mustBackup {
				require.Equal(t, 1, backuper.called, "backup must be called exactly once")
			} else {
				require.Equal(t, 0, backuper.called, "backup must not be called")
			}

			require.ElementsMatch(t, tc.mustRemove, store.removed)
			require.ElementsMatch(t, tc.mustSet, store.set)
		})
	}
}

type setOp struct {
	key   string
	value interface{}
}
type state struct {
	key   string
	value mapstr.M
}

type storeMock struct {
	backend.Store

	set     []setOp
	removed []string
	states  []state
}

func (s *storeMock) Set(key string, value interface{}) error {
	s.set = append(s.set, setOp{key: key, value: value})
	return nil
}

func (s *storeMock) Remove(key string) error {
	s.removed = append(s.removed, key)
	return nil
}

func (s *storeMock) Each(fn func(key string, value backend.ValueDecoder) (bool, error)) error {
	for _, s := range s.states {
		vd := mockValueDecoder{
			value: s.value,
		}
		ok, err := fn(s.key, vd)
		if !ok || err != nil {
			return err
		}
	}

	return nil
}

type mockValueDecoder struct {
	backend.ValueDecoder
	value mapstr.M
}

func (vd mockValueDecoder) Decode(to interface{}) error {
	toMap, ok := to.(*mapstr.M)
	if !ok || toMap == nil {
		panic("wrong value type for the mock value decoder")
	}
	*toMap = vd.value.Clone()
	return nil
}

type backuperMock struct {
	backup.Backuper

	called int
}

func (b *backuperMock) Backup() error {
	b.called++
	return nil
}
