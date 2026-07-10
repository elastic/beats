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

	"go.uber.org/zap"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

// harvestSession implements loginp.HarvesterSession for the filestream input.
//
// It keeps the file handle open across read slices but does NOT keep the reader
// pipeline: each ReadSlice seeks the file to the last published offset and
// builds a fresh non-blocking pipeline, dropping it when the slice ends. Because
// only N slices run at once (one per worker), live pipelines — and their 16 KiB
// scratch buffers — are O(workers), not O(files). The last published offset is
// always a clean message boundary, so re-seeking re-reads at most a partial
// trailing line; nothing is lost or duplicated.
//
// A session is operated by a single worker at a time (the scheduler guarantees
// one worker per source), so it needs no internal locking.
type harvestSession struct {
	inp     *filestream
	log     *logp.Logger
	src     fileSource
	cursor  loginp.Cursor
	metrics *loginp.Metrics

	file       File              // kept open across slices; nil when done/closed
	enc        encoding.Encoding // detected once at open, reused per slice
	state      state
	readOffset int64

	done          bool      // terminal reached at open (e.g. GZIP already at EOF)
	closed        bool      // Close has been called
	pendingDelete bool      // a worker must delete the file on the next slice
	openedAt      time.Time // when the session was opened; for close.reader.after_interval
	lastData      time.Time // last time a slice read a message; for close_inactive
}

// OpenSession opens (or resumes) a reading session for the source. It opens the
// file handle and detects the encoding once; the reader pipeline is built per
// slice in ReadSlice. It implements loginp.SessionHarvester.
func (inp *filestream) OpenSession(
	ctx input.Context,
	src loginp.Source,
	cursor loginp.Cursor,
	metrics *loginp.Metrics,
) (loginp.HarvesterSession, error) {
	fs, ok := src.(fileSource)
	if !ok {
		return nil, fmt.Errorf("not file source")
	}

	log := ctx.Logger.WithLazy(
		zap.String("path", fs.newPath), zap.String("state-id", src.Name()))
	st := initState(log, cursor, fs)

	s := &harvestSession{
		inp:      inp,
		log:      log,
		src:      fs,
		cursor:   cursor,
		metrics:  metrics,
		state:    st,
		openedAt: time.Now(),
		lastData: time.Now(),
	}

	if st.EOF {
		log.Debugf("GZIP file already read to EOF, not reading it again, file name '%s'",
			fs.newPath)
		s.done = true
		return s, nil
	}

	f, enc, truncated, err := inp.openFile(log, fs.newPath, st.Offset)
	if err != nil {
		log.Errorf("File could not be opened for reading: %v", err)
		return nil, err
	}
	if truncated {
		s.state.Offset = 0
	}
	s.file = f
	s.enc = enc
	s.readOffset = s.state.Offset

	return s, nil
}

// ReadSlice reads and publishes events until the file has no data currently
// available (SliceYield) or a terminal condition is reached (SliceDone). It
// implements loginp.HarvesterSession.
func (s *harvestSession) ReadSlice(
	ctx input.Context,
	p loginp.Publisher,
) (loginp.SliceVerdict, error) {
	if s.done || s.file == nil {
		return loginp.SliceDone, nil
	}

	// The waker flagged this file for deletion (close_inactive + delete.enabled).
	// The delete is done here, on a worker, so its grace-period wait does not
	// block the waker.
	if s.pendingDelete {
		s.pendingDelete = false
		if err := s.inp.deleteFile(ctx, s.log, s.cursor, s.src.newPath); err != nil {
			return loginp.SliceDone, fmt.Errorf("cannot remove file '%s': %w", s.src.newPath, err)
		}
		return loginp.SliceDone, nil
	}

	isGZIP := s.src.desc.GZIP

	// Position the file at the last published offset (undoing any read-ahead
	// from the previous slice) and build a fresh non-blocking pipeline for this
	// slice. asPooled() keeps the file handle open when the pipeline is closed.
	if _, err := s.file.Seek(s.state.Offset, io.SeekStart); err != nil {
		return loginp.SliceDone,
			fmt.Errorf("cannot seek '%s' to offset %d: %w", s.src.newPath, s.state.Offset, err)
	}
	r, logReader, err := s.inp.buildPipeline(s.log, ctx.Cancelation, s.file, s.enc, s.src, s.state.Offset)
	if err != nil {
		return loginp.SliceDone,
			fmt.Errorf("cannot build reader pipeline for '%s': %w", s.src.newPath, err)
	}
	defer r.Close() // keepFileOpen: closes the pipeline, leaves the fd open

	var deadline time.Time
	if s.inp.sliceBudget > 0 {
		deadline = time.Now().Add(s.inp.sliceBudget)
	}

	for ctx.Cancelation.Err() == nil {
		if !deadline.IsZero() && time.Now().After(deadline) {
			s.readOffset = logReader.ReadOffset()
			s.log.Debugf("Slice time budget reached: %s; yielding.", s.src.newPath)
			return loginp.SliceYield, nil
		}

		message, err := r.Next()
		if err != nil {
			switch {
			case errors.Is(err, ErrWouldBlock):
				// No complete message available right now: park.
				s.readOffset = logReader.ReadOffset()
				s.log.Debugf("End of file reached: %s; Backoff now.", s.src.newPath)
				return loginp.SliceYield, nil
			case errors.Is(err, io.EOF):
				// EOF only reaches here for closeable files (close_eof, GZIP,
				// archived); tailing files yield via ErrWouldBlock instead.
				s.log.Debugf("EOF has been reached. Closing. Path='%s'", s.src.newPath)
				if s.inp.deleterConfig.Enabled {
					if derr := s.inp.deleteFile(ctx, s.log, s.cursor, s.src.newPath); derr != nil {
						return loginp.SliceDone,
							fmt.Errorf("cannot remove file '%s': %w", s.src.newPath, derr)
					}
				}
				return loginp.SliceDone, nil
			case errors.Is(err, ErrFileTruncate):
				// The file shrank: stop reading and let the prospector restart
				// the harvester with a reset offset.
				s.log.Infof("File was truncated, nothing to read. Path='%s'", s.src.newPath)
				return loginp.SliceDone, nil
			case errors.Is(err, ErrClosed):
				return loginp.SliceDone, nil
			default:
				s.log.Errorf("Read line error: %v", err)
				s.metrics.ProcessingErrors.Inc()
				if isGZIP {
					s.metrics.ProcessingGZIPErrors.Inc()
				}
				return loginp.SliceDone, nil
			}
		}

		s.state.Offset += int64(message.Bytes) + int64(message.Offset)

		if flags, ferr := message.Fields.GetValue("log.flags"); ferr == nil {
			if flagsList, ok := flags.([]string); ok && slices.Contains(flagsList, "truncated") {
				s.metrics.MessagesTruncated.Add(1)
				if isGZIP {
					// Truncation shouldn't happen for GZIP files, but as
					// we cannot guarantee it, we account for it anyway.
					s.metrics.MessagesGZIPTruncated.Add(1)
				}
			}
		}
		s.metrics.MessagesRead.Inc()
		if isGZIP {
			s.metrics.MessagesGZIPRead.Inc()
		}

		// close_inactive measures time since the file last produced data to read,
		// not since an event was last published: a file whose lines are all
		// filtered out (include_lines/exclude_lines) is still active and must not
		// be closed as inactive. Mark activity here, before the drop check.
		s.lastData = time.Now()

		if message.IsEmpty() || (s.inp.hasLineFilter && s.inp.isDroppedLine(s.log, message.Content)) {
			continue
		}

		//nolint:gosec // message.Bytes is always positive
		s.metrics.BytesProcessed.Add(uint64(message.Bytes))
		if isGZIP {
			//nolint:gosec // message.Bytes is always positive
			s.metrics.BytesGZIPProcessed.Add(uint64(message.Bytes))
		}

		if s.inp.takeOver.Enabled {
			_ = mapstr.AddTags(message.Fields, []string{"take_over"})
		}

		if isGZIP {
			if perr, ok := (message.Private).(error); ok && errors.Is(perr, io.EOF) {
				s.state.EOF = true
			}
		}

		if err := p.Publish(message.ToEvent(), s.state); err != nil {
			s.metrics.ProcessingErrors.Inc()
			if isGZIP {
				s.metrics.ProcessingGZIPErrors.Inc()
			}
			return loginp.SliceDone, err
		}

		s.metrics.EventsProcessed.Inc()
		s.metrics.ProcessingTime.Update(time.Since(message.Ts).Nanoseconds())
		if isGZIP {
			s.metrics.EventsGZIPProcessed.Inc()
			s.metrics.ProcessingGZIPTime.Update(time.Since(message.Ts).Nanoseconds())
		}
	}

	return loginp.SliceDone, ctx.Cancelation.Err()
}

// Poll evaluates a parked session: whether the file grew (resume), met a close
// condition (close), or is unchanged (keep parked). It only stats the file; it
// never reads or publishes. It implements loginp.HarvesterSession.
func (s *harvestSession) Poll() loginp.PollResult {
	if s.done || s.file == nil {
		return loginp.PollClose
	}

	// close.reader.after_interval closes the harvester a fixed time after it was
	// opened, regardless of activity.
	if interval := s.inp.closerConfig.Reader.AfterInterval; interval > 0 &&
		time.Since(s.openedAt) > interval {
		s.log.Debugf("close.reader.after_interval reached for %s", s.src.newPath)
		return loginp.PollClose
	}

	closer := s.inp.closerConfig.OnStateChange

	fi, statErr := s.file.Stat()
	if statErr != nil {
		if closer.Removed && errors.Is(statErr, os.ErrNotExist) {
			s.log.Debugf("close.on_state_change.removed and file %s has been removed", s.src.newPath)
			return loginp.PollClose
		}
		// Unexpected stat error: keep the file open hoping it recovers.
		return loginp.PollPark
	}

	if closer.Renamed && !isSameFile(s.src.newPath, fi) {
		s.log.Debugf("close.on_state_change.renamed and file %s has been renamed", s.src.newPath)
		return loginp.PollClose
	}

	// On Unix, a removed (unlinked) file can still be stat-able while its fd
	// remains open, so the stat above won't fail; also check the fd-level
	// removed state to catch that case.
	if closer.Removed && file.IsRemoved(s.file.OSFile()) {
		s.log.Debugf("close.on_state_change.removed and file %s has been removed", s.src.newPath)
		return loginp.PollClose
	}

	// GZIP offsets are tracked on the decompressed stream, so a size comparison
	// is invalid; resume until the session reads to EOF (SliceDone).
	if s.src.desc.GZIP || fi.Size() != s.readOffset {
		return loginp.PollResume
	}

	if closer.Inactive > 0 && time.Since(s.lastData) > closer.Inactive {
		s.log.Debugf("File is inactive. Closing. Path='%s'", s.src.newPath)
		if s.inp.deleterConfig.Enabled {
			// Hand the file to a worker to run the (possibly blocking) delete
			// rather than deleting on the waker goroutine.
			s.pendingDelete = true
			return loginp.PollResume
		}
		return loginp.PollClose
	}

	return loginp.PollPark
}

// Offset returns the current read offset.
func (s *harvestSession) Offset() int64 { return s.state.Offset }

// IsGZIP reports whether the session reads a GZIP-compressed source.
func (s *harvestSession) IsGZIP() bool { return s.src.desc.GZIP }

// Close releases the file handle held by the session.
func (s *harvestSession) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	if s.file != nil {
		err := s.file.Close()
		s.file = nil
		s.log.Debugf("Closed reader. Path='%s'", s.src.newPath)
		if err != nil {
			s.log.Errorf("Error stopping filestream reader: %v", err)
		}
		return err
	}
	return nil
}
