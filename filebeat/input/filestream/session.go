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
	"sync/atomic"
	"time"

	loginp "github.com/elastic/beats/v7/filebeat/input/filestream/internal/input-logfile"
	input "github.com/elastic/beats/v7/filebeat/input/v2"
	"github.com/elastic/beats/v7/libbeat/common/file"
	"github.com/elastic/beats/v7/libbeat/reader"
	"github.com/elastic/beats/v7/libbeat/reader/readfile/encoding"
	"github.com/elastic/beats/v7/pkg/logp"
	"github.com/elastic/beats/v7/pkg/mapstr"
)

// harvestSession implements loginp.HarvesterSession for the filestream input.
//
// It keeps the file handle open across read slices but, normally, does NOT keep
// the reader pipeline: each ReadSlice seeks the file to the last published
// offset and builds a fresh non-blocking pipeline, dropping it when the slice
// ends. Because only N slices run at once (one per worker), live pipelines —
// and their 16 KiB scratch buffers — are O(workers), not O(files). The last
// published offset is always a clean message boundary, so re-seeking re-reads
// at most a partial trailing line; nothing is lost or duplicated.
//
// The one exception is a pipeline whose parsers hold buffered state for an
// incomplete message (e.g. the lines of an unfinished multiline event). That
// state cannot be rebuilt by re-reading from the published offset without
// re-publishing, and dropping it would split the event, so the pipeline is
// retained across slices — only for as long as it has buffered state. The
// published offset still lags the buffered lines, so a crash re-reads them.
// The buffered state is flushed when its deadline elapses, when a close
// condition is met, or on a read_until_eof drain.
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
	pendingClose  bool      // a close condition was met while pipeline held buffered state: flush it, then finish
	openedAt      time.Time // when the session was opened; for close.reader.after_interval
	lastData      time.Time // last time the file produced data to read; for close_inactive

	// pipeline and logReader are the reader pipeline retained across slices
	// while its parsers hold buffered state for an incomplete message; both are
	// nil otherwise. pendingDeadline is when that state must be flushed, or
	// zero if it has no timeout (e.g. multiline count mode).
	pipeline        reader.Reader
	logReader       *logFile
	pendingDeadline time.Time

	// metricsOffset, when non-nil, is the shared atomic the harvester ingestion
	// progress metrics read from; updated as ReadSlice publishes messages.
	// cleanupMetricsOffset removes it on Close. Both are nil for GZIP sources:
	// their progress can't be represented by a plain offset/size comparison.
	metricsOffset        *atomic.Int64
	cleanupMetricsOffset func()

	// harvesterState is this session's entry in the input's fileStateTable.
	harvesterState *openFileState
}

// OpenSession opens (or resumes) a reading session for the source. id is the
// source's current registry key, used to key harvester progress metrics. It
// opens the file handle and detects the encoding once; the reader pipeline is
// built per slice in ReadSlice. It implements loginp.SessionHarvester.
func (inp *filestream) OpenSession(
	ctx input.Context,
	src loginp.Source,
	id string,
	cursor loginp.Cursor,
	metrics *loginp.Metrics,
) (loginp.HarvesterSession, error) {
	fs, ok := src.(fileSource)
	if !ok {
		return nil, fmt.Errorf("not file source")
	}

	log := ctx.Logger
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

	s.harvesterState = inp.harvesterState.NewHandle()
	// If fstat fails, the entry has no usable open file identity.
	if fi, err := f.Stat(); err == nil {
		s.harvesterState.PinOSState(file.GetOSState(fi))
	}
	if truncated {
		s.state.Offset = 0
	}
	s.file = f
	s.enc = enc
	s.readOffset = s.state.Offset

	if !fs.desc.GZIP {
		s.metricsOffset, s.cleanupMetricsOffset = metrics.RegisterHarvesterOffset(id, s.state.Offset)
	}

	return s, nil
}

// ReadSlice reads and publishes events until the file has no data currently
// available (SliceYield), the slice's time budget elapsed with data still
// available (SliceBudget), or a terminal condition is reached (SliceDone). It
// implements loginp.HarvesterSession.
//
// A pipeline whose parsers still buffer an incomplete message when the slice
// ends is retained for the next slice (see harvestSession); otherwise it is
// closed so an idle file holds no pipeline.
func (s *harvestSession) ReadSlice(
	ctx input.Context,
	p loginp.Publisher,
) (loginp.SliceVerdict, error) {
	if s.done || s.file == nil {
		return loginp.SliceDone, nil
	}

	// Poll flagged the session to finish: either a close condition was met while
	// the pipeline buffered an incomplete message, or the file went inactive with
	// delete.enabled. Buffered events are published first (the delete compares
	// the published offset with the file size), and the delete runs here, on a
	// worker, so its grace-period wait does not block the waker.
	if s.pendingClose || s.pendingDelete {
		return s.finishFlagged(ctx, p)
	}

	isGZIP := s.src.desc.GZIP

	// Take ownership of the retained pipeline, if any. It is put back by retain
	// only if it still buffers state when the slice ends.
	r, logReader := s.pipeline, s.logReader
	s.pipeline, s.logReader, s.pendingDeadline = nil, nil, time.Time{}

	if r == nil {
		// Position the file at the last published offset (undoing any read-ahead
		// from the previous slice) and build a fresh non-blocking pipeline for
		// this slice. A retained pipeline must NOT be re-seeked: its buffered
		// lines are already behind the file position. asPooled() keeps the file
		// handle open when the pipeline is closed.
		if _, err := s.file.Seek(s.state.Offset, io.SeekStart); err != nil {
			return loginp.SliceDone,
				fmt.Errorf("cannot seek '%s' to offset %d: %w", s.src.newPath, s.state.Offset, err)
		}
		var err error
		r, logReader, err = s.inp.buildPipeline(s.log, ctx.Cancelation, s.file, s.enc, s.src, s.state.Offset, s.harvesterState)
		if err != nil {
			return loginp.SliceDone,
				fmt.Errorf("cannot build reader pipeline for '%s': %w", s.src.newPath, err)
		}
	}

	retained := false
	defer func() {
		if !retained {
			_ = r.Close() // keepFileOpen: closes the pipeline, leaves the fd open
		}
	}()

	// retain ends the slice's use of the pipeline: it records how far the file
	// was read and keeps the pipeline for the next slice if its parsers buffer
	// an incomplete message.
	retain := func() {
		s.syncReadOffset(logReader)
		if pending, pd := reader.PendingDeadline(r); pending {
			s.pipeline, s.logReader, s.pendingDeadline = r, logReader, pd
			retained = true
		}
	}

	var deadline time.Time
	if s.inp.sliceBudget != 0 {
		// A negative budget makes the deadline already past, so the slice yields on
		// its first budget check without reading.
		deadline = time.Now().Add(s.inp.sliceBudget)
	}

	for ctx.Cancelation.Err() == nil {
		if !deadline.IsZero() && time.Now().After(deadline) {
			retain()
			s.log.Debugf("Slice time budget reached: %s; yielding.", s.src.newPath)
			return loginp.SliceBudget, nil
		}

		message, err := r.Next()
		if err != nil {
			switch {
			case errors.Is(err, ErrWouldBlock):
				// No complete message available right now. If the parsers buffer
				// an incomplete message whose deadline elapsed, no further line
				// arrived in time: flush it as a complete event. Next ran first,
				// so a line written just before the deadline is not missed.
				if pending, pd := reader.PendingDeadline(r); pending && !pd.IsZero() && !time.Now().Before(pd) {
					flushed, ok, ferr := reader.FlushPending(r, true)
					if ferr != nil {
						s.logReadError(ferr, isGZIP)
						return loginp.SliceDone, nil
					}
					if ok {
						if perr := s.processMessage(flushed, p, isGZIP); perr != nil {
							return loginp.SliceDone, perr
						}
						continue
					}
				}

				// Park, keeping the pipeline only if it still buffers state.
				retain()
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
				s.logReadError(err, isGZIP)
				return loginp.SliceDone, nil
			}
		}

		// close_inactive measures time since the file last produced data to read,
		// not since an event was last published: a file whose lines are all
		// filtered out (include_lines/exclude_lines) is still active and must not
		// be closed as inactive. Mark activity here, before the drop check. Only
		// messages read from the file count: a message flushed from the parsers'
		// buffer carries no new data.
		s.lastData = time.Now()

		if err := s.processMessage(message, p, isGZIP); err != nil {
			return loginp.SliceDone, err
		}
	}

	return loginp.SliceDone, ctx.Cancelation.Err()
}

// processMessage accounts for a message handed up by the pipeline (read from
// it or flushed from its buffered state), advances the published offset and
// publishes it, unless it is empty or dropped by include_lines/exclude_lines.
// It returns an error only if publishing failed.
func (s *harvestSession) processMessage(message reader.Message, p loginp.Publisher, isGZIP bool) error {
	s.state.Offset += int64(message.Bytes) + int64(message.Offset)
	if s.metricsOffset != nil {
		s.metricsOffset.Store(s.state.Offset)
	}

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

	if message.IsEmpty() || (s.inp.hasLineFilter && s.inp.isDroppedLine(s.log, message.Content)) {
		return nil
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
		return err
	}

	s.metrics.EventsProcessed.Inc()
	s.metrics.ProcessingTime.Update(time.Since(message.Ts).Nanoseconds())
	if isGZIP {
		s.metrics.EventsGZIPProcessed.Inc()
		s.metrics.ProcessingGZIPTime.Update(time.Since(message.Ts).Nanoseconds())
	}
	return nil
}

// logReadError logs and counts an unexpected read (or flush) error.
func (s *harvestSession) logReadError(err error, isGZIP bool) {
	s.log.Errorf("Read line error: %v", err)
	s.metrics.ProcessingErrors.Inc()
	if isGZIP {
		s.metrics.ProcessingGZIPErrors.Inc()
	}
}

// syncReadOffset records how far into the file the pipeline has read. Bytes
// read since the last slice mean the file is active, even if they are still
// buffered by a parser and nothing was published yet.
func (s *harvestSession) syncReadOffset(lr *logFile) {
	offset := lr.ReadOffset()
	if offset > s.readOffset {
		s.lastData = time.Now()
	}
	s.readOffset = offset
}

// finishFlagged ends a session Poll flagged for closing: it publishes any
// buffered message, then runs the pending delete, and reports the session done.
func (s *harvestSession) finishFlagged(
	ctx input.Context,
	p loginp.Publisher,
) (loginp.SliceVerdict, error) {
	del := s.pendingDelete
	s.pendingClose, s.pendingDelete = false, false

	if err := s.flushPipeline(p); err != nil {
		return loginp.SliceDone, err
	}

	if del {
		if err := s.inp.deleteFile(ctx, s.log, s.cursor, s.src.newPath); err != nil {
			return loginp.SliceDone, fmt.Errorf("cannot remove file '%s': %w", s.src.newPath, err)
		}
	}
	return loginp.SliceDone, nil
}

// flushPipeline publishes every message the retained pipeline still buffers
// and releases it. It is a no-op if no pipeline is retained. It returns an
// error only if publishing failed.
func (s *harvestSession) flushPipeline(p loginp.Publisher) error {
	r := s.pipeline
	if r == nil {
		return nil
	}
	defer s.closePipeline()

	isGZIP := s.src.desc.GZIP
	for {
		msg, ok, err := reader.FlushPending(r, false)
		if err != nil {
			s.logReadError(err, isGZIP)
			return nil
		}
		if !ok {
			return nil
		}
		if err := s.processMessage(msg, p, isGZIP); err != nil {
			return err
		}
	}
}

// closePipeline closes the retained pipeline, if any. Closing it releases its
// scratch buffers; the file handle stays open.
func (s *harvestSession) closePipeline() {
	if s.pipeline != nil {
		_ = s.pipeline.Close()
	}
	s.pipeline, s.logReader, s.pendingDeadline = nil, nil, time.Time{}
}

// Flush publishes the message a retained pipeline still buffers and releases
// the pipeline. The runner calls it after draining the file to EOF on a
// read_until_eof shutdown, when no further line will complete the message. It
// implements loginp.HarvesterSession.
func (s *harvestSession) Flush(ctx input.Context, p loginp.Publisher) error {
	if err := ctx.Cancelation.Err(); err != nil {
		// Cancelled (e.g. the drain timeout): publishing would fail anyway.
		s.closePipeline()
		return err
	}
	return s.flushPipeline(p)
}

// pendingTimedOut reports whether the retained pipeline buffers a message whose
// flush deadline has elapsed.
func (s *harvestSession) pendingTimedOut() bool {
	return s.pipeline != nil && !s.pendingDeadline.IsZero() && !time.Now().Before(s.pendingDeadline)
}

// closeOrFlush is the Poll result for a met close condition: close at once if
// nothing is buffered, otherwise hand the session to a worker to publish the
// buffered message first.
func (s *harvestSession) closeOrFlush() loginp.PollResult {
	if s.pipeline != nil {
		s.pendingClose = true
		return loginp.PollResume
	}
	return loginp.PollClose
}

// Poll evaluates a parked session: whether the file grew or buffered parser
// state timed out (resume), met a close condition (close), or is unchanged
// (keep parked). It only stats the file; it never reads or publishes. A close
// condition met while the pipeline buffers an incomplete message resumes the
// session instead, so a worker publishes that message before the session is
// closed. It implements loginp.HarvesterSession.
func (s *harvestSession) Poll() loginp.PollResult {
	if s.done || s.file == nil {
		return loginp.PollClose
	}

	// close.reader.after_interval closes the harvester a fixed time after it was
	// opened, regardless of activity.
	if interval := s.inp.closerConfig.Reader.AfterInterval; interval > 0 &&
		time.Since(s.openedAt) > interval {
		s.log.Debugf("close.reader.after_interval reached for %s", s.src.newPath)
		return s.closeOrFlush()
	}

	closer := s.inp.closerConfig.OnStateChange

	// Evaluated once so every decision below agrees on it.
	timedOut := s.pendingTimedOut()

	fi, statErr := s.file.Stat()
	if statErr != nil {
		if closer.Removed && errors.Is(statErr, os.ErrNotExist) {
			s.log.Debugf("close.on_state_change.removed and file %s has been removed", s.src.newPath)
			return s.closeOrFlush()
		}
		// Unexpected stat error: keep the file open hoping it recovers. Flushing
		// timed-out state needs no stat, and parking would re-poll it at once.
		if timedOut {
			return loginp.PollResume
		}
		return loginp.PollPark
	}

	if closer.Renamed && !isSameFile(s.src.newPath, fi) {
		s.log.Debugf("close.on_state_change.renamed and file %s has been renamed", s.src.newPath)
		return s.closeOrFlush()
	}

	// On Unix, a removed (unlinked) file can still be stat-able while its fd
	// remains open, so the stat above won't fail; also check the fd-level
	// removed state to catch that case.
	if closer.Removed && file.IsRemoved(s.file.OSFile()) {
		s.log.Debugf("close.on_state_change.removed and file %s has been removed", s.src.newPath)
		return s.closeOrFlush()
	}

	// GZIP offsets are tracked on the decompressed stream, so a size comparison
	// is invalid; resume until the session reads to EOF (SliceDone). A buffered
	// message past its deadline also resumes, to be flushed: the file may not
	// have changed at all.
	if s.src.desc.GZIP || fi.Size() != s.readOffset || timedOut {
		return loginp.PollResume
	}

	if closer.Inactive > 0 && time.Since(s.lastData) > closer.Inactive {
		s.log.Debugf("File is inactive. Closing. Path='%s'", s.src.newPath)
		if s.inp.deleterConfig.Enabled {
			// Hand the file to a worker to run the (possibly blocking) delete
			// rather than deleting on the waker goroutine. The worker publishes
			// any buffered message before deleting.
			s.pendingDelete = true
			return loginp.PollResume
		}
		return s.closeOrFlush()
	}

	return loginp.PollPark
}

// TrackSource lists the open file under the source's current identity with
// that identity's scanner descriptor.
func (s *harvestSession) TrackSource(current loginp.Source) {
	fs, ok := current.(fileSource)
	if !ok {
		return
	}
	s.harvesterState.Publish(fs.Name(), fs.desc)
}

// Offset returns the offset up to which events have been published. It lags
// ReadOffset while a parser buffers an incomplete message or a trailing partial
// line was read.
func (s *harvestSession) Offset() int64 { return s.state.Offset }

// ReadOffset returns how far into the file the session has read, including
// bytes buffered by a parser or a trailing partial line.
func (s *harvestSession) ReadOffset() int64 { return s.readOffset }

// PendingDeadline returns when the message buffered by the retained pipeline
// must be flushed, or the zero time if none is buffered or it has no timeout.
func (s *harvestSession) PendingDeadline() time.Time { return s.pendingDeadline }

// IsGZIP reports whether the session reads a GZIP-compressed source.
func (s *harvestSession) IsGZIP() bool { return s.src.desc.GZIP }

// Close releases the file handle held by the session.
func (s *harvestSession) Close() error {
	if s.closed {
		return nil
	}
	s.closed = true
	// Release a retained pipeline (and its scratch buffers) before the file
	// it reads from.
	s.closePipeline()
	if s.cleanupMetricsOffset != nil {
		s.cleanupMetricsOffset()
	}
	// Remove the recorded identity before closing its file.
	s.inp.harvesterState.Deregister(s.harvesterState)
	s.harvesterState = nil
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
