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

//go:build linux

package journald

import (
	"errors"
	"fmt"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

//go:generate moq -out journalReadMock_test.go . journalReader
type journalReader interface {
	Close() error
	Next(cancel input.Canceler) (journalctl.JournalEntry, error)
}

type journald struct {
	Backoff            time.Duration
	MaxBackoff         time.Duration
	Since              time.Duration
	Seek               journalctl.SeekMode
	Matches            journalfield.IncludeMatches
	Units              []string
	Transports         []string
	Identifiers        []string
	SaveRemoteHostname bool
	Parsers            parser.Config
	Journalctl         bool
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

func configure(cfg *conf.C) ([]cursor.Source, cursor.Input, error) {
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
		Since:              config.Since,
		Seek:               config.Seek,
		Matches:            journalfield.IncludeMatches(config.Matches),
		Units:              config.Units,
		Transports:         config.Transports,
		Identifiers:        config.Identifiers,
		SaveRemoteHostname: config.SaveRemoteHostname,
		Parsers:            config.Parsers,
	}, nil
}

func (inp *journald) Name() string { return pluginName }

func (inp *journald) Test(src cursor.Source, ctx input.TestContext) error {
	reader, err := journalctl.New(
		ctx.Logger,
		ctx.Cancelation,
		inp.Units,
		inp.Identifiers,
		inp.Transports,
		inp.Matches,
		journalctl.SeekHead,
		"",
		inp.Since,
		src.Name(),
	)
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
	logger := ctx.Logger.With("path", src.Name())
	currentCheckpoint := initCheckpoint(logger, cursor)

	mode := inp.Seek
	pos := currentCheckpoint.Position
	reader, err := journalctl.New(
		logger,
		ctx.Cancelation,
		inp.Units,
		inp.Identifiers,
		inp.Transports,
		inp.Matches,
		mode,
		pos,
		inp.Since,
		src.Name(),
	)
	if err != nil {
		return fmt.Errorf("could not start journal reader: %w", err)
	}

	defer reader.Close()

	parser := inp.Parsers.Create(
		&readerAdapter{
			r:                  reader,
			converter:          journalfield.NewConverter(ctx.Logger, nil),
			canceler:           ctx.Cancelation,
			saveRemoteHostname: inp.SaveRemoteHostname,
		})

	for {
		entry, err := parser.Next()
		if err != nil {
			// The input has been cancelled, gracefully return
			if errors.Is(err, journalctl.ErrCancelled) {
				return nil
			}
			logger.Errorf("could not read event: %s", err)
			return err
		}

		event := entry.ToEvent()
		if err := publisher.Publish(event, event.Private); err != nil {
			logger.Errorf("could not publish event: %s", err)
			return err
		}
	}
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

// readerAdapter wraps journalread.Reader and adds two functionalities:
//   - Allows it to behave like a reader.Reader
//   - Translates the fields names from the journald format to something
//     more human friendly
type readerAdapter struct {
	r                  journalReader
	canceler           input.Canceler
	converter          *journalfield.Converter
	saveRemoteHostname bool
}

func (r *readerAdapter) Close() error {
	return r.r.Close()
}

func (r *readerAdapter) Next() (reader.Message, error) {
	data, err := r.r.Next(r.canceler)
	if err != nil {
		return reader.Message{}, err
	}

	created := time.Now()

	// Journald documents that 'MESSAGE' is always a string,
	// see https://www.man7.org/linux/man-pages/man7/systemd.journal-fields.7.html.
	// However while testing 'journalctl -o json' outputs the 'MESSAGE'
	// like [1, 2, 3, 4]. Which seems to be the result of a binary encoding
	// of a journal field (see https://systemd.io/JOURNAL_NATIVE_PROTOCOL/).
	//
	// Trying to be smart and convert the contents into string
	// byte by byte did not work well because one test case contained
	// control characters and new line characters.
	// To avoid issues later in the ingestion pipeline we just convert
	// the whole thing to a string using fmt.Sprint.
	//
	// Look at 'pkg/journalctl/testdata/corner-cases.json'
	// for some real world examples.
	msg := data.Fields["MESSAGE"]
	msgStr, isString := msg.(string)
	if !isString {
		msgStr = fmt.Sprint(msg)
	}
	content := []byte(msgStr)
	delete(data.Fields, "MESSAGE")

	fields := r.converter.Convert(data.Fields)
	fields.Put("event.kind", "event")
	fields.Put("event.created", created)

	// if entry is coming from a remote journal, add_host_metadata overwrites
	// the source hostname, so it has to be copied to a different field
	if r.saveRemoteHostname {
		remoteHostname, err := fields.GetValue("host.hostname")
		if err == nil {
			fields.Put("log.source.address", remoteHostname)
		}
	}

	m := reader.Message{
		Ts:      time.UnixMicro(int64(data.RealtimeTimestamp)),
		Content: content,
		Bytes:   len(content),
		Fields:  fields,
		Private: checkpoint{
			Version:            cursorVersion,
			RealtimeTimestamp:  data.RealtimeTimestamp,
			MonotonicTimestamp: data.MonotonicTimestamp,
			Position:           data.Cursor,
		},
	}

	return m, nil
}
