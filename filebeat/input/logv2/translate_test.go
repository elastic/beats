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

package logv2

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/filebeat/input/filestream"
	"github.com/elastic/elastic-agent-libs/config"
)

func TestDirectTranslate(t *testing.T) {

	logYaml := `
id: foo
paths:
  - /var/log/*.log
  - /foo/bar.log
recursive_glob.enabled: false
encoding: "utf-8"
exclude_lines:
  - "^DBG"
include_lines:
  - "^ERR"
  - "^WARN"
harvester_buffer_size: 42000
max_bytes: 44000
json:
  keys_under_root: true
  overwrite_keys: true
  expand_keys: true
  add_error_key: true
  message_key: "message"
  document_id: "the_id_key"
  ignore_decoding_error: true
multiline.type: pattern
multiline.pattern: "^\\["
multiline.negate: true
multiline.match: after
exclude_files:
  - "\\.gz$"
ignore_older: 10h
clean_inactive: 20h
clean_removed: false
close_eof: true
close_inactive: 3h
close_removed: false
close_renamed: true
close_timeout: 42s
scan_frequency: 50s
tail_files: true
symlinks: true
backoff: 20s
max_backoff: 200s
harvester_limit: 10000
`
	fsCfg := `
{
  "allow_deprecated_id_duplication": false,
  "backoff": {
    "init": "20s",
    "max": "3m20s"
  },
  "buffer_size": 42000,
  "clean_inactive": "20h",
  "clean_removed": false,
  "close": {
    "on_state_change": {
      "check_interval": "5s",
      "inactive": "3h0m0s",
      "removed": false,
      "renamed": true
    },
    "reader": {
      "after_interval": "42s",
      "on_eof": true
    }
  },
  "delete": {
    "enabled": false,
    "grace_period": "30m0s"
  },
  "encoding": "utf-8",
  "exclude_lines": [
    "^DBG"
  ],
  "gzip_experimental": false,
  "harvester_limit": 10000,
  "id": "foo",
  "ignore_inactive": "since_last_start",
  "ignore_older": "10h",
  "include_lines": [
    "^ERR",
    "^WARN"
  ],
  "legacy_clean_inactive": false,
  "message_max_bytes": 44000,
  "parsers": [
    {
      "ndjson": {
        "add_error_key": true,
        "document_id": "the_id_key",
        "expand_keys": true,
        "field":"",
        "ignore_decoding_error": true,
        "keys_under_root": true,
        "message_key": "message",
        "overwrite_keys": true,
        "target": ""
      }
    },
    {
      "multiline": {
        "match": "after",
        "negate": true,
        "pattern": "^\\[",
        "type": "pattern"
      }
    }
  ],
  "paths": [
    "/var/log/*.log",
    "/foo/bar.log"
  ],
  "prospector": {
    "scanner": {
      "-": false,
      "check_interval": "50s",
      "exclude_files": [
        "\\.gz$"
      ],
      "fingerprint": {
        "enabled": true,
        "length": 1024,
        "offset": 0
      },
      "include_files": [],
      "recursive_glob": false,
      "resend_on_touch": false,
      "symlinks": true
    }
  },
  "seek_to_tail": false,
  "suffix": "",
  "take_over": {
    "enabled": true,
    "from_ids": []
  }
}
`

	srcCfg := config.MustNewConfigFrom(logYaml)

	newCfg, err := translateCfg(srcCfg)
	if err != nil {
		t.Fatalf("cannot translate config: %s", err)
	}

	// The keys from the log input will affect the comparison, remove them
	for _, key := range logInputExclusiveKeys {
		_, err := newCfg.Remove(key, -1)
		if err != nil {
			t.Fatalf("cannot remove '%s': %s", key, err)
		}
	}

	convertedConfig := config.DebugString(newCfg, false)
	// t.Log("========================= New Config =========================")
	// t.Log(convertedConfig)

	// Validate we can convert the struct
	fsCfgStruct := filestream.Config{}
	if err := newCfg.Unpack(&fsCfgStruct); err != nil {
		t.Fatalf("cannot unpack translated config: %s", err)
	}

	require.JSONEq(t, fsCfg, convertedConfig, "configs are not equal")
}
