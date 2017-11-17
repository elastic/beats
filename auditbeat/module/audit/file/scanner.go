package file

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/juju/ratelimit"

	"github.com/elastic/beats/libbeat/logp"
)

// scannerID is used as a global monotonically increasing counter for assigning
// a unique name to each scanner instance for logging purposes. Use
// atomic.AddUint32() to get a new value.
var scannerID uint32

type scanner struct {
	fileCount   uint64
	byteCount   uint64
	tokenBucket *ratelimit.Bucket

	done   <-chan struct{}
	eventC chan Event

	logID     string // Unique ID to correlate log messages to a single instance.
	logPrefix string
	config    Config
}

// NewFileSystemScanner creates a new EventProducer instance that scans the
// configured file paths.
func NewFileSystemScanner(c Config) (EventProducer, error) {
	logID := fmt.Sprintf("[scanner-%v]", atomic.AddUint32(&scannerID, 1))
	return &scanner{
		logID:     logID,
		logPrefix: fmt.Sprintf("%v %v", logPrefix, logID),
		config:    c,
		eventC:    make(chan Event, 1),
	}, nil
}

// Start starts the EventProducer. The provided done channel can be used to stop
// the EventProducer prematurely. The returned Event channel will be closed when
// scanning is complete. The channel must drained otherwise the scanner will
// block.
func (s *scanner) Start(done <-chan struct{}) (<-chan Event, error) {
	s.done = done

	if s.config.ScanRateBytesPerSec > 0 {
		debugf("%v creating token bucket with rate %v/sec and capacity %v",
			s.logID, s.config.ScanRatePerSec,
			humanize.Bytes(s.config.MaxFileSizeBytes))

		s.tokenBucket = ratelimit.NewBucketWithRate(
			float64(s.config.ScanRateBytesPerSec)/2., // Fill Rate
			int64(s.config.MaxFileSizeBytes))         // Max Capacity
		s.tokenBucket.TakeAvailable(math.MaxInt64)
	}

	go s.scan()
	return s.eventC, nil
}

// scan iterates over the configured paths and generates events for each file.
func (s *scanner) scan() {
	if logp.IsDebug(metricsetName) {
		debugf("%v File system scanner is starting for paths [%v].",
			s.logID, strings.Join(s.config.Paths, ", "))
		defer debugf("%v File system scanner is stopping.", s.logID)
	}
	defer close(s.eventC)
	startTime := time.Now()

	for _, path := range s.config.Paths {
		// Resolve symlinks to ensure we have an absolute path.
		evalPath, err := filepath.EvalSymlinks(path)
		if err != nil {
			logp.Warn("%v failed to scan %v: %v", s.logPrefix, path, err)
			continue
		}

		if err = s.walkDir(evalPath); err != nil {
			logp.Warn("%v failed to scan %v: %v", s.logPrefix, evalPath, err)
		}
	}

	duration := time.Since(startTime)
	byteCount := atomic.LoadUint64(&s.byteCount)
	fileCount := atomic.LoadUint64(&s.fileCount)
	logp.Info("%v File system scan completed after %v (%v files, %v bytes, %v/sec, %f files/sec).",
		s.logPrefix, duration, s.fileCount, byteCount,
		humanize.Bytes(uint64(float64(byteCount)/float64(duration)*float64(time.Second))),
		float64(fileCount)/float64(duration)*float64(time.Second))
}

func (s *scanner) walkDir(dir string) error {
	errDone := errors.New("done")
	startTime := time.Now()
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		defer func() { startTime = time.Now() }()

		event := s.newScanEvent(path, info, err)
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

	wait := s.tokenBucket.Take(int64(fileSize))
	if wait > 0 {
		timer := time.NewTimer(wait)
		select {
		case <-timer.C:
		case <-s.done:
		}
	}
}

func (s *scanner) newScanEvent(path string, info os.FileInfo, err error) Event {
	event := NewEventFromFileInfo(path, info, err, None, SourceScan,
		s.config.MaxFileSizeBytes, s.config.HashTypes)

	// Update metrics.
	atomic.AddUint64(&s.fileCount, 1)
	if event.Info != nil {
		atomic.AddUint64(&s.byteCount, event.Info.Size)
	}
	return event
}
