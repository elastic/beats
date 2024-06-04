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
	"context"
	"errors"
	"flag"
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/coreos/go-systemd/v22/dbus"
	"github.com/coreos/go-systemd/v22/sdjournal"
	"github.com/urso/sderr"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalfield"
	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalread"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	cursor "github.com/elastic/beats/v7/filebeat/input/v2/input-cursor"
	"github.com/elastic/beats/v7/libbeat/common/backoff"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
)

var noVersionCheck bool

func init() {
	flag.BoolVar(&noVersionCheck,
		"ignore-journald-version",
		false,
		"Does not check Journald version when starting the Journald input. This might cause Filebeat to crash!")
}

type journald struct {
	Backoff            time.Duration
	MaxBackoff         time.Duration
	Since              *time.Duration
	Seek               journalread.SeekMode
	CursorSeekFallback journalread.SeekMode
	Matches            journalfield.IncludeMatches
	Units              []string
	Transports         []string
	Identifiers        []string
	SaveRemoteHostname bool
	Parsers            parser.Config
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

// ErrSystemdVersionNotSupported is returned by the plugin manager when the
// Systemd version is not supported.
var ErrSystemdVersionNotSupported = errors.New("systemd version must be >= 255")

// ErrCannotGetSystemdVersion is returned by the plugin manager when it is
// not possible to get the Systemd version via D-Bus.
var ErrCannotGetSystemdVersion = errors.New("cannot get systemd version")

// Plugin creates a new journald input plugin for creating a stateful input.
func Plugin(log *logp.Logger, store cursor.StateStore) input.Plugin {
	m := &cursor.InputManager{
		Logger:     log,
		StateStore: store,
		Type:       pluginName,
		Configure:  configure,
	}
	p := input.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
		Deprecated: false,
		Info:       "journald input",
		Doc:        "The journald input collects logs from the local journald service",
		Manager:    m,
	}

	if noVersionCheck {
		log.Warn("Journald version check has been DISABLED! Filebeat might crash if Journald version is < 255.")
		return p
	}

	version, err := systemdVersion()
	if err != nil {
		configErr := fmt.Errorf("%w: %s", ErrCannotGetSystemdVersion, err)
		m.Configure = func(_ *conf.C) ([]cursor.Source, cursor.Input, error) {
			return nil, nil, configErr
		}
		return p
	}

	if version < 255 {
		configErr := fmt.Errorf("%w. Systemd version: %d", ErrSystemdVersionNotSupported, version)
		m.Configure = func(_ *conf.C) ([]cursor.Source, cursor.Input, error) {
			return nil, nil, configErr
		}
		return p
	}

	return p
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
		Backoff:            config.Backoff,
		MaxBackoff:         config.MaxBackoff,
		Since:              config.Since,
		Seek:               config.Seek,
		CursorSeekFallback: config.CursorSeekFallback,
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
	currentCheckpoint := initCheckpoint(log, cursor)

	reader, err := inp.open(ctx.Logger, ctx.Cancelation, src)
	if err != nil {
		return err
	}
	defer reader.Close()

	mode, pos := seekBy(ctx.Logger, currentCheckpoint, inp.Seek, inp.CursorSeekFallback)
	if mode == journalread.SeekSince {
		err = reader.SeekRealtimeUsec(uint64(time.Now().Add(*inp.Since).UnixMicro()))
	} else {
		err = reader.Seek(mode, pos)
	}
	if err != nil {
		log.Error("Continue from current position. Seek failed with: %v", err)
	}

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
			return err
		}

		event := entry.ToEvent()
		if err := publisher.Publish(event, event.Private); err != nil {
			return err
		}
	}
}

func (inp *journald) open(log *logp.Logger, canceler input.Canceler, src cursor.Source) (*journalread.Reader, error) {
	backoff := backoff.NewExpBackoff(canceler.Done(), inp.Backoff, inp.MaxBackoff)
	reader, err := journalread.Open(log, src.Name(), backoff,
		withFilters(inp.Matches),
		withUnits(inp.Units),
		withTransports(inp.Transports),
		withSyslogIdentifiers(inp.Identifiers))
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

func withFilters(filters journalfield.IncludeMatches) func(*sdjournal.Journal) error {
	return func(j *sdjournal.Journal) error {
		return journalfield.ApplyIncludeMatches(j, filters)
	}
}

func withUnits(units []string) func(*sdjournal.Journal) error {
	return func(j *sdjournal.Journal) error {
		return journalfield.ApplyUnitMatchers(j, units)
	}
}

func withTransports(transports []string) func(*sdjournal.Journal) error {
	return func(j *sdjournal.Journal) error {
		return journalfield.ApplyTransportMatcher(j, transports)
	}
}

func withSyslogIdentifiers(identifiers []string) func(*sdjournal.Journal) error {
	return func(j *sdjournal.Journal) error {
		return journalfield.ApplySyslogIdentifierMatcher(j, identifiers)
	}
}

// seekBy tries to find the last known position in the journal, so we can continue collecting
// from the last known position.
// The checkpoint is ignored if the user has configured the input to always
// seek to the head/tail/since of the journal on startup.
func seekBy(log *logp.Logger, cp checkpoint, seek, defaultSeek journalread.SeekMode) (mode journalread.SeekMode, pos string) {
	mode = seek
	if mode == journalread.SeekCursor && cp.Position == "" {
		mode = defaultSeek
		switch mode {
		case journalread.SeekHead, journalread.SeekTail, journalread.SeekSince:
		default:
			log.Error("Invalid option for cursor_seek_fallback")
			mode = journalread.SeekHead
		}
	}
	return mode, cp.Position
}

// readerAdapter wraps journalread.Reader and adds two functionalities:
//   - Allows it to behave like a reader.Reader
//   - Translates the fields names from the journald format to something
//     more human friendly
type readerAdapter struct {
	r                  *journalread.Reader
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

	content := []byte(data.Fields["MESSAGE"])
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

// parseSystemdVersion parses the string version from Systemd fetched via D-Bus.
// The function will parse and return the 3 digit major version, minor version
// and patch are ignored.
func parseSystemdVersion(ver string) (int, error) {
	re := regexp.MustCompile(`(v)?(?P<version>\d\d\d)(\.)?`)
	matches := re.FindStringSubmatch(ver)
	if len(matches) == 0 {
		return 0, fmt.Errorf("unsupported Systemd version format '%s'", ver)
	}

	// This should never fail because the regexp ensures we're getting a 3-digt
	// integer, however, better safe than sorry.
	version, err := strconv.Atoi(matches[2])
	if err != nil {
		return 0, fmt.Errorf("could not convert '%s' to int: %w", matches[2], err)
	}

	return version, nil
}

// getSystemdVersionViaDBus gets the Systemd version from sd-bus
//
// The Systemd D-Bus documentation states:
//
//	 Version encodes the version string of the running systemd
//	 instance. Note that the version string is purely informational,
//	 it should not be parsed, one may not assume the version to be
//	 formatted in any particular way. We take the liberty to change
//	 the versioning scheme at any time and it is not part of the API.
//	Source: https://www.freedesktop.org/wiki/Software/systemd/dbus/
func getSystemdVersionViaDBus() (string, error) {
	// Get a context with timeout just to be on the safe side
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, err := dbus.NewSystemConnectionContext(ctx)
	if err != nil {
		return "", fmt.Errorf("cannot connect to sd-bus: %w", err)
	}

	version, err := conn.GetManagerProperty("Version")
	if err != nil {
		return "", fmt.Errorf("cannot get version property: %w", err)
	}

	return version, nil
}

func systemdVersion() (int, error) {
	versionStr, err := getSystemdVersionViaDBus()
	if err != nil {
		return 0, fmt.Errorf("caanot get Systemd version: %w", err)
	}

	version, err := parseSystemdVersion(versionStr)
	if err != nil {
		return 0, fmt.Errorf("cannot parse Systemd version: %w", err)
	}

	return version, nil
}
