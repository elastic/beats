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

// Code generated by moq; DO NOT EDIT.
// github.com/matryer/moq

//go:build linux

package journald

import (
	"sync"

	"github.com/elastic/beats/v7/filebeat/input/journald/pkg/journalctl"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
)

// Ensure, that journalReaderMock does implement journalReader.
// If this is not the case, regenerate this file with moq.
var _ journalReader = &journalReaderMock{}

// journalReaderMock is a mock implementation of journalReader.
//
//	func TestSomethingThatUsesjournalReader(t *testing.T) {
//
//		// make and configure a mocked journalReader
//		mockedjournalReader := &journalReaderMock{
//			CloseFunc: func() error {
//				panic("mock out the Close method")
//			},
//			NextFunc: func(cancel input.Canceler) (journalctl.JournalEntry, error) {
//				panic("mock out the Next method")
//			},
//		}
//
//		// use mockedjournalReader in code that requires journalReader
//		// and then make assertions.
//
//	}
type journalReaderMock struct {
	// CloseFunc mocks the Close method.
	CloseFunc func() error

	// NextFunc mocks the Next method.
	NextFunc func(cancel input.Canceler) (journalctl.JournalEntry, error)

	// calls tracks calls to the methods.
	calls struct {
		// Close holds details about calls to the Close method.
		Close []struct {
		}
		// Next holds details about calls to the Next method.
		Next []struct {
			// Cancel is the cancel argument value.
			Cancel input.Canceler
		}
	}
	lockClose sync.RWMutex
	lockNext  sync.RWMutex
}

// Close calls CloseFunc.
func (mock *journalReaderMock) Close() error {
	if mock.CloseFunc == nil {
		panic("journalReaderMock.CloseFunc: method is nil but journalReader.Close was just called")
	}
	callInfo := struct {
	}{}
	mock.lockClose.Lock()
	mock.calls.Close = append(mock.calls.Close, callInfo)
	mock.lockClose.Unlock()
	return mock.CloseFunc()
}

// CloseCalls gets all the calls that were made to Close.
// Check the length with:
//
//	len(mockedjournalReader.CloseCalls())
func (mock *journalReaderMock) CloseCalls() []struct {
} {
	var calls []struct {
	}
	mock.lockClose.RLock()
	calls = mock.calls.Close
	mock.lockClose.RUnlock()
	return calls
}

// Next calls NextFunc.
func (mock *journalReaderMock) Next(cancel input.Canceler) (journalctl.JournalEntry, error) {
	if mock.NextFunc == nil {
		panic("journalReaderMock.NextFunc: method is nil but journalReader.Next was just called")
	}
	callInfo := struct {
		Cancel input.Canceler
	}{
		Cancel: cancel,
	}
	mock.lockNext.Lock()
	mock.calls.Next = append(mock.calls.Next, callInfo)
	mock.lockNext.Unlock()
	return mock.NextFunc(cancel)
}

// NextCalls gets all the calls that were made to Next.
// Check the length with:
//
//	len(mockedjournalReader.NextCalls())
func (mock *journalReaderMock) NextCalls() []struct {
	Cancel input.Canceler
} {
	var calls []struct {
		Cancel input.Canceler
	}
	mock.lockNext.RLock()
	calls = mock.calls.Next
	mock.lockNext.RUnlock()
	return calls
}
