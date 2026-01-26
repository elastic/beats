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
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/management/status"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

//go:generate moq -out journalReadMock_test.go . journalReader
type journalReader interface {
	Close() error
	Next(cancel input.Canceler) (journalctl.JournalEntry, error)
}

type journald struct {
	ID                 string
	Backoff            time.Duration
	MaxBackoff         time.Duration
	Since              time.Duration
	Seek               journalctl.SeekMode
	Matches            journalfield.IncludeMatches
	Units              []string
	Transports         []string
	Identifiers        []string
	Facilities         []int
	SaveRemoteHostname bool
	Parsers            parser.Config
	Merge              bool
	Chroot             string
	JournalctlPath     string
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
func Plugin(log *logp.Logger, store statestore.States) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "journald input",
		Doc:        "The journald input collects logs from the local journald service",
		Manager: &cursor.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  Configure,
		},
	}
}

type pathSource string

var cursorVersion = 1

func (p pathSource) Name() string { return string(p) }

func Configure(cfg *conf.C, _ *logp.Logger) ([]cursor.Source, cursor.Input, error) {
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

	if config.Chroot != "" {
		// When using chroot we need absolute paths for the journalctl binary
		// (see https://github.com/golang/go/issues/39341). However for a
		// normal operation we look for 'journalctl' in the PATH.
		// So if chroot is set, change the default value.
		if config.JournalctlPath == defaultJournalCtlPath {
			config.JournalctlPath = defaultJournalCtlPathChroot
		}
		chrootStat, err := os.Stat(config.Chroot)
		if err != nil {
			return nil, nil, fmt.Errorf("cannot stat chroot:  %w", err)
		}
		if !chrootStat.IsDir() {
			return nil, nil, fmt.Errorf("provided chroot (%s) is not a directory", config.Chroot)
		}

		if !filepath.IsAbs(config.JournalctlPath) {
			return nil, nil, errors.New("journalctl_path must be an absolute path when chroot is set")
		}

		fullPath := filepath.Join(config.Chroot, config.JournalctlPath)
		if _, err := os.Stat(fullPath); err != nil {
			return nil, nil, fmt.Errorf("cannot stat journalctl binary in chroot: %w", err)
		}
	}

	return sources, &journald{
		ID:                 config.ID,
		Since:              config.Since,
		Seek:               config.Seek,
		Matches:            journalfield.IncludeMatches(config.Matches),
		Units:              config.Units,
		Transports:         config.Transports,
		Identifiers:        config.Identifiers,
		Facilities:         config.Facilities,
		SaveRemoteHostname: config.SaveRemoteHostname,
		Parsers:            config.Parsers,
		Merge:              config.Merge,
		Chroot:             config.Chroot,
		JournalctlPath:     config.JournalctlPath,
	}, nil
}

func (inp *journald) Name() string { return pluginName }

func (inp *journald) Test(src cursor.Source, ctx input.TestContext) error {
	reader, err := journalctl.New(
		ctx.Logger.With("input_id", inp.ID),
		ctx.Cancelation,
		inp.Units,
		inp.Identifiers,
		inp.Transports,
		inp.Matches,
		inp.Facilities,
		journalctl.SeekHead,
		"",
		inp.Since,
		src.Name(),
		inp.Merge,
		journalctl.NewFactory(inp.Chroot, inp.JournalctlPath),
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
	logger := ctx.Logger.
		With("path", src.Name()).
		With("input_id", inp.ID)

	ctx.UpdateStatus(status.Starting, "Starting")
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
		inp.Facilities,
		mode,
		pos,
		inp.Since,
		src.Name(),
		inp.Merge,
		journalctl.NewFactory(inp.Chroot, inp.JournalctlPath),
	)
	if err != nil {
		wrappedErr := fmt.Errorf("could not start journal reader: %w", err)
		ctx.UpdateStatus(status.Failed, wrappedErr.Error())
		return wrappedErr
	}

	defer reader.Close()

	parser := inp.Parsers.Create(
		&readerAdapter{
			r:                  reader,
			converter:          journalfield.NewConverter(ctx.Logger, nil),
			canceler:           ctx.Cancelation,
			saveRemoteHostname: inp.SaveRemoteHostname,
		}, logger)

	ctx.UpdateStatus(status.Running, "Running")
	for {
		entry, err := parser.Next()
		if err != nil {
			switch {
			// The input has been cancelled, gracefully return
			case errors.Is(err, journalctl.ErrCancelled):
				return nil
				// Journalctl is restarting, do ignore the empty event
			case errors.Is(err, journalctl.ErrRestarting):
				continue
			default:
				msg := fmt.Sprintf("could not read event: %s", err)
				ctx.UpdateStatus(status.Failed, msg)
				logger.Error(msg)
				return err
			}
		}

		event := entry.ToEvent()
		if err := publisher.Publish(event, event.Private); err != nil {
			msg := fmt.Sprintf("could not publish event: %s", err)
			ctx.UpdateStatus(status.Failed, msg)
			logger.Errorf("%s", msg)
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
	// like [1, 2, 3, 4]. Which is the result of a binary encoding of a journal
	// field (see https://systemd.io/JOURNAL_NATIVE_PROTOCOL/).
	//
	// The binary encoding is used when a '\n' is present in the field or when
	// some unprintable bytes are part of the message.
	//
	// When outputting as JSON journalctl already parses the binary
	// representation and gives us only the data, the size is not present
	// any more.
	//
	// So in order to not send slices of bytes in the message, we check if
	// 'MESSAGE' is a string or a slice, if it is a slice, we
	// safely convert it to a []byte, then convert it to a string.
	//
	// Look at 'pkg/journalctl/testdata/corner-cases.json'
	// for some real world examples or at testdata/binary.export for some
	// hand crafted ones.
	var content []byte
	failed := false
	switch msg := data.Fields["MESSAGE"].(type) {
	case string:
		content = []byte(msg)
	case []any:
		// MESSAGE can be a byte array, in its JSON representation, it is a
		// []any where all elements are float64.
		// Safely convert it to a []byte
		content = make([]byte, len(msg))
		for i, v := range msg {
			if b, ok := v.(float64); ok {
				content[i] = byte(b)
			} else {
				failed = true
				break
			}
		}
	default:
		// This should never happen, but just in case we fall back to just
		// getting a string representation using the `fmt` package.
		failed = true
	}

	if failed {
		content = fmt.Append([]byte{}, data.Fields["MESSAGE"])
	}

	delete(data.Fields, "MESSAGE")

	fields := r.converter.Convert(data.Fields)
	fields.Put("event.kind", "event")
	fields.Put("event.created", created)

	// IF 'container.partial' is present, we can parse it and it's true, then
	// add 'partial_message' to tags.
	if partialMessageRaw, err := fields.GetValue("container.partial"); err == nil {
		partialMessage, err := strconv.ParseBool(fmt.Sprint(partialMessageRaw))
		if err == nil && partialMessage {
			// 'fields' came directly from the journal,
			// so there is no chance tags already exist
			fields.Put("tags", []string{"partial_message"})
		}
	}

	// Delete 'container.partial', if there are any errors, ignore it
	_ = fields.Delete("container.partial")

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
