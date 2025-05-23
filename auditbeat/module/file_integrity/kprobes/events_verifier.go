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

package kprobes

import (
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/sys/unix"
)

type eventID struct {
	path string
	op   uint32
}

var eventGenerators = []func(*eventsVerifier, string, string) error{
	// create file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		file, err := os.OpenFile(targetFilePath, os.O_RDWR|os.O_CREATE, 0o644)
		if err != nil {
			return err
		}
		defer file.Close()
		e.addEventToExpect(targetFilePath, unix.IN_CREATE)
		return nil
	},
	// truncate file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Truncate(targetFilePath, 0); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_MODIFY)
		return nil
	},
	// write to file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		file, err := os.OpenFile(targetFilePath, os.O_WRONLY, 0o644)
		if err != nil {
			return err
		}
		defer file.Close()
		if _, err := file.WriteString("test"); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_MODIFY)
		return nil
	},
	// change owner of file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Chown(targetFilePath, os.Getuid(), os.Getgid()); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// change mode of file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Chmod(targetFilePath, 0o700); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// change times of file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := unix.Utimes(targetFilePath, []unix.Timeval{
			unix.NsecToTimeval(time.Now().UnixNano()),
			unix.NsecToTimeval(time.Now().UnixNano()),
		}); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// add attribute to file - generates 1 event
	// Note that this may fail if the filesystem doesn't support extended attributes
	// This is allVerified we just skip adding the respective event to verify
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		attrName := "user.myattr"
		attrValue := []byte("Hello, xattr!")
		if err := unix.Setxattr(targetFilePath, attrName, attrValue, 0); err != nil {
			if !errors.Is(err, unix.EOPNOTSUPP) {
				return err
			}
		} else {
			e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		}
		return nil
	},
	// move file - generates 2 events
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Rename(targetFilePath, targetMovedFilePath); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_MOVED_FROM)
		e.addEventToExpect(targetMovedFilePath, unix.IN_MOVED_TO)
		return nil
	},
	// remove file - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Remove(targetMovedFilePath); err != nil {
			return err
		}
		e.addEventToExpect(targetMovedFilePath, unix.IN_DELETE)
		return nil
	},
	// create a directory - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Mkdir(targetFilePath, 0o600); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_CREATE)
		return nil
	},
	// change mode of directory - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Chmod(targetFilePath, 0o644); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// change owner of directory - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Chown(targetFilePath, os.Getuid(), os.Getgid()); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// add attribute to directory - generates 1 event
	// Note that this may fail if the filesystem doesn't support extended attributes
	// This is allVerified we just skip adding the respective event to verify
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		attrName := "user.myattr"
		attrValue := []byte("Hello, xattr!")
		if err := unix.Setxattr(targetFilePath, attrName, attrValue, 0); err != nil {
			if !errors.Is(err, unix.EOPNOTSUPP) {
				return err
			}
		} else {
			e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		}
		return nil
	},
	// change times of directory - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := unix.Utimes(targetFilePath, []unix.Timeval{
			unix.NsecToTimeval(time.Now().UnixNano()),
			unix.NsecToTimeval(time.Now().UnixNano()),
		}); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_ATTRIB)
		return nil
	},
	// move directory - generates 2 events
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Rename(targetFilePath, targetMovedFilePath); err != nil {
			return err
		}
		e.addEventToExpect(targetFilePath, unix.IN_MOVED_FROM)
		e.addEventToExpect(targetMovedFilePath, unix.IN_MOVED_TO)
		return nil
	},
	// remove the directory - generates 1 event
	func(e *eventsVerifier, targetFilePath string, targetMovedFilePath string) error {
		if err := os.Remove(targetMovedFilePath); err != nil {
			return err
		}
		e.addEventToExpect(targetMovedFilePath, unix.IN_DELETE)
		return nil
	},
}

type eventsVerifier struct {
	sync.Mutex
	basePath         string
	eventsToExpect   map[eventID]int
	eventsToExpectNr int
}

func newEventsVerifier(basePath string) (*eventsVerifier, error) {
	return &eventsVerifier{
		basePath:       basePath,
		eventsToExpect: make(map[eventID]int),
	}, nil
}

func (e *eventsVerifier) validateEvent(path string, _ uint32, op uint32) error {
	e.Lock()
	defer e.Unlock()

	eID := eventID{
		path: path,
		op:   op,
	}
	_, exists := e.eventsToExpect[eID]

	if !exists {
		return ErrVerifyUnexpectedEvent
	}

	e.eventsToExpect[eID]--
	return nil
}

// addEventToExpect adds an event to the eventsVerifier's list of expected events.
func (e *eventsVerifier) addEventToExpect(path string, op uint32) {
	e.eventsToExpectNr++

	eID := eventID{
		path: path,
		op:   op,
	}
	_, exists := e.eventsToExpect[eID]

	if !exists {
		e.eventsToExpect[eID] = 1
		return
	}

	e.eventsToExpect[eID]++
}

func (e *eventsVerifier) GenerateEvents() error {
	targetFilePath := filepath.Join(e.basePath, "validate_file")
	targetMovedFilePath := targetFilePath + "_moved"

	for _, genFunc := range eventGenerators {
		e.Lock()
		if err := genFunc(e, targetFilePath, targetMovedFilePath); err != nil {
			e.Unlock()
			return err
		}
		e.Unlock()
	}

	return nil
}

// Verified checks that all expected events filled during GenerateEvents() are present without any missing
// or duplicated.
func (e *eventsVerifier) Verified() error {
	if e.eventsToExpectNr == 0 {
		return ErrVerifyNoEventsToExpect
	}

	for _, status := range e.eventsToExpect {
		switch {
		case status < 0:
			return ErrVerifyOverlappingEvents
		case status > 0:
			return ErrVerifyMissingEvents
		}
	}

	return nil
}
