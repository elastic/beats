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
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"golang.org/x/text/transform"

	"github.com/elastic/go-concert/ctxtool"

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
)

const pluginName = "filestream"

type state struct {
	Offset int64 `json:"offset" struct:"offset"`
	EOF    bool  `json:"eof" struct:"eof"`
}

type fileMeta struct {
	Source         string `json:"source" struct:"source"`
	IdentifierName string `json:"identifier_name" struct:"identifier_name"`
}

// filestream is the input for reading from files which
// are actively written by other applications.
type filestream struct {
	readerConfig              readerConfig
	encodingFactory           encoding.EncodingFactory
	closerConfig              closerConfig
	deleterConfig             deleterConfig
	parsers                   parser.Config
	takeOver                  loginp.TakeOverConfig
	scannerCheckInterval      time.Duration
	compression               string
	includeFileOwnerName      bool
	includeFileOwnerGroupName bool

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

	c := defaultConfig()
	if err := cfg.Unpack(&c); err != nil {
		return nil, nil, err
	}

	// zero must also disable clean_inactive, see:
	// https://github.com/elastic/beats/issues/45601
	// for more details. At the same time we need to allow
	// users to keep the old behaviour.
	if !c.LegacyCleanInactive && c.CleanInactive == 0 {
		c.CleanInactive = -1
	}

	// log warning if deprecated params are set
	c.checkUnsupportedParams(log)

	c.TakeOver.LogWarnings(log)

	prospector, err := newProspector(c, log, src)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create prospector: %w", err)
	}

	encodingFactory, ok := encoding.FindEncoding(c.Reader.Encoding)
	if !ok || encodingFactory == nil {
		return nil, nil, fmt.Errorf("unknown encoding('%v')", c.Reader.Encoding)
	}

	filestream := &filestream{
		readerConfig:              c.Reader,
		encodingFactory:           encodingFactory,
		closerConfig:              c.Close,
		parsers:                   c.Reader.Parsers,
		takeOver:                  c.TakeOver,
		compression:               c.Compression,
		includeFileOwnerName:      c.IncludeFileOwnerName,
		includeFileOwnerGroupName: c.IncludeFileOwnerGroupName,
		deleterConfig:             c.Delete,
		waitGracePeriodFn:         waitGracePeriod,
		tickFn:                    time.Tick,
		removeFn:                  os.Remove,
		statFn:                    os.Stat,
	}

	// Read the scan interval from the prospector so we can use during the
	// grace period of the delete
	filestream.scannerCheckInterval = c.FileWatcher.Interval

	return prospector, filestream, nil
}

func (inp *filestream) Name() string { return pluginName }

func (inp *filestream) Test(src loginp.Source, ctx input.TestContext) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	reader, _, err := inp.open(ctx.Logger, ctx.Cancelation, fs, 0)
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
	metrics *loginp.Metrics,
) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	log := ctx.Logger.With("path", fs.newPath).With("state-id", src.Name())
	state := initState(log, cursor, fs)
	if state.EOF {
		// TODO: change it to debug once GZIP isn't experimental anymore.
		log.Infof("GZIP file already read to EOF, not reading it again, file name '%s'",
			fs.newPath)
		return nil
	}

	r, truncated, err := inp.open(log, ctx.Cancelation, fs, state.Offset)
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
	if fs.desc.GZIP {
		metrics.FilesGZIPActive.Inc()
		metrics.HarvesterGZIPRunning.Inc()
		defer metrics.FilesGZIPActive.Dec()
		defer metrics.HarvesterGZIPRunning.Dec()
	}

	_, streamCancel := ctxtool.WithFunc(ctx.Cancelation, func() {
		log.Debug("Closing reader of filestream")
		err := r.Close()
		if err != nil {
			log.Errorf("Error stopping filestream reader: %v", err)
		}
	})
	defer streamCancel()

	// The caller of Run already reports the error and filters out errors that
	// must not be reported, like 'context cancelled'.
	err = inp.readFromSource(
		ctx, log, r, fs.newPath, state, publisher, fs.desc.GZIP, metrics)
	if err != nil {
		// First handle actual errors
		if !errors.Is(err, io.EOF) && !errors.Is(err, ErrInactive) {
			return fmt.Errorf("error reading from source: %w", err)
		}

		if inp.deleterConfig.Enabled {
			if err := inp.deleteFile(ctx, log, cursor, fs.newPath); err != nil {
				return fmt.Errorf("cannot remove file '%s': %w", fs.newPath, err)
			}
		}
	}

	return nil
}

func (inp *filestream) deleteFile(
	ctx input.Context,
	logger *logp.Logger,
	cursor loginp.Cursor,
	path string,
) error {
	// We can only try deleting the file if all events have been published.
	// There are some cases when not all events have been published:
	//   - The output is a little behind the input
	//   - Filebeat is experiencing back pressure
	//   - The output is down
	// If not all events have been published, return so the harvester
	// can close. It will be recreated in the next scan.
	if !cursor.AllEventsPublished() {
		logger.Debugf(
			"not all events from '%s' have been published, "+
				"closing harvester",
			path)
		return nil
	}
	logger.Infof(
		"all events from '%s' have been published, waiting for %s grace period",
		path, inp.deleterConfig.GracePeriod.String())

	canRemove, err := inp.waitGracePeriodFn(
		ctx,
		logger,
		cursor,
		path,
		inp.deleterConfig.GracePeriod,
		inp.scannerCheckInterval,
		inp.statFn,
	)
	if err != nil {
		return err
	}

	if !canRemove {
		return nil
	}

	if err := inp.removeFn(path); err != nil {
		// The first try at removing the file failed,
		// retry with a constant backoff
		lastErr := err

		tickerChan := inp.tickFn(inp.deleterConfig.retryBackoff)

		retries := 0
		for retries < inp.deleterConfig.retries {
			logger.Errorf(
				"could not remove '%s', retrying in 2s. Error: %s",
				path,
				lastErr,
			)

			select {
			case <-ctx.Cancelation.Done():
				return ctx.Cancelation.Err()

			case <-tickerChan:
				retries++
				err := inp.removeFn(path)
				if err == nil {
					logger.Infof("'%s' removed", path)
					return nil
				}
				if errors.Is(err, os.ErrNotExist) {
					logger.Infof("'%s' was removed by an external process", path)
					return nil
				}

				lastErr = err
			}
		}

		return fmt.Errorf(
			"cannot remove '%s' after %d retries. Last error: %w",
			path,
			retries,
			lastErr)
	}

	logger.Infof("'%s' removed", path)
	return nil
}

// waitGracePeriod waits for the delete grace period while monitoring the file
// for any changes, if the file changes or any error is encountered, false
// is returned. True is only returned if the grace period expires with
// no error and no context cancellation.
func waitGracePeriod(
	ctx input.Context,
	logger *logp.Logger,
	cursor loginp.Cursor,
	path string,
	gracePeriod, checkInterval time.Duration,
	statFn func(string) (os.FileInfo, error),
) (bool, error) {
	// Check if file grows during the grace period
	// We know all events have been published because cursor.AllEventsPublished
	// returns is true (it is called by deleteFile), so we can get the offset
	// from the cursor and compare it with the file size.
	st := state{}
	if err := cursor.Unpack(&st); err != nil {
		return false, fmt.Errorf("cannot unpack cursor from '%s' to read offset: %w",
			path,
			err)
	}

	if gracePeriod > 0 {
		graceTimerChan := time.After(gracePeriod)
		checkIntervalTickerChan := time.Tick(checkInterval)
		// Wait for the grace period or for the context to be cancelled
	LOOP:
		for {
			select {
			case <-ctx.Cancelation.Done():
				return false, ctx.Cancelation.Err()
			case <-checkIntervalTickerChan:
				canDelete, err := canDeleteFile(logger, path, st.Offset, statFn)
				if err != nil && !canDelete {
					return false, err
				}
			case <-graceTimerChan:
				break LOOP
			}
		}
	}

	return canDeleteFile(logger, path, st.Offset, statFn)
}

// canDeleteFile returns true if the file size has not changed and
// the file can be removed. If the file does not exist, false is returned.
// If there is an error reading the file size, false and an error are returned.
func canDeleteFile(
	logger *logp.Logger,
	path string,
	expectedSize int64,
	statFn func(string) (os.FileInfo, error),
) (bool, error) {

	stat, err := statFn(path)
	if err != nil {
		// If the file does not exist any more, return false
		// (do not delete) and no error.
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		// Return the error and cause the harvester to close
		return false, fmt.Errorf("cannot stat '%s': %w", path, err)
	}

	// If the file has been written to, close the harvester so the filewatcher
	// can start a new one
	if stat.Size() != expectedSize {
		logger.Debugf("'%s' was updated, won't remove. Closing harvester", path)
		return false, nil
	}

	return true, nil
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
) (reader.Reader, bool, error) {

	f, encoding, truncated, err := inp.openFile(log, fs.newPath, offset)
	if err != nil {
		return nil, truncated, err
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
	logReader, err := newFileReader(log, canceler, f, inp.readerConfig, closerCfg)
	if err != nil {
		return nil, truncated, err
	}

	dbgReader, err := debug.AppendReaders(logReader, log)
	if err != nil {
		return nil, truncated, err
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
		return nil, truncated, err
	}

	r = readfile.NewStripNewline(r, inp.readerConfig.LineTerminator)

	r = readfile.NewFilemeta(r, fs.newPath, fs.desc.Info, inp.includeFileOwnerName, inp.includeFileOwnerGroupName, fs.desc.Fingerprint, offset)

	r = inp.parsers.Create(r, log)

	r = readfile.NewLimitReader(r, inp.readerConfig.MaxBytes)

	if f.IsGZIP() {
		r = NewEOFLookaheadReader(r, io.EOF)
	}

	ok = true // no need to close the file
	return r, truncated, nil
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned.
//
// openFile will also detect and handle file truncation. If a file is truncated
// then the 3rd return value is true.
func (inp *filestream) openFile(
	log *logp.Logger,
	path string,
	offset int64,
) (File, encoding.Encoding, bool, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed to stat source file %s: %w", path, err)
	}

	// it must be checked if the file is not a named pipe before we try to open it
	// if it is a named pipe os.OpenFile fails, so there is no need to try opening it.
	if fi.Mode()&os.ModeNamedPipe != 0 {
		return nil, nil, false, fmt.Errorf("failed to open file %s, named pipes are not supported", fi.Name())
	}

	rawFile, err := file.ReadOpen(path)
	if err != nil {
		return nil, nil, false, fmt.Errorf("failed opening %s: %w", path, err)
	}

	f, err := inp.newFile(rawFile)
	if err != nil {
		return nil, nil, false,
			fmt.Errorf("failed to create a File from a os.File: %w", err)
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
	// GZIP files are considered static, they're not supposed to change or be
	// truncated. Also:
	//  - as the offset is tracked on the decompressed data, it's
	// expected to see offset > fi.Size()
	//  - it should not start reading GZIP files from the beginning if it
	//  already started ingesting the file.
	// The only situation a GZIP file should change is if it's still been
	// written to disk when filebeat picks it up. It should only grow, not
	// shrink.
	// Therefore, only check truncation for plain files.
	if !f.IsGZIP() && fi.Size() < offset {
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

	enc, err := inp.encodingFactory(f)
	if err != nil {
		if errors.Is(err, transform.ErrShortSrc) {
			return nil, nil, truncated, fmt.Errorf("initialising encoding for '%v' failed due to file being too short", f)
		}
		return nil, nil, truncated, fmt.Errorf("initialising encoding for '%v' failed: %w", f, err)
	}

	ok = true // no need to close the file
	return f, enc, truncated, nil
}

// newFile wraps the given os.File into an appropriate File interface implementation.
//
// The behavior depends on the compression setting:
//   - "" (none): returns a plain file reader (plainFile)
//   - "gzip": always creates a gzipSeekerReader (errors if file is not gzip)
//   - "auto": auto-detects gzip files; returns gzipSeekerReader for gzip files,
//     plainFile otherwise
//
// It returns an error if any happens.
func (inp *filestream) newFile(rawFile *os.File) (File, error) {
	switch inp.compression {
	case CompressionNone:
		return newPlainFile(rawFile), nil

	case CompressionGZIP:
		f, err := newGzipSeekerReader(rawFile, inp.readerConfig.BufferSize)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create gzip reader for %s: %w", rawFile.Name(), err)
		}
		return f, nil

	case CompressionAuto:
		isGZIP, err := IsGZIP(rawFile)
		if err != nil {
			return nil, fmt.Errorf(
				"gzip detection error on %s: %w", rawFile.Name(), err)
		}

		if !isGZIP {
			return newPlainFile(rawFile), nil
		}

		f, err := newGzipSeekerReader(rawFile, inp.readerConfig.BufferSize)
		if err != nil {
			return nil, fmt.Errorf(
				"failed to create gzip reader for %s: %w", rawFile.Name(), err)
		}
		return f, nil

	default:
		// This should not happen as validation catches invalid values
		return nil, fmt.Errorf("invalid compression mode: %q", inp.compression)
	}
}

func checkFileBeforeOpening(fi os.FileInfo) error {
	if !fi.Mode().IsRegular() {
		return fmt.Errorf("tried to open non regular file: %q %s", fi.Mode(), fi.Name())
	}

	return nil
}

func (inp *filestream) initFileOffset(file File, offset int64) error {
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
	isGZIP bool,
	metrics *loginp.Metrics) error {

	metrics.FilesOpened.Inc()
	metrics.HarvesterOpenFiles.Inc()
	metrics.HarvesterStarted.Inc()
	defer metrics.FilesClosed.Inc()
	defer metrics.HarvesterOpenFiles.Dec()
	defer metrics.HarvesterClosed.Inc()

	if isGZIP {
		metrics.FilesGZIPOpened.Inc()
		metrics.HarvesterOpenGZIPFiles.Inc()
		metrics.HarvesterGZIPStarted.Inc()
		defer metrics.FilesGZIPClosed.Inc()
		defer metrics.HarvesterOpenGZIPFiles.Dec()
		defer metrics.HarvesterGZIPClosed.Inc()
	}

	for ctx.Cancelation.Err() == nil {
		// next line - r needs to be reading from a gzipped file
		message, err := r.Next()
		if err != nil {
			if errors.Is(err, ErrFileTruncate) {
				log.Infof("File was truncated, nothing to read. Path='%s'", path)
			} else if errors.Is(err, ErrClosed) {
				log.Debugf("Reader was closed. Closing. Path='%s'", path)
			} else if errors.Is(err, io.EOF) {
				log.Debugf("EOF has been reached. Closing. Path='%s'", path)
				if inp.deleterConfig.Enabled {
					return err
				}
			} else if errors.Is(err, ErrInactive) {
				log.Debugf("File is inactive. Closing. Path='%s'", path)
				return err
			} else {
				log.Errorf("Read line error: %v", err)
				metrics.ProcessingErrors.Inc()
				if isGZIP {
					metrics.ProcessingGZIPErrors.Inc()
				}
			}

			return nil
		}

		// sate offset increase
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
		if message.IsEmpty() || inp.isDroppedLine(log, string(message.Content)) {
			continue
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
		if err := p.Publish(message.ToEvent(), s); err != nil {
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
	}
	return nil
}

// isDroppedLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (inp *filestream) isDroppedLine(log *logp.Logger, line string) bool {
	if len(inp.readerConfig.IncludeLines) > 0 {
		if !matchAny(inp.readerConfig.IncludeLines, line) {
			log.Debug("Drop line as it does not match any of the include patterns %s", line)
			return true
		}
	}
	if len(inp.readerConfig.ExcludeLines) > 0 {
		if matchAny(inp.readerConfig.ExcludeLines, line) {
			log.Debug("Drop line as it does match one of the exclude patterns%s", line)
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
