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

// +build !integration

package log

import (
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"io"
)

func TestCheckStatus_closeRename(t *testing.T) {
	logFile, err := getTestFilePath()
	assert.Nil(t, err)

	file, err := newTestFileWithContent(logFile, "line 1\nline 2\nline 3\n")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	defer file.Close()

	// Open file for reading
	readFile, err := os.Open(logFile)

	defer readFile.Close()
	assert.Nil(t, err)

	source := File{File: readFile}
	config := LogConfig{
		CloseInactive:       5 * time.Second,
		Backoff:             100 * time.Millisecond,
		MaxBackoff:          1 * time.Second,
		BackoffFactor:       2,
		CloseRemoved:        true,
		CloseRenamed:        true,
		CheckStatusInterval: 0 * time.Second,
	}

	l, err := NewLog(source, config)
	assert.Nil(t, err)
	assert.NotNil(t, l)
	defer l.Close()

	newFileName := readFile.Name() + "2"
	err = os.Rename(readFile.Name(), newFileName)

	newFileWithOldName, err := newTestFileWithContent(logFile, "line 1\nline 2\nline 3\n")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	defer newFileWithOldName.Close()

	err = l.checkStatus()
	assert.NotNil(t, err)
	assert.Equal(t, ErrRenamed, err)
}

func TestCheckStatus_closeRemove(t *testing.T) {
	logFile, err := getTestFilePath()
	assert.Nil(t, err)

	file, err := newTestFileWithContent(logFile, "line 1\nline 2\nline 3\n")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	defer file.Close()

	// Open file for reading
	readFile, err := os.Open(logFile)
	defer readFile.Close()
	assert.Nil(t, err)

	source := File{File: readFile}
	config := LogConfig{
		CloseInactive:       5 * time.Second,
		Backoff:             100 * time.Millisecond,
		MaxBackoff:          1 * time.Second,
		BackoffFactor:       2,
		CloseRemoved:        true,
		CloseRenamed:        true,
		CheckStatusInterval: 0 * time.Second,
	}

	l, err := NewLog(source, config)
	assert.Nil(t, err)
	assert.NotNil(t, l)
	defer l.Close()

	err = os.Remove(readFile.Name())
	assert.Nil(t, err)

	err = l.checkStatus()
	assert.NotNil(t, err)
	assert.Equal(t, ErrRemoved, err)
}

func TestCheckStatusInterval(t *testing.T) {
	logFile, err := getTestFilePath()
	assert.Nil(t, err)

	file, err := newTestFileWithContent(logFile, "line 1\nline 2\nline 3\n")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	defer file.Close()

	file.startAddingContent("new line\n", 90*time.Millisecond)
	defer file.stop()

	// Open file for reading
	readFile, err := os.Open(logFile)

	defer readFile.Close()
	assert.Nil(t, err)

	source := File{File: readFile}
	config := LogConfig{
		CloseInactive:       5 * time.Second,
		Backoff:             100 * time.Millisecond,
		MaxBackoff:          1 * time.Second,
		BackoffFactor:       2,
		CloseRemoved:        true,
		CloseRenamed:        true,
		CloseEOF:            true,
		CheckStatusInterval: 1 * time.Second,
	}

	l, err := NewLog(source, config)
	assert.Nil(t, err)
	assert.NotNil(t, l)
	defer l.Close()

	assert.NotNil(t, l.checkStatusTicker, "checkStatusTicker should not be nil if CheckStatusInterval > 0")

	c := l.checkStatusC()
	assert.NotNil(t, c, "checkStatusChannel should not be nil if CheckStatus is enable")

	buf := make([]byte, 10)

	i := 0
	var receivedError error
	for {
		if i > 20 {
			assert.Fail(t, "Should notice that file is removed before 20th iteration or 2 secs")
			break
		}
		if i == 5 {
			// remove file on 10th iteration
			err = os.Remove(readFile.Name())
			assert.Nil(t, err)
		}
		_, receivedError = l.Read(buf)
		if receivedError != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
		i++
	}

	assert.Equal(t, ErrRemoved, receivedError)
}

func TestDisableCheckStatus(t *testing.T) {
	logFile, err := getTestFilePath()
	assert.Nil(t, err)

	file, err := newTestFileWithContent(logFile, "line 1\nline 2\nline 3\n")
	assert.Nil(t, err)
	assert.NotNil(t, file)
	defer file.Close()

	file.startAddingContent("new line\n", 90*time.Millisecond)
	defer file.stop()

	// Open file for reading
	readFile, err := os.Open(logFile)
	assert.Nil(t, err)

	source := File{File: readFile}
	config := LogConfig{
		CloseInactive: 5 * time.Second,
		Backoff:       100 * time.Millisecond,
		MaxBackoff:    1 * time.Second,
		BackoffFactor: 2,
		CloseRemoved:  true,
		CloseRenamed:  true,
		CloseEOF:      true,
	}

	l, err := NewLog(source, config)
	assert.Nil(t, err)
	assert.NotNil(t, l)
	defer l.Close()

	assert.Nil(t, l.checkStatusTicker, "checkStatusTicker should be nil if CheckStatusInterval is not set")

	c := l.checkStatusC()
	assert.Nil(t, c, "checkStatusChannel should be nil if CheckStatus is disable")

	buf := make([]byte, 10)

	i := 0
	var receivedError error
	for {
		if i >= 100 {
			break
		}
		if i == 20 {
			file.stop()
		}
		if i == 5 {
			// remove file on 10th iteration
			err = os.Remove(readFile.Name())
			assert.Nil(t, err)
		}
		_, receivedError = l.Read(buf)
		if receivedError != nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
		i++
	}

	assert.Equal(t, io.EOF, receivedError)
}

type fileForTesting struct {
	name    string
	file    *os.File
	writing chan struct{}
}

func newTestFileWithContent(filename string, content string) (*fileForTesting, error) {
	file, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	_, err = file.WriteString(content)

	if err != nil {
		file.Close()
		return nil, err
	}
	file.Sync()
	f := &fileForTesting{
		name: filename,
		file: file,
	}
	return f, nil
}

func (file *fileForTesting) startAddingContent(content string, duration time.Duration) {
	go func() {
		f, err := os.OpenFile(file.name, os.O_APPEND|os.O_WRONLY, os.ModeAppend)
		if file.writing != nil {
			file.stop()
		}
		file.writing = make(chan struct{})
		defer f.Close()
		if err != nil {
			return
		}
		for {
			select {
			case <-file.writing:
				break
			default:
				_, err = f.WriteString("new line\n")
				if err != nil {
					return
				}
				f.Sync()
			}
			time.Sleep(duration)
		}
	}()
}

func (file *fileForTesting) stop() {
	select {
	case <-file.writing:
		break
	default:
		close(file.writing)
	}
}

func (file *fileForTesting) Close() {
	file.file.Close()
}

func getTestFilePath() (string, error) {
	absPath, err := filepath.Abs("../../tests/files/logs/")
	if err != nil {
		return "", err
	}

	// All files starting with tmp are ignored
	logFile := absPath + "/tmp" + strconv.Itoa(rand.Int()) + ".log"
	return logFile, nil
}
