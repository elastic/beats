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

// +build linux,cgo,withjournald

package journald

import (
	"time"

	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/urso/sderr"

	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/journalbeat/pkg/journalfield"
	"github.com/elastic/beats/v7/journalbeat/pkg/journalread"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
)

type journald struct {
	Backoff            time.Duration
	MaxBackoff         time.Duration
	Seek               journalread.SeekMode
	CursorSeekFallback journalread.SeekMode
	Matches            []journalfield.Matcher
	SaveRemoteHostname bool
}

type checkpoint struct {
	Version            int
	Position           string
	RealtimeTimestamp  uint64
	MonotonicTimestamp uint64
}

// LocalSystemJournalID is the ID of the local system journal.
const localSystemJournalID = "LOCAL_SYSTEM_JOURNAL"

const pluginName = "journald"

// Plugin creates a new journald input plugin for creating a stateful input.
func Plugin(log *logp.Logger, store cursor.StateStore) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "journald input",
		Doc:        "The journald input collects logs from the local journald service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

type pathSource string

var cursorVersion = 1

func (p pathSource) Name() string { return string(p) }

func configure(cfg *common.Config) ([]cursor.Source, cursor.Input, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	paths := config.Paths
	if len(paths) == 0 {
		paths = []string{localSystemJournalID}
	}

	sources := make([]cursor.Source, len(paths))
	for i, p := range paths {
		sources[i] = pathSource(p)
	}

	return sources, &journald{
		Backoff:            config.Backoff,
		MaxBackoff:         config.MaxBackoff,
		Seek:               config.Seek,
		CursorSeekFallback: config.CursorSeekFallback,
		Matches:            config.Matches,
		SaveRemoteHostname: config.SaveRemoteHostname,
	}, nil
}

func (inp *journald) Name() string { return pluginName }

func (inp *journald) Test(src cursor.Source, ctx input.TestContext) error {
	reader, err := inp.open(ctx.Logger, ctx.Cancelation, src)
	if err != nil {
		return err
	}
	return reader.Close()
}

func (inp *journald) Run(
	ctx input.Context,
	src cursor.Source,
	cursor cursor.Cursor,
	publisher cursor.Publisher,
) error {
	log := ctx.Logger.With("path", src.Name())
	checkpoint := initCheckpoint(log, cursor)

	reader, err := inp.open(ctx.Logger, ctx.Cancelation, src)
	if err != nil {
		return err
	}
	defer reader.Close()

	if err := reader.Seek(seekBy(ctx.Logger, checkpoint, inp.Seek, inp.CursorSeekFallback)); err != nil {
		log.Error("Continue from current position. Seek failed with: %v", err)
	}

	for {
		entry, err := reader.Next(ctx.Cancelation)
		if err != nil {
			return err
		}

		event := eventFromFields(ctx.Logger, entry.RealtimeTimestamp, entry.Fields, inp.SaveRemoteHostname)

		checkpoint.Position = entry.Cursor
		checkpoint.RealtimeTimestamp = entry.RealtimeTimestamp
		checkpoint.MonotonicTimestamp = entry.MonotonicTimestamp

		if err := publisher.Publish(event, checkpoint); err != nil {
			return err
		}
	}
}

func (inp *journald) open(log *logp.Logger, canceler input.Canceler, src cursor.Source) (*journalread.Reader, error) {
	backoff := backoff.NewExpBackoff(canceler.Done(), inp.Backoff, inp.MaxBackoff)
	reader, err := journalread.Open(log, src.Name(), backoff, withFilters(inp.Matches))
	if err != nil {
		return nil, sderr.Wrap(err, "failed to create reader for %{path} journal", src.Name())
	}

	return reader, nil
}

func initCheckpoint(log *logp.Logger, c cursor.Cursor) checkpoint {
	if c.IsNew() {
		return checkpoint{Version: cursorVersion}
	}

	var cp checkpoint
	err := c.Unpack(&cp)
	if err != nil {
		log.Errorf("Reset journald position. Failed to read checkpoint from registry: %v", err)
		return checkpoint{Version: cursorVersion}
	}

	if cp.Version != cursorVersion {
		log.Error("Reset journald position. invalid journald position entry.")
		return checkpoint{Version: cursorVersion}
	}

	return cp
}

func withFilters(filters []journalfield.Matcher) func(*sdjournal.Journal) error {
	return func(j *sdjournal.Journal) error {
		return journalfield.ApplyMatchersOr(j, filters)
	}
}

// seekBy tries to find the last known position in the journal, so we can continue collecting
// from the last known position.
// The checkpoint is ignored if the user has configured the input to always
// seek to the head/tail of the journal on startup.
func seekBy(log *logp.Logger, cp checkpoint, seek, defaultSeek journalread.SeekMode) (journalread.SeekMode, string) {
	mode := seek
	if mode == journalread.SeekCursor && cp.Position == "" {
		mode = defaultSeek
		if mode != journalread.SeekHead && mode != journalread.SeekTail {
			log.Error("Invalid option for cursor_seek_fallback")
			mode = journalread.SeekHead
		}
	}
	return mode, cp.Position
}
