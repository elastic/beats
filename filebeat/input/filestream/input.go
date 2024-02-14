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

	"golang.org/x/text/transform"

	"github.com/elastic/go-concert/ctxtool"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/cleanup"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/debug"
	"github.com/elastic/beats/v7/libbeat/reader/parser"
	"github.com/elastic/beats/v7/libbeat/reader/readfile"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
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
	readerConfig    readerConfig
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	closerConfig    closerConfig
	parsers         parser.Config
}

// Plugin creates a new filestream input plugin for creating a stateful input.
func Plugin(log *logp.Logger, store loginp.StateStore) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Stable,
		Deprecated: false,
		Info:       "filestream input",
		Doc:        "The filestream input collects logs from the local filestream service",
		Manager: &loginp.InputManager{
			Logger:     log,
			StateStore: store,
			Type:       pluginName,
			Configure:  configure,
		},
	}
}

func configure(cfg *common.Config) (loginp.Prospector, loginp.Harvester, error) {
	config := defaultConfig()
	if err := cfg.Unpack(&config); err != nil {
		return nil, nil, err
	}

	prospector, err := newProspector(config)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create prospector: %w", err)
	}

	encodingFactory, ok := encoding.FindEncoding(config.Reader.Encoding)
	if !ok || encodingFactory == nil {
		return nil, nil, fmt.Errorf("unknown encoding('%v')", config.Reader.Encoding)
	}

	filestream := &filestream{
		readerConfig:    config.Reader,
		encodingFactory: encodingFactory,
		closerConfig:    config.Close,
		parsers:         config.Reader.Parsers,
	}

	return prospector, filestream, nil
}

func (inp *filestream) Name() string { return pluginName }

func (inp *filestream) Test(src loginp.Source, ctx input.TestContext) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	reader, err := inp.open(ctx.Logger, ctx.Cancelation, fs, 0)
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
) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	log := ctx.Logger.With("path", fs.newPath).With("state-id", src.Name())
	state := initState(log, cursor, fs)

	r, err := inp.open(log, ctx.Cancelation, fs, state.Offset)
	if err != nil {
		log.Errorf("File could not be opened for reading: %v", err)
		return err
	}

	_, streamCancel := ctxtool.WithFunc(ctx.Cancelation, func() {
		log.Debug("Closing reader of filestream")
		err := r.Close()
		if err != nil {
			log.Errorf("Error stopping filestream reader %v", err)
		}
	})
	defer streamCancel()

	return inp.readFromSource(ctx, log, r, fs.newPath, state, publisher)
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

func (inp *filestream) open(log *logp.Logger, canceler input.Canceler, fs fileSource, offset int64) (reader.Reader, error) {
	f, err := inp.openFile(log, fs.newPath, offset)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	dbgReader, err := debug.AppendReaders(logReader)
	if err != nil {
		return nil, err
	}

	// Configure MaxBytes limit for EncodeReader as multiplied by 4
	// for the worst case scenario where incoming UTF32 charchers are decoded to the single byte UTF-8 characters.
	// This limit serves primarily to avoid memory bload or potential OOM with expectedly long lines in the file.
	// The further size limiting is performed by LimitReader at the end of the readers pipeline as needed.
	encReaderMaxBytes := inp.readerConfig.MaxBytes * 4

	var r reader.Reader
	r, err = readfile.NewEncodeReader(dbgReader, readfile.Config{
		Codec:      inp.encoding,
		BufferSize: inp.readerConfig.BufferSize,
		Terminator: inp.readerConfig.LineTerminator,
		MaxBytes:   encReaderMaxBytes,
	})
	if err != nil {
		return nil, err
	}

	r = readfile.NewStripNewline(r, inp.readerConfig.LineTerminator)

	r = readfile.NewFilemeta(r, fs.newPath, offset)

	r = inp.parsers.Create(r)

	r = readfile.NewLimitReader(r, inp.readerConfig.MaxBytes)

	ok = true // no need to close the file
	return r, nil
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (inp *filestream) openFile(log *logp.Logger, path string, offset int64) (*os.File, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source file %s: %w", path, err)
	}

	// it must be checked if the file is not a named pipe before we try to open it
	// if it is a named pipe os.OpenFile fails, so there is no need to try opening it.
	if fi.Mode()&os.ModeNamedPipe != 0 {
		return nil, fmt.Errorf("failed to open file %s, named pipes are not supported", fi.Name())
	}

	f, err := file.ReadOpen(path)
	if err != nil {
		return nil, fmt.Errorf("failed opening %s: %w", path, err)
	}
	ok := false
	defer cleanup.IfNot(&ok, cleanup.IgnoreError(f.Close))

	fi, err = f.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat source file %s: %w", path, err)
	}

	err = checkFileBeforeOpening(fi)
	if err != nil {
		return nil, err
	}

	if fi.Size() < offset {
		log.Infof("File was truncated. Reading file from offset 0. Path=%s", path)
		offset = 0
	}
	err = inp.initFileOffset(f, offset)
	if err != nil {
		return nil, err
	}

	inp.encoding, err = inp.encodingFactory(f)
	if err != nil {
		if errors.Is(err, transform.ErrShortSrc) {
			return nil, fmt.Errorf("initialising encoding for '%v' failed due to file being too short", f)
		}
		return nil, fmt.Errorf("initialising encoding for '%v' failed: %w", f, err)
	}

	ok = true // no need to close the file
	return f, nil
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
) error {
	for ctx.Cancelation.Err() == nil {
		message, err := r.Next()
		if err != nil {
			if errors.Is(err, ErrFileTruncate) {
				log.Infof("File was truncated. Begin reading file from offset 0. Path=%s", path)
			} else if errors.Is(err, ErrClosed) {
				log.Info("Reader was closed. Closing.")
			} else if errors.Is(err, io.EOF) {
				log.Debugf("EOF has been reached. Closing.")
			} else {
				log.Errorf("Read line error: %v", err)
			}
			return nil
		}

		s.Offset += int64(message.Bytes)

		if message.IsEmpty() || inp.isDroppedLine(log, string(message.Content)) {
			continue
		}

		if err := p.Publish(message.ToEvent(), s); err != nil {
			return err
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
