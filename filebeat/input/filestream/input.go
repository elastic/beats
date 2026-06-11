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

package filestream

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"go.uber.org/zap"
	"golang.org/x/text/transform"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/debug"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/v7/libbeat/statestore"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-concert/ctxtool"
)

const pluginName = "filestream"

type state struct {
	Offset int64 `json:"offset" struct:"offset"`
}

type fileMeta struct {
	Source         string `json:"source" struct:"source"`
	IdentifierName string `json:"identifier_name" struct:"identifier_name"`
}

// filestream is the input for reading from files which
// are actively written by other applications.
type filestream struct {
<<<<<<< HEAD
	readerConfig    readerConfig
	encodingFactory encoding.EncodingFactory
	closerConfig    closerConfig
	parsers         parser.Config
	takeOver        bool
=======
	readerConfig              readerConfig
	encodingFactory           encoding.EncodingFactory
	closerConfig              closerConfig
	deleterConfig             deleterConfig
	parsers                   parser.Config
	takeOver                  loginp.TakeOverConfig
	scannerCheckInterval      time.Duration
	readUntilEOF              loginp.ReadUntilEOFConfig
	compression               string
	includeFileOwnerName      bool
	includeFileOwnerGroupName bool
	hasLineFilter             bool

	// Function references for testing
	waitGracePeriodFn func(
		ctx input.Context,
		logger *logp.Logger,
		cursor loginp.Cursor,
		path string,
		gracePeriod, checkInterval time.Duration,
		statFn func(string) (os.FileInfo, error),
	) (bool, error)
	tickFn   func(time.Duration) <-chan time.Time
	removeFn func(string) error
	statFn   func(string) (os.FileInfo, error)
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
}

// Plugin creates a new filestream input plugin for creating a stateful input.
func Plugin(log *logp.Logger, store statestore.States) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "filestream input",
		Doc:        "The filestream input collects logs from the local filestream service",
		Manager: &loginp.InputManager{
			Logger:              log,
			StateStore:          store,
			Type:                pluginName,
			Configure:           configure,
			DefaultCleanTimeout: -1,
		},
	}
}

func configure(
	cfg *conf.C,
	log *logp.Logger,
	src *loginp.SourceIdentifier) (loginp.Prospector, loginp.Harvester, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	prospector, err := newProspector(config, log, src)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create prospector: %w", err)
	}

	encodingFactory, ok := encoding.FindEncoding(config.Reader.Encoding)
	if !ok || encodingFactory == nil {
		return nil, nil, fmt.Errorf("unknown encoding('%v')", config.Reader.Encoding)
	}

	filestream := &filestream{
<<<<<<< HEAD
		readerConfig:    config.Reader,
		encodingFactory: encodingFactory,
		closerConfig:    config.Close,
		parsers:         config.Reader.Parsers,
		takeOver:        config.TakeOver,
=======
		readerConfig:              c.Reader,
		encodingFactory:           encodingFactory,
		closerConfig:              c.Close,
		readUntilEOF:              c.ReadUntilEOF,
		parsers:                   c.Reader.Parsers,
		takeOver:                  c.TakeOver,
		compression:               c.Compression,
		includeFileOwnerName:      c.IncludeFileOwnerName,
		includeFileOwnerGroupName: c.IncludeFileOwnerGroupName,
		hasLineFilter:             len(c.Reader.IncludeLines) > 0 || len(c.Reader.ExcludeLines) > 0,
		deleterConfig:             c.Delete,
		waitGracePeriodFn:         waitGracePeriod,
		tickFn:                    time.Tick,
		removeFn:                  os.Remove,
		statFn:                    os.Stat,
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
	}

	return prospector, filestream, nil
}

func (inp *filestream) Name() string { return pluginName }

func (inp *filestream) Test(src loginp.Source, ctx input.TestContext) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	reader, _, _, err := inp.open(ctx.Logger, ctx.Cancelation, fs, 0)
	if err != nil {
		return err
	}
	return reader.Close()
}

func (inp *filestream) Run(
	ctx input.Context,
	src loginp.Source,
	cursor loginp.Cursor,
	publisher loginp.Publisher,
	metrics *loginp.Metrics) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	log := ctx.Logger.WithLazy(zap.String("path", fs.newPath), zap.String("state-id", src.Name()))
	state := initState(log, cursor, fs)

	// The reader is tied to ctx.Cancelation so it exits promptly on shutdown
	// (upstream behavior). When read_until_eof is enabled, it "resets" the
	// reader via startReadUntilEOF by swapping in a fresh, read_until_eof-scoped
	// context so the drain read can proceed past ctx.Cancelation.
	r, startReadUntilEOF, truncated, err := inp.open(log, ctx.Cancelation, fs, state.Offset)
	if err != nil {
		log.Errorf("File could not be opened for reading: %v", err)
		return err
	}

	if truncated {
		state.Offset = 0
	}

	metrics.FilesActive.Inc()
	metrics.HarvesterRunning.Inc()
	defer metrics.FilesActive.Dec()
	defer metrics.HarvesterRunning.Dec()

	defer func() {
		log.Debug("Closing reader of filestream")
<<<<<<< HEAD
		err := r.Close()
		if err != nil {
			log.Errorf("Error stopping filestream reader %v", err)
=======
		if err := r.Close(); err != nil {
			log.Errorf("Error stopping filestream reader: %v", err)
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
		}
	}()

	// The caller of Run already reports the error and filters out errors that
	// must not be reported, like 'context cancelled'.
<<<<<<< HEAD
	err = inp.readFromSource(ctx, log, r, fs.newPath, state, publisher, metrics)
=======
	err = inp.readFromSource(
		ctx, log, r, fs.newPath, state, publisher, fs.desc.GZIP, metrics,
		startReadUntilEOF)
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
	if err != nil {
		// First handle actual errors
		if !errors.Is(err, io.EOF) && !errors.Is(err, ErrInactive) {
			return fmt.Errorf("error reading from source: %w", err)
		}
	}

	return nil
}

func initState(log *logp.Logger, c loginp.Cursor, s fileSource) state {
	var state state
	if c.IsNew() || s.truncated {
		return state
	}

	err := c.Unpack(&state)
	if err != nil {
		log.Error("Cannot serialize cursor data into file state: %+v", err)
	}

	return state
}

func (inp *filestream) open(
	log *logp.Logger,
	canceler input.Canceler,
	fs fileSource,
	offset int64,
) (reader.Reader, func(ctxtool.CancelContext), bool, error) {

	f, encoding, truncated, err := inp.openFile(log, fs.newPath, offset)
	if err != nil {
		return nil, nil, truncated, err
	}

	if truncated {
		offset = 0
	}

	ok := false // used for cleanup
	defer cleanup.IfNot(&ok, cleanup.IgnoreError(f.Close))

	log.Debug("newLogFileReader with config.MaxBytes:", inp.readerConfig.MaxBytes)

	// if the file is archived, it means that it is not going to be updated in the future
	// thus, when EOF is reached, it can be closed
	closerCfg := inp.closerConfig
	if fs.archived && !inp.closerConfig.Reader.OnEOF {
		closerCfg = closerConfig{
			Reader: readerCloserConfig{
				OnEOF:         true,
				AfterInterval: inp.closerConfig.Reader.AfterInterval,
			},
			OnStateChange: inp.closerConfig.OnStateChange,
		}
	}
	// NewLineReader uses additional buffering to deal with encoding and testing
	// for new lines in input stream. Simple 8-bit based encodings, or plain
	// don't require 'complicated' logic.
	logReader, startReadUntilEOF, err := newFileReader(
		log, canceler, f, inp.readerConfig, closerCfg, inp.readUntilEOF.Enabled)
	if err != nil {
		return nil, nil, truncated, err
	}

	dbgReader, err := debug.AppendReaders(logReader, log)
	if err != nil {
		return nil, nil, truncated, err
	}

	// Configure MaxBytes limit for EncodeReader as multiplied by 4
	// for the worst case scenario where incoming UTF32 charchers are decoded to the single byte UTF-8 characters.
	// This limit serves primarily to avoid memory bload or potential OOM with expectedly long lines in the file.
	// The further size limiting is performed by LimitReader at the end of the readers pipeline as needed.
	encReaderMaxBytes := inp.readerConfig.MaxBytes * 4

	var r reader.Reader
	r, err = readfile.NewEncodeReader(dbgReader, readfile.Config{
		Codec:      encoding,
		BufferSize: inp.readerConfig.BufferSize,
		Terminator: inp.readerConfig.LineTerminator,
		MaxBytes:   encReaderMaxBytes,
	}, log)
	if err != nil {
		return nil, nil, truncated, err
	}

	r = readfile.NewStripNewline(r, inp.readerConfig.LineTerminator)

	r = readfile.NewFilemeta(r, fs.newPath, fs.desc.Info, fs.desc.Fingerprint, offset)

	r = inp.parsers.Create(r, log)

	r = readfile.NewLimitReader(r, inp.readerConfig.MaxBytes)

<<<<<<< HEAD
	ok = true // no need to close the file
	return r, truncated, nil
=======
	if f.IsGZIP() {
		r = NewEOFLookaheadReader(r, io.EOF)
	}

	ok = true // used for cleanup: no need to close the file
	return r, startReadUntilEOF, truncated, nil
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned.
//
// openFile will also detect and hadle file truncation. If a file is truncated
// then the 3rd return value is true.
func (inp *filestream) openFile(
	log *logp.Logger,
	path string,
	offset int64,
) (*os.File, encoding.Encoding, bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to stat source file %s: %w", path, err)
	}

	// it must be checked if the file is not a named pipe before we try to open it
	// if it is a named pipe os.OpenFile fails, so there is no need to try opening it.
	if fi.Mode()&os.ModeNamedPipe != 0 {
		return nil, nil, false, fmt.Errorf("failed to open file %s, named pipes are not supported", fi.Name())
	}

	f, err := file.ReadOpen(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed opening %s: %w", path, err)
	}
	ok := false
	defer cleanup.IfNot(&ok, cleanup.IgnoreError(f.Close))

	fi, err = f.Stat()
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to stat source file %s: %w", path, err)
	}

	err = checkFileBeforeOpening(fi)
	if err != nil {
		return nil, nil, false, err
	}

	truncated := false
	if fi.Size() < offset {
		// if the file was truncated we need to reset the offset and notify
		// all callers so they can also reset their offsets
		truncated = true
		log.Infof("File was truncated. Reading file from offset 0. Path=%s", path)
		offset = 0
	}
	err = inp.initFileOffset(f, offset)
	if err != nil {
		return nil, nil, truncated, err
	}

	encoding, err := inp.encodingFactory(f)
	if err != nil {
		if errors.Is(err, transform.ErrShortSrc) {
			return nil, nil, truncated, fmt.Errorf("initialising encoding for '%v' failed due to file being too short", f)
		}
		return nil, nil, truncated, fmt.Errorf("initialising encoding for '%v' failed: %w", f, err)
	}

	ok = true // no need to close the file
	return f, encoding, truncated, nil
}

func checkFileBeforeOpening(fi os.FileInfo) error {
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("tried to open non regular file: %q %s", fi.Mode(), fi.Name())
	}

	return nil
}

func (inp *filestream) initFileOffset(file *os.File, offset int64) error {
	if offset > 0 {
		_, err := file.Seek(offset, io.SeekCurrent)
		return err
	}

	// get offset from file in case of encoding factory was required to read some data.
	_, err := file.Seek(0, io.SeekCurrent)
	return err
}

func (inp *filestream) readFromSource(
	ctx input.Context,
	log *logp.Logger,
	r reader.Reader,
	path string,
	s state,
	p loginp.Publisher,
<<<<<<< HEAD
	metrics *loginp.Metrics,
) error {
=======
	isGZIP bool,
	metrics *loginp.Metrics,
	startReadUntilEOF func(ctxtool.CancelContext)) error {

>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
	metrics.FilesOpened.Inc()
	metrics.HarvesterOpenFiles.Inc()
	metrics.HarvesterStarted.Inc()
	defer metrics.FilesClosed.Inc()
	defer metrics.HarvesterOpenFiles.Dec()
	defer metrics.HarvesterClosed.Inc()

<<<<<<< HEAD
	for ctx.Cancelation.Err() == nil {
		message, err := r.Next()
		if err != nil {
			if errors.Is(err, ErrFileTruncate) {
				log.Infof("File was truncated, nothing to read. Path='%s'", path)
			} else if errors.Is(err, ErrClosed) {
				log.Debugf("Reader was closed. Closing. Path='%s'", path)
			} else if errors.Is(err, io.EOF) {
				log.Debugf("EOF has been reached. Closing. Path='%s'", path)
			} else if errors.Is(err, ErrInactive) {
				log.Debugf("File is inactive. Closing. Path='%s'", path)
				return err
			} else {
				log.Errorf("Read line error: %v", err)
				metrics.ProcessingErrors.Inc()
			}

			return nil
		}

		s.Offset += int64(message.Bytes) + int64(message.Offset)

		flags, err := message.Fields.GetValue("log.flags")
		if err == nil {
			if flags, ok := flags.([]string); ok {
				if slices.Contains(flags, "truncated") { //nolint:typecheck,nolintlint // linter fails to infer generics
					metrics.MessagesTruncated.Add(1)
				}
			}
		}

		metrics.MessagesRead.Inc()
		if message.IsEmpty() || inp.isDroppedLine(log, string(message.Content)) {
			continue
		}

		metrics.BytesProcessed.Add(uint64(message.Bytes))

		// add "take_over" tag if `take_over` is set to true
		if inp.takeOver {
			_ = mapstr.AddTags(message.Fields, []string{"take_over"})
		}

		if err := p.Publish(message.ToEvent(), s); err != nil {
			metrics.ProcessingErrors.Inc()
=======
	if isGZIP {
		metrics.FilesGZIPOpened.Inc()
		metrics.HarvesterOpenGZIPFiles.Inc()
		metrics.HarvesterGZIPStarted.Inc()
		defer metrics.FilesGZIPClosed.Inc()
		defer metrics.HarvesterOpenGZIPFiles.Dec()
		defer metrics.HarvesterGZIPClosed.Inc()
	}

	var err error
	for ctx.Cancelation.Err() == nil {
		err = inp.readLineFromSource(r, log, metrics, isGZIP, &s, p)
		err, shouldContinue := inp.handleReadError(ctx, err, log, path, metrics, isGZIP)
		if !shouldContinue {
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
			return err
		}
	}

<<<<<<< HEAD
		metrics.EventsProcessed.Inc()
		metrics.ProcessingTime.Update(time.Since(message.Ts).Nanoseconds())
=======
	if inp.readUntilEOF.Enabled {
		eofCtx, cancel := context.WithTimeout(
			context.Background(), inp.readUntilEOF.Timeout)
		defer cancel()
		eofCancelCtx := ctxtool.WithCancelContext(eofCtx)
		// Set the underlying logFile into close-on-EOF mode and wake any
		// in-flight backoff so the next EOF terminates this loop.
		startReadUntilEOF(eofCancelCtx)

		log.Debugf("input closing, read_until_eof enabled, waiting EOF or %s timeout, whichever happens first",
			inp.readUntilEOF.Timeout)
	LOOP:
		for eofCancelCtx.Err() == nil {
			err = inp.readLineFromSource(r, log, metrics, isGZIP, &s, p)
			err, shouldContinue := inp.handleReadError(ctx, err, log, path, metrics, isGZIP)
			if errors.Is(err, io.EOF) {
				log.Debug("read_until_eof enabled, EOF reached. closing input")
				break LOOP
			}

			if !shouldContinue {
				return err
			}
		}
		if eofCancelCtx.Err() != nil {
			log.Infof("read_until_eof enabled, %s timeout reached. closing input", inp.readUntilEOF.Timeout)
		}
>>>>>>> 14ddacbbc (filebeat: add `read_until_eof` to filestream (#50324))
	}
	return nil
}

func (inp *filestream) readLineFromSource(r reader.Reader, log *logp.Logger, metrics *loginp.Metrics, isGZIP bool, s *state, p loginp.Publisher) error {
	message, err := r.Next()
	if err != nil {
		return err
	}

	// state offset increase. Mutated through *s so subsequent reads in
	// readFromSource see the accumulated offset
	s.Offset += int64(message.Bytes) + int64(message.Offset)

	flags, err := message.Fields.GetValue("log.flags")
	if err == nil {
		if flags, ok := flags.([]string); ok {
			if slices.Contains(flags, "truncated") { //nolint:typecheck,nolintlint // linter fails to infer generics
				metrics.MessagesTruncated.Add(1)
				if isGZIP {
					// Truncation shouldn't happen for GZIP files, but as
					// there it the overall metric for filestream, this case
					// is handled for completeness.
					metrics.MessagesGZIPTruncated.Add(1)
				}
			}
		}
	}

	metrics.MessagesRead.Inc()
	if isGZIP {
		metrics.MessagesGZIPRead.Inc()
	}
	if message.IsEmpty() || (inp.hasLineFilter && inp.isDroppedLine(log, message.Content)) {
		return nil
	}

	//nolint:gosec // message.Bytes is always positive
	metrics.BytesProcessed.Add(uint64(message.Bytes))
	if isGZIP {
		//nolint:gosec // message.Bytes is always positive, no risk of overflow here
		metrics.BytesGZIPProcessed.Add(uint64(message.Bytes))
	}

	// add "take_over" tag if `take_over` is set to true
	if inp.takeOver.Enabled {
		_ = mapstr.AddTags(message.Fields, []string{"take_over"})
	}

	if isGZIP {
		if err, ok := (message.Private).(error); ok && errors.Is(err, io.EOF) {
			s.EOF = true
		}
	}
	if err := p.Publish(message.ToEvent(), *s); err != nil {
		metrics.ProcessingErrors.Inc()
		if isGZIP {
			metrics.ProcessingGZIPErrors.Inc()
		}
		return err
	}

	metrics.EventsProcessed.Inc()
	metrics.ProcessingTime.Update(time.Since(message.Ts).Nanoseconds())
	if isGZIP {
		metrics.EventsGZIPProcessed.Inc()
		metrics.ProcessingGZIPTime.Update(time.Since(message.Ts).Nanoseconds())
	}

	return nil
}

func (inp *filestream) handleReadError(
	ctx input.Context,
	err error,
	log *logp.Logger,
	path string,
	metrics *loginp.Metrics,
	isGZIP bool) (error, bool) {
	if err == nil {
		return nil, true
	}

	if errors.Is(err, ErrFileTruncate) {
		log.Infof("File was truncated, nothing to read. Path='%s'", path)
	} else if errors.Is(err, ErrClosed) {
		// Enter the readUntilEOF drain only when the input itself is being
		// cancelled: returning (nil, true) here makes readFromSource's
		// outer loop re-check ctx.Cancelation and fall through to the
		// readUntilEOF block.
		//
		// For any other source of ErrClosed — close.reader.after_interval,
		// close.on_state_change.removed, close.on_state_change.renamed, or
		// an explicit Close — the input is not shutting down and we must
		// close normally.
		if inp.readUntilEOF.Enabled && ctx.Cancelation.Err() != nil {
			return nil, true
		}

		log.Debugf("Reader was closed. Closing. Path='%s'", path)
	} else if errors.Is(err, io.EOF) {
		log.Debugf("EOF has been reached. Closing. Path='%s'", path)
		if inp.deleterConfig.Enabled || inp.readUntilEOF.Enabled {
			return err, false
		}
	} else if errors.Is(err, ErrInactive) {
		log.Debugf("File is inactive. Closing. Path='%s'", path)
		return err, false
	} else {
		log.Errorf("Read line error: %v", err)
		metrics.ProcessingErrors.Inc()
		if isGZIP {
			metrics.ProcessingGZIPErrors.Inc()
		}
	}

	return nil, false
}

// isDroppedLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (inp *filestream) isDroppedLine(log *logp.Logger, line string) bool {
	if len(inp.readerConfig.IncludeLines) > 0 {
		if !matchAny(inp.readerConfig.IncludeLines, line) {
			if log.IsDebug() {
				log.Debug("Drop line as it does not match any of the include patterns %s", line)
			}
			return true
		}
	}
	if len(inp.readerConfig.ExcludeLines) > 0 {
		if matchAny(inp.readerConfig.ExcludeLines, line) {
			if log.IsDebug() {
				log.Debug("Drop line as it does match one of the exclude patterns%s", line)
			}
			return true
		}
	}

	return false
}

func matchAny(matchers []match.Matcher, text string) bool {
	for _, m := range matchers {
		if m.MatchString(text) {
			return true
		}
	}
	return false
}
