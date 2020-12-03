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
	"fmt"
	"os"

	"golang.org/x/text/transform"

	"github.com/elastic/go-concert/ctxtool"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/match"
	"github.com/elastic/beats/v7/libbeat/feature"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/debug"
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
	bufferSize      int
	tailFile        bool // TODO
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding
	lineTerminator  readfile.LineTerminator
	excludeLines    []match.Matcher
	includeLines    []match.Matcher
	maxBytes        int
	closerConfig    closerConfig
}

// Plugin creates a new filestream input plugin for creating a stateful input.
func Plugin(log *logp.Logger, store loginp.StateStore) input.Plugin {
	return input.Plugin{
		Name:       pluginName,
		Stability:  feature.Experimental,
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

	filewatcher, err := newFileWatcher(config.Paths, config.FileWatcher)
	if err != nil {
		return nil, nil, fmt.Errorf("error while creating filewatcher %v", err)
	}

	identifier, err := newFileIdentifier(config.FileIdentity)
	if err != nil {
		return nil, nil, fmt.Errorf("error while creating file identifier: %v", err)
	}

	encodingFactory, ok := encoding.FindEncoding(config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, nil, fmt.Errorf("unknown encoding('%v')", config.Encoding)
	}

	prospector := &fileProspector{
		filewatcher:       filewatcher,
		identifier:        identifier,
		ignoreOlder:       config.IgnoreOlder,
		cleanRemoved:      config.CleanRemoved,
		stateChangeCloser: config.Close.OnStateChange,
	}

	filestream := &filestream{
		readerConfig:    config.readerConfig,
		bufferSize:      config.BufferSize,
		encodingFactory: encodingFactory,
		lineTerminator:  config.LineTerminator,
		excludeLines:    config.ExcludeLines,
		includeLines:    config.IncludeLines,
		maxBytes:        config.MaxBytes,
		closerConfig:    config.Close,
	}

	return prospector, filestream, nil
}

func (inp *filestream) Name() string { return pluginName }

func (inp *filestream) Test(src loginp.Source, ctx input.TestContext) error {
	fs, ok := src.(fileSource)
	if !ok {
		return fmt.Errorf("not file source")
	}

	reader, err := inp.open(ctx.Logger, ctx.Cancelation, fs.newPath, 0)
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

	r, err := inp.open(log, ctx.Cancelation, fs.newPath, state.Offset)
	if err != nil {
		log.Errorf("File could not be opened for reading: %v", err)
		return err
	}

	_, streamCancel := ctxtool.WithFunc(ctxtool.FromCanceller(ctx.Cancelation), func() {
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
	if c.IsNew() {
		return state
	}

	err := c.Unpack(&state)
	if err != nil {
		log.Error("Cannot serialize cursor data into file state: %+v", err)
	}

	return state
}

func (inp *filestream) open(log *logp.Logger, canceler input.Canceler, path string, offset int64) (reader.Reader, error) {
	f, err := inp.openFile(path, offset)
	if err != nil {
		return nil, err
	}

	log.Debug("newLogFileReader with config.MaxBytes:", inp.maxBytes)

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	logReader, err := newFileReader(log, canceler, f, inp.readerConfig, inp.closerConfig)
	if err != nil {
		return nil, err
	}

	dbgReader, err := debug.AppendReaders(logReader)
	if err != nil {
		f.Close()
		return nil, err
	}

	// Configure MaxBytes limit for EncodeReader as multiplied by 4
	// for the worst case scenario where incoming UTF32 charchers are decoded to the single byte UTF-8 characters.
	// This limit serves primarily to avoid memory bload or potential OOM with expectedly long lines in the file.
	// The further size limiting is performed by LimitReader at the end of the readers pipeline as needed.
	encReaderMaxBytes := inp.maxBytes * 4

	var r reader.Reader
	r, err = readfile.NewEncodeReader(dbgReader, readfile.Config{
		Codec:      inp.encoding,
		BufferSize: inp.bufferSize,
		Terminator: inp.lineTerminator,
		MaxBytes:   encReaderMaxBytes,
	})
	if err != nil {
		f.Close()
		return nil, err
	}

	r = readfile.NewStripNewline(r, inp.lineTerminator)
	r = readfile.NewLimitReader(r, inp.maxBytes)

	return r, nil
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (inp *filestream) openFile(path string, offset int64) (*os.File, error) {
	err := inp.checkFileBeforeOpening(path)
	if err != nil {
		return nil, err
	}

	f, err := os.OpenFile(path, os.O_RDONLY, os.FileMode(0))
	if err != nil {
		return nil, fmt.Errorf("failed opening %s: %s", path, err)
	}

	err = inp.initFileOffset(f, offset)
	if err != nil {
		f.Close()
		return nil, err
	}

	inp.encoding, err = inp.encodingFactory(f)
	if err != nil {
		f.Close()
		if err == transform.ErrShortSrc {
			return nil, fmt.Errorf("initialising encoding for '%v' failed due to file being too short", f)
		}
		return nil, fmt.Errorf("initialising encoding for '%v' failed: %v", f, err)
	}

	return f, nil
}

func (inp *filestream) checkFileBeforeOpening(path string) error {
	fi, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("failed to stat source file %s: %v", path, err)
	}

	if !fi.Mode().IsRegular() {
		return fmt.Errorf("tried to open non regular file: %q %s", fi.Mode(), fi.Name())
	}

	if fi.Mode()&os.ModeNamedPipe != 0 {
		return fmt.Errorf("failed to open file %s, named pipes are not supported", path)
	}

	return nil
}

func (inp *filestream) initFileOffset(file *os.File, offset int64) error {
	if offset > 0 {
		_, err := file.Seek(offset, os.SEEK_SET)
		return err
	}

	// get offset from file in case of encoding factory was required to read some data.
	_, err := file.Seek(0, os.SEEK_CUR)
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
			switch err {
			case ErrFileTruncate:
				log.Info("File was truncated. Begin reading file from offset 0.")
				s.Offset = 0
			case ErrClosed:
				log.Info("Reader was closed. Closing.")
			case reader.ErrLineUnparsable:
				log.Info("Skipping unparsable line in file.")
				continue
			default:
				log.Errorf("Read line error: %v", err)
			}
			return nil
		}

		if message.IsEmpty() || inp.isDroppedLine(log, string(message.Content)) {
			continue
		}

		event := inp.eventFromMessage(message, path)
		s.Offset += int64(message.Bytes)

		if err := p.Publish(event, s); err != nil {
			return err
		}
	}
	return nil
}

// isDroppedLine decides if the line is exported or not based on
// the include_lines and exclude_lines options.
func (inp *filestream) isDroppedLine(log *logp.Logger, line string) bool {
	if len(inp.includeLines) > 0 {
		if !matchAny(inp.includeLines, line) {
			log.Debug("Drop line as it does not match any of the include patterns %s", line)
			return true
		}
	}
	if len(inp.excludeLines) > 0 {
		if matchAny(inp.excludeLines, line) {
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

func (inp *filestream) eventFromMessage(m reader.Message, path string) beat.Event {
	fields := common.MapStr{
		"log": common.MapStr{
			"offset": m.Bytes, // Offset here is the offset before the starting char.
			"file": common.MapStr{
				"path": path,
			},
		},
	}
	fields.DeepUpdate(m.Fields)

	if len(m.Content) > 0 {
		if fields == nil {
			fields = common.MapStr{}
		}
		fields["message"] = string(m.Content)
	}

	return beat.Event{
		Timestamp: m.Ts,
		Fields:    fields,
	}
}
