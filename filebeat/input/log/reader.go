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

// Package log harvests different inputs for new information. Currently
// two harvester types exist:
//
//   * log
//   * stdin
//
//  The log harvester reads a file line by line. In case the end of a file is found
//  with an incomplete line, the line pointer stays at the beginning of the incomplete
//  line. As soon as the line is completed, it is read and returned.
//
//  The stdin harvesters reads data from stdin.
package log

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/elastic/beats/filebeat/harvester"
	"github.com/elastic/beats/filebeat/input/file"
	file_helper "github.com/elastic/beats/libbeat/common/file"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/reader"
	"github.com/elastic/beats/libbeat/reader/debug"
	"github.com/elastic/beats/libbeat/reader/multiline"
	"github.com/elastic/beats/libbeat/reader/readfile"
	"github.com/elastic/beats/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/libbeat/reader/readjson"
	"github.com/gofrs/uuid"
	"github.com/mitchellh/hashstructure"
	"golang.org/x/text/transform"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	fileReaderManager = newFileReaderManager()

	ErrHarvesterExists  = errors.New("harvester was exists")
	ErrHarvesterDone    = errors.New("harvester was done")
	ErrContextNotExists = errors.New("harvester context dose not exists")
)

//ReuseMessage
type ReuseMessage struct {
	message reader.Message
	error   error
}

// ReuseHarvester: 对Harvester模块暴露的模块，它通过FileReaderManager获取相应的FileReader
// FileReaderManager根据 采集目标的类型、配置及采集进度，确认是否复用同一个FileReader
// 一个FileReader会对应多个ReuseReader, 并根据采集进度判断是否复用同一个FD
type ReuseHarvester struct {
	HarvesterID uuid.UUID
	Config      config
	State       file.State
	done        chan struct{}
	closeOnce   sync.Once
	fileReader  *FileHarvester
	message     chan ReuseMessage
}

// NewReuseHarvester creates a new reader by harvester
func NewReuseHarvester(
	harvesterID uuid.UUID,
	config config,
	state file.State,
) (*ReuseHarvester, error) {
	r := &ReuseHarvester{
		HarvesterID: harvesterID,
		done:        make(chan struct{}),
		Config:      config,
		State:       state,
		message:     make(chan ReuseMessage),
	}
	var err error
	r.fileReader, err = fileReaderManager.GetFileReader(r)
	if err != nil {
		return nil, err
	}
	//add forwarder
	err = r.fileReader.AddForwarder(r)
	if err != nil {
		return nil, err
	}
	return r, nil
}

//Next: 按行读取文件内容，并根据harvester offset返回
func (r *ReuseHarvester) Next() (reader.Message, error) {
	select {
	case <-r.done:
		return reader.Message{}, ErrHarvesterDone
	case <-r.fileReader.done:
		return reader.Message{}, ErrHarvesterDone
	case msg := <-r.message:
		return msg.message, msg.error
	}
}

//OnMessage:
func (r *ReuseHarvester) OnMessage(message ReuseMessage) error {
	select {
	case <-r.done:
		return ErrHarvesterDone
	case <-r.fileReader.done:
		return ErrHarvesterDone
	case r.message <- message:
		return nil
	}
}

//Stop: 停止harvester
func (r *ReuseHarvester) Stop() {
	r.closeOnce.Do(func() {
		close(r.done)
	})
}

//HasState
func (r *ReuseHarvester) HasState() bool {
	return r.fileReader.HasState()
}

//GetState
func (r *ReuseHarvester) GetState() file.State {
	return r.State
}

// FileReaderManager
type FileReaderManager struct {
	fileReaders     map[string][]*FileHarvester
	fileReadersLock sync.Mutex
}

func newFileReaderManager() *FileReaderManager {
	m := &FileReaderManager{
		fileReaders: make(map[string][]*FileHarvester),
	}
	return m
}

// GetFileReader:
func (m *FileReaderManager) GetFileReader(reuseReader *ReuseHarvester) (*FileHarvester, error) {
	id := m.getFileReaderHash(reuseReader)
	var fileReaders []*FileHarvester
	var ok bool

	// reuse reader
	m.fileReadersLock.Lock()
	defer func() {
		m.cleanup()
		m.fileReadersLock.Unlock()
	}()

	if fileReaders, ok = m.fileReaders[id]; ok {
		for _, fileReader := range fileReaders {
			select {
			case <-fileReader.done:
				continue
			default:
				if fileReader.state.Offset-reuseReader.State.Offset < reuseReader.Config.ReuseMaxBytes {
					logp.Debug("harvester reuse file reader, id: %s", id)
					return fileReader, nil
				}
			}
		}
		id = fmt.Sprintf("%s%s", reuseReader.State.ID(), reuseReader.HarvesterID.String())
	}

	// create new fileReader
	logp.Debug("harvester use a new file reader, id: %s", id)
	fileReader, err := newFileHarvester(reuseReader)
	if err != nil {
		return nil, err
	}
	m.fileReaders[id] = append(m.fileReaders[id], fileReader)
	return fileReader, nil
}

func (m *FileReaderManager) getFileReaderHash(reuseReader *ReuseHarvester) string {
	if !reuseReader.Config.ReuseHarvester {
		return fmt.Sprintf("%s%s", reuseReader.State.ID(), reuseReader.HarvesterID.String())
	}
	plaintext := fmt.Sprintf("%s%s%s%d%v",
		reuseReader.Config.Type,
		reuseReader.State.ID(),
		reuseReader.Config.Encoding,
		reuseReader.Config.MaxBytes,
		reuseReader.Config.Multiline)
	hashValue, _ := hashstructure.Hash(plaintext, nil)
	return strconv.FormatUint(hashValue, 10)
}

// cleanup: 清理已关闭的reader
func (m *FileReaderManager) cleanup() {
	for id, fileReaders := range m.fileReaders {
		leftFileReader := make([]*FileHarvester, 0)
		length := len(fileReaders)
		for i := 0; i < length; i++ {
			select {
			case <-fileReaders[i].done:
				continue
			default:
				leftFileReader = append(leftFileReader, fileReaders[i])
			}
		}
		fileReaders = leftFileReader
		if len(fileReaders) == 0 {
			delete(m.fileReaders, id)
			logp.Info("ReuseHarvester has release the FileHarvester, id: %s", id)
			continue
		}
		m.fileReaders[id] = fileReaders
	}
}

//FileHarvester:
type FileHarvester struct {
	config config
	state  file.State

	runOnce sync.Once

	done      chan struct{}
	closeOnce sync.Once

	// file reader pipeline
	source          harvester.Source // the source being watched
	log             *Log
	reader          reader.Reader
	encodingFactory encoding.EncodingFactory
	encoding        encoding.Encoding

	readerDone sync.WaitGroup

	//harvester
	forwarders     map[uuid.UUID]*ReuseHarvester
	forwardersLock sync.Mutex
	forwarder      chan *ReuseHarvester
}

//newFileHarvester: get file harvester
func newFileHarvester(reuseReader *ReuseHarvester) (*FileHarvester, error) {
	r := &FileHarvester{
		config:     reuseReader.Config,
		state:      reuseReader.State,
		done:       make(chan struct{}),
		forwarders: make(map[uuid.UUID]*ReuseHarvester),
		forwarder:  make(chan *ReuseHarvester),
	}
	encodingFactory, ok := encoding.FindEncoding(r.config.Encoding)
	if !ok || encodingFactory == nil {
		return nil, fmt.Errorf("unknown encoding('%v')", r.config.Encoding)
	}
	r.encodingFactory = encodingFactory

	err := r.Setup()
	if err != nil {
		r.Close()
		return nil, err
	}
	return r, nil
}

//addForwarder:
func (h *FileHarvester) AddForwarder(reuseReader *ReuseHarvester) error {
	h.forwardersLock.Lock()
	defer h.forwardersLock.Unlock()

	// Add Max close inactive
	if reuseReader.Config.CloseInactive > 0 && reuseReader.Config.CloseInactive > h.config.CloseInactive {
		h.config.CloseInactive = reuseReader.Config.CloseInactive
	}
	// Add ttl if clean_inactive is set
	if h.config.CleanInactive > 0 && h.config.CleanInactive > h.state.TTL {
		h.state.TTL = h.config.CleanInactive
	}

	//add forwarder
	go func() {
		select {
		case <-h.done:
			logp.Err("add forwarder failed, because FileHarvester is quit")
			return
		case h.forwarder <- reuseReader:
			return
		}
	}()

	// start to read file
	h.runOnce.Do(func() {
		go h.Run()
	})

	return nil
}

//HasReuseReader
func (h *FileHarvester) HasReuseReader() bool {
	return len(h.forwarders) > 0
}

//HasState
func (h *FileHarvester) HasState() bool {
	return h.source.HasState()
}

//Run: 从最小的Offset读取一行，并发送给所有匹配的reader
func (h *FileHarvester) Run() {
	defer func() {
	L:
		for {
			select {
			case rr := <-h.forwarder:
				logp.Info("forwarder(%s) cannot join. now exit. file:%s", rr.HarvesterID, h.state.Source)
				continue
			default:
				break L
			}
		}
		h.closeFile()
		h.readerDone.Wait()
		h.Close()
	}()

	isEmptyForwarderTimes := 0
	newForwarders := make([]*ReuseHarvester, 0)
	tick := time.NewTicker(3 * time.Second)
	defer tick.Stop()
	for {
		logp.Info("current len of forwarder is %d, file:%s", len(h.forwarders), h.state.Source)
		select {
		case <-h.done:
			return
		case reuseReader := <-h.forwarder:
			logp.Info("new forward join. ID(%s), file:%s", reuseReader.HarvesterID, h.state.Source)
			newForwarders = append(newForwarders, reuseReader)
		case <-tick.C:
			if len(newForwarders) > 0 {
				logp.Info("found new forward join, reload file:%s", h.state.Source)
				for _, reuseReader := range newForwarders {
					h.forwarders[reuseReader.HarvesterID] = reuseReader
				}
				newForwarders = make([]*ReuseHarvester, 0)

				offset, err := h.reloadFileOffset()
				if err != nil {
					logp.Err("reload file offset err: %v, file:%s", err, h.state.Source)
					return
				}
				h.state.Offset = offset
				logp.Info("reload file offset to (%d) success. file:%s", offset, h.state.Source)

				// until reader close, only one reader can running
				h.readerDone.Wait()

				// read file
				h.readerDone.Add(1)
				go h.loopRead()
			} else {
				for _, reuseReader := range h.forwarders {
					select {
					case <-reuseReader.done:
						logp.Info("forwarder is done, delete forwarder(%s)", reuseReader.HarvesterID)
						delete(h.forwarders, reuseReader.HarvesterID)
					default:
					}
				}
			}

			if len(h.forwarders) > 0 {
				isEmptyForwarderTimes = 0
			} else {
				isEmptyForwarderTimes++
				if isEmptyForwarderTimes >= 3 {
					logp.Info("forwarder is empty reach 3 times, stop fileharvester")
					return
				}
			}
		}
	}
}

// loopRead: loop read file, then forward to receive
func (h *FileHarvester) loopRead() {
	defer func() {
		h.readerDone.Done()
		logp.Info("loop Read quit. because file(%s) is close.", h.state.Source)
	}()

	for {
		select {
		case <-h.done:
			return
		default:
			message, err := h.reader.Next()
			if err != nil {
				logp.Info("read message error: %v, file:%s", err, h.state.Source)

				// 文件被关闭异常，不需要转发到外层。 在调用Close()后会引发，属于内部错误
				if pathErr, ok := err.(*os.PathError); ok {
					if pathErr.Err == os.ErrClosed {
						return
					}
				}

				h.forward(message, err)
				// 读取文件异常，关闭整个reader
				h.Close()
				return
			}

			// Strip UTF-8 BOM if beginning of file
			// As all BOMS are converted to UTF-8 it is enough to only remove this one
			if h.state.Offset == 0 {
				message.Content = bytes.Trim(message.Content, "\xef\xbb\xbf")
			}

			//Step 3: 转发消息，并添加offset
			h.forward(message, err)
			h.state.Offset += int64(message.Bytes)
			logp.Info("after read, Offset is:%d, file:%s", h.state.Offset, h.state.Source)
		}
	}
}

func (h *FileHarvester) forward(message reader.Message, err error) {
	reuseMsg := ReuseMessage{
		message: message,
		error:   err,
	}

	for _, reuseReader := range h.forwarders {
		select {
		case <-h.done:
			return
		default:
		}
		//有异常或超过原进度才发送
		if err != nil || h.state.Offset >= reuseReader.State.Offset {
			err := reuseReader.OnMessage(reuseMsg)
			if err != nil {
				switch err {
				case ErrHarvesterDone:
					logp.Info("log forward done: %v", err)
				default:
					logp.Err("log forward err: %v", err)
				}
				delete(h.forwarders, reuseReader.HarvesterID)
				continue
			}
			//更新采集任务进度
			reuseReader.State.Offset += int64(reuseMsg.message.Bytes)
		}
	}
}

// open does open the file given under h.Path and assigns the file handler to h.log
func (h *FileHarvester) open() error {
	switch h.config.Type {
	case harvester.StdinType:
		return h.openStdin()
	case harvester.LogType:
		return h.openFile()
	case harvester.DockerType:
		return h.openFile()
	default:
		return fmt.Errorf("invalid harvester type: %+v", h.config)
	}
}

func (h *FileHarvester) Close() {
	h.closeOnce.Do(func() {
		close(h.done)
	})
}

//Setup: 打开文件FD，首次执行会直接转到第一个state.offset
func (h *FileHarvester) Setup() error {
	err := h.open()
	if err != nil {
		return fmt.Errorf("harvester setup failed. Unexpected file opening error: %s", err)
	}

	h.reader, err = h.newLogFileReader()
	if err != nil {
		h.closeFile()
		return fmt.Errorf("harvester setup failed. Unexpected encoding line reader error: %s", err)
	}

	return nil
}

//Close: 关闭FD
func (h *FileHarvester) closeFile() {
	if h.source != nil {
		logp.Info("file harvester is close, file:%s", h.state.Source)
		err := h.source.Close()
		if err != nil {
			logp.Err("harvester reuse reader close failed. Unexpected error: %s", err)
		}
	}
}

// openFile opens a file and checks for the encoding. In case the encoding cannot be detected
// or the file cannot be opened because for example of failing read permissions, an error
// is returned and the harvester is closed. The file will be picked up again the next time
// the file system is scanned
func (h *FileHarvester) openFile() error {
	f, err := file_helper.ReadOpen(h.state.Source)
	if err != nil {
		return fmt.Errorf("failed opening %s: %s", h.state.Source, err)
	}

	harvesterOpenFiles.Add(1)

	// Makes sure file handler is also closed on errors
	err = h.validateFile(f)
	if err != nil {
		f.Close()
		harvesterOpenFiles.Add(-1)
		return err
	}

	h.source = File{File: f}
	return nil
}

func (h *FileHarvester) validateFile(f *os.File) error {
	info, err := f.Stat()
	if err != nil {
		return fmt.Errorf("Failed getting stats for file %s: %s", h.state.Source, err)
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("Tried to open non regular file: %q %s", info.Mode(), info.Name())
	}

	// Compares the stat of the opened file to the state given by the input. Abort if not match.
	if !os.SameFile(h.state.Fileinfo, info) {
		return errors.New("file info is not identical with opened file. Aborting harvesting and retrying file later again")
	}

	h.encoding, err = h.encodingFactory(f)
	if err != nil {

		if err == transform.ErrShortSrc {
			logp.Info("Initialising encoding for '%v' failed due to file being too short", f)
		} else {
			logp.Err("Initialising encoding for '%v' failed: %v", f, err)
		}
		return err
	}

	// get file offset. Only update offset if no error
	offset, err := h.initFileOffset(f)
	if err != nil {
		return err
	}

	logp.Debug("harvester", "Setting offset for file: %s. Offset: %d ", h.state.Source, offset)
	h.state.Offset = offset

	return nil
}

func (h *FileHarvester) reloadFileOffset() (int64, error) {
	hasState := h.source.HasState()
	if !hasState {
		return h.state.Offset, nil
	}

	var minOffset int64
	first := true

	for _, reuseReader := range h.forwarders {
		if first {
			minOffset = reuseReader.State.Offset
			first = false
			continue
		}
		if minOffset > reuseReader.State.Offset {
			minOffset = reuseReader.State.Offset
		}
	}

	if h.state.Offset == minOffset {
		return h.state.Offset, nil
	}

	//重新打开文件
	h.closeFile()
	h.state.Offset = minOffset
	return minOffset, h.Setup()
}

func (h *FileHarvester) initFileOffset(file *os.File) (int64, error) {
	// continue from last known offset
	if h.state.Offset > 0 {
		logp.Debug("harvester", "Set previous offset for file: %s. Offset: %d ", h.state.Source, h.state.Offset)
		return file.Seek(h.state.Offset, os.SEEK_SET)
	}

	// get offset from file in case of encoding factory was required to read some data.
	logp.Debug("harvester", "Setting offset for file based on seek: %s", h.state.Source)
	return file.Seek(0, os.SEEK_CUR)
}

// newLogFileReader creates a new reader to read log files
//
// It creates a chain of readers which looks as following:
//
//   limit -> (multiline -> timeout) -> strip_newline -> json -> encode -> line -> log_file
//
// Each reader on the left, contains the reader on the right and calls `Next()` to fetch more data.
// At the base of all readers the the log_file reader. That means in the data is flowing in the opposite direction:
//
//   log_file -> line -> encode -> json -> strip_newline -> (timeout -> multiline) -> limit
//
// log_file implements io.Reader interface and encode reader is an adapter for io.Reader to
// reader.Reader also handling file encodings. All other readers implement reader.Reader
func (h *FileHarvester) newLogFileReader() (reader.Reader, error) {
	var r reader.Reader
	var err error

	// TODO: NewLineReader uses additional buffering to deal with encoding and testing
	//       for new lines in input stream. Simple 8-bit based encodings, or plain
	//       don't require 'complicated' logic.
	h.log, err = NewLog(h.source, h.config.LogConfig)
	if err != nil {
		return nil, err
	}

	reader, err := debug.AppendReaders(h.log)
	if err != nil {
		return nil, err
	}

	// Configure MaxBytes limit for EncodeReader as multiplied by 4
	// for the worst case scenario where incoming UTF32 charchers are decoded to the single byte UTF-8 characters.
	// This limit serves primarily to avoid memory bload or potential OOM with expectedly long lines in the file.
	// The further size limiting is performed by LimitReader at the end of the readers pipeline as needed.
	encReaderMaxBytes := h.config.MaxBytes * 4

	r, err = readfile.NewEncodeReader(reader, readfile.Config{
		Codec:      h.encoding,
		BufferSize: h.config.BufferSize,
		MaxBytes:   encReaderMaxBytes,
	})
	if err != nil {
		return nil, err
	}

	if h.config.DockerJSON != nil {
		// Docker json-file format, add custom parsing to the pipeline
		r = readjson.New(r, h.config.DockerJSON.Stream, h.config.DockerJSON.Partial, h.config.DockerJSON.ForceCRI, h.config.DockerJSON.CRIFlags)
	}

	if h.config.JSON != nil {
		r = readjson.NewJSONReader(r, h.config.JSON)
	}

	r = readfile.NewStripNewline(r)

	if h.config.Multiline != nil {
		r, err = multiline.New(r, "\n", h.config.MaxBytes, h.config.Multiline)
		if err != nil {
			return nil, err
		}
	}

	return readfile.NewLimitReader(r, h.config.MaxBytes), nil
}
