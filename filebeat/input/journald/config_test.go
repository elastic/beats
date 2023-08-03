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

//go:build linux && cgo && withjournald

package journald

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jr "github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalread"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestConfigIncludeMatches(t *testing.T) {
	verify := func(t *testing.T, yml string) {
		t.Helper()

		c, err := conf.NewConfigWithYAML([]byte(yml), "source")
		require.NoError(t, err)

		config := defaultConfig()
		require.NoError(t, c.Unpack(&config))

		assert.EqualValues(t, "_SYSTEMD_UNIT=foo.service", config.Matches.OR[0].Matches[0].String())
		assert.EqualValues(t, "_SYSTEMD_UNIT=bar.service", config.Matches.OR[1].Matches[0].String())
	}

	t.Run("normal", func(t *testing.T) {
		const yaml = `
include_matches:
  or:
  - match: _SYSTEMD_UNIT=foo.service
  - match: _SYSTEMD_UNIT=bar.service
`
		verify(t, yaml)
	})

	t.Run("backwards-compatible", func(t *testing.T) {
		const yaml = `
include_matches:
  - _SYSTEMD_UNIT=foo.service
  - _SYSTEMD_UNIT=bar.service
`

		verify(t, yaml)
	})
}

func TestConfigValidate(t *testing.T) {
	t.Run("table", func(t *testing.T) {

		nameOf := [...]string{
			jr.SeekInvalid: "invalid",
			jr.SeekHead:    "head",
			jr.SeekTail:    "tail",
			jr.SeekCursor:  "cursor",
			jr.SeekSince:   "since",
		}

		modes := []jr.SeekMode{
			jr.SeekInvalid,
			jr.SeekHead,
			jr.SeekTail,
			jr.SeekCursor,
			jr.SeekSince,
		}
		const n = jr.SeekSince + 1

		errSeek := errInvalidSeek
		errFall := errInvalidSeekFallback
		errSince := errInvalidSeekSince
		// Want is the tables of expectations: seek in major, fallback in minor.
		want := map[bool][n][n]error{
			false: { // No since option set.
				jr.SeekInvalid: {jr.SeekInvalid: errSeek, jr.SeekHead: errSeek, jr.SeekTail: errSeek, jr.SeekCursor: errSeek, jr.SeekSince: errSeek},
				jr.SeekHead:    {jr.SeekInvalid: errFall, jr.SeekHead: nil, jr.SeekTail: nil, jr.SeekCursor: errFall, jr.SeekSince: nil},
				jr.SeekTail:    {jr.SeekInvalid: errFall, jr.SeekHead: nil, jr.SeekTail: nil, jr.SeekCursor: errFall, jr.SeekSince: nil},
				jr.SeekCursor:  {jr.SeekInvalid: errFall, jr.SeekHead: nil, jr.SeekTail: nil, jr.SeekCursor: errFall, jr.SeekSince: errSince},
				jr.SeekSince:   {jr.SeekInvalid: errFall, jr.SeekHead: errSince, jr.SeekTail: errSince, jr.SeekCursor: errFall, jr.SeekSince: errSince},
			},
			true: { // Since option set.
				jr.SeekInvalid: {jr.SeekInvalid: errSeek, jr.SeekHead: errSeek, jr.SeekTail: errSeek, jr.SeekCursor: errSeek, jr.SeekSince: errSeek},
				jr.SeekHead:    {jr.SeekInvalid: errFall, jr.SeekHead: errSince, jr.SeekTail: errSince, jr.SeekCursor: errFall, jr.SeekSince: errSince},
				jr.SeekTail:    {jr.SeekInvalid: errFall, jr.SeekHead: errSince, jr.SeekTail: errSince, jr.SeekCursor: errFall, jr.SeekSince: errSince},
				jr.SeekCursor:  {jr.SeekInvalid: errFall, jr.SeekHead: errSince, jr.SeekTail: errSince, jr.SeekCursor: errFall, jr.SeekSince: nil},
				jr.SeekSince:   {jr.SeekInvalid: errFall, jr.SeekHead: nil, jr.SeekTail: nil, jr.SeekCursor: errFall, jr.SeekSince: nil},
			},
		}

		for setSince := range want {
			for _, seek := range modes {
				for _, fallback := range modes {
					name := fmt.Sprintf("seek_%s_fallback_%s_since_%t", nameOf[seek], nameOf[fallback], setSince)
					t.Run(name, func(t *testing.T) {
						cfg := config{Seek: seek, CursorSeekFallback: fallback}
						if setSince {
							cfg.Since = new(time.Duration)
						}
						got := cfg.Validate()
						if !errors.Is(got, want[setSince][seek][fallback]) {
							t.Errorf("unexpected error: got:%v want:%v", got, want[setSince][seek][fallback])
						}
					})
				}
			}
		}
	})

	t.Run("use", func(t *testing.T) {
		logger := logp.L()
		for seek := jr.SeekInvalid; seek <= jr.SeekSince+1; seek++ {
			for seekFallback := jr.SeekInvalid; seekFallback <= jr.SeekSince+1; seekFallback++ {
				for _, since := range []*time.Duration{nil, new(time.Duration)} {
					for _, pos := range []string{"", "defined"} {
						// Construct a config with fields checked by Validate.
						cfg := config{
							Since:              since,
							Seek:               seek,
							CursorSeekFallback: seekFallback,
						}
						if err := cfg.Validate(); err != nil {
							continue
						}

						// Confirm we never get to seek since mode with a nil since.
						cp := checkpoint{Position: pos}
						mode, _ := seekBy(logger, cp, cfg.Seek, cfg.CursorSeekFallback)
						if mode == jr.SeekSince {
							if cfg.Since == nil {
								// If we reach here we would have panicked in Run.
								t.Errorf("got nil since in valid seek since mode: seek=%d seek_fallback=%d since=%d cp=%+v",
									seek, seekFallback, since, cp)
							}
						}
					}
				}
			}
		}
	})
}
