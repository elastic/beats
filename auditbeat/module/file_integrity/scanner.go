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

package file_integrity

import (
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"

	"github.com/elastic/elastic-agent-libs/logp"
)

// scannerID is used as a global monotonically increasing counter for assigning
// a unique name to each scanner instance for logging purposes. Use
// atomic.AddUint32() to get a new value.
var scannerID uint32

type scanner struct {
	fileCount   uint64
	byteCount   uint64
	tokenBucket *rate.Limiter

	done   <-chan struct{}
	eventC chan Event

	log      *logp.Logger
	config   Config
	newPaths map[string]struct{}
}

// NewFileSystemScanner creates a new EventProducer instance that scans the
// configured file paths. Files and directories in new paths are recorded with
// the action `found`.
func NewFileSystemScanner(c Config, newPathsInConfig map[string]struct{}) (EventProducer, error) {
	return &scanner{
		log:      logp.NewLogger(moduleName).With("scanner_id", atomic.AddUint32(&scannerID, 1)),
		config:   c,
		newPaths: newPathsInConfig,
		eventC:   make(chan Event, 1),
	}, nil
}

// Start starts the EventProducer. The provided done channel can be used to stop
// the EventProducer prematurely. The returned Event channel will be closed when
// scanning is complete. The channel must drained otherwise the scanner will
// block.
func (s *scanner) Start(done <-chan struct{}) (<-chan Event, error) {
	s.done = done

	if s.config.ScanRateBytesPerSec > 0 {
		s.log.With(
			"bytes_per_sec", s.config.ScanRateBytesPerSec,
			"capacity_bytes", s.config.MaxFileSizeBytes).
			Debugf("Creating token bucket with rate %v/sec and capacity %v",
				s.config.ScanRatePerSec,
				s.config.MaxFileSize)

		s.tokenBucket = rate.NewLimiter(
			rate.Limit(s.config.ScanRateBytesPerSec), // Fill Rate
			int(s.config.MaxFileSizeBytes))           // Max Capacity
		s.tokenBucket.ReserveN(time.Now(), int(s.config.MaxFileSizeBytes))
	}

	go s.scan()
	return s.eventC, nil
}

// scan iterates over the configured paths and generates events for each file.
func (s *scanner) scan() {
	s.log.Debugw("File system scanner is starting", "file_path", s.config.Paths, "new_path", s.newPaths)
	defer s.log.Debug("File system scanner is stopping")
	defer close(s.eventC)
	startTime := time.Now()

	for _, path := range s.config.Paths {
		// Resolve symlinks to ensure we have an absolute path.
		evalPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			s.log.Warnw("Failed to scan", "file_path", path, "error", err)
			continue
		}

		// If action is None it will be filled later in the Metricset
		action := None
		if _, exists := s.newPaths[evalPath]; exists {
			action = InitialScan
		}

		if err = s.walkDir(evalPath, action); err != nil {
			s.log.Warnw("Failed to scan", "file_path", evalPath, "error", err)
		}
	}

	duration := time.Since(startTime)
	byteCount := atomic.LoadUint64(&s.byteCount)
	fileCount := atomic.LoadUint64(&s.fileCount)
	s.log.Infow("File system scan completed",
		"took", duration,
		"file_count", fileCount,
		"total_bytes", byteCount,
		"bytes_per_sec", float64(byteCount)/float64(duration)*float64(time.Second),
		"files_per_sec", float64(fileCount)/float64(duration)*float64(time.Second),
	)
}

func (s *scanner) walkDir(dir string, action Action) error {
	errDone := errors.New("done")
	startTime := time.Now()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			if !os.IsNotExist(err) {
				s.log.Warnw("Scanner is skipping a path because of an error",
					"file_path", path, "error", err)
			}
			return nil
		}

		if s.config.IsExcludedPath(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && !s.config.IsIncludedPath(path) {
			return nil
		}

		defer func() { startTime = time.Now() }()

		event := s.newScanEvent(path, info, err, action)
		event.rtt = time.Since(startTime)
		select {
		case s.eventC <- event:
		case <-s.done:
			return errDone
		}

		// Throttle reading and hashing rate.
		if event.Info != nil && len(event.Hashes) > 0 {
			s.throttle(event.Info.Size)
		}

		// Always traverse into the start dir.
		if !info.IsDir() || dir == path {
			return nil
		}

		// Only step into directories if recursion is enabled.
		// Skip symlinks to dirs.
		m := info.Mode()
		if !s.config.Recursive || m&os.ModeSymlink > 0 {
			return filepath.SkipDir
		}

		return nil
	})
	if err == errDone {
		err = nil
	}
	return err
}

func (s *scanner) throttle(fileSize uint64) {
	if s.tokenBucket == nil {
		return
	}

	reservation := s.tokenBucket.ReserveN(time.Now(), int(fileSize))
	if !reservation.OK() {
		// This would happen if the file size was greater than the token
		// buckets burst rate, but that can't happen because we don't hash files
		// larger than the burst rate (scan_max_file_size).
		return
	}

	delay := reservation.Delay()
	if delay == 0 {
		return
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-s.done:
	case <-timer.C:
	}
}

func (s *scanner) newScanEvent(path string, info os.FileInfo, err error, action Action) Event {
	event := NewEventFromFileInfo(path, info, err, action, SourceScan,
		s.config.MaxFileSizeBytes, s.config.HashTypes)

	// Update metrics.
	atomic.AddUint64(&s.fileCount, 1)
	if event.Info != nil {
		atomic.AddUint64(&s.byteCount, event.Info.Size)
	}
	return event
}
