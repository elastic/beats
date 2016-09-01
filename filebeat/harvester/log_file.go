package harvester

import (
	"io"
	"os"
	"sync"
	"time"

	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

type LogFile struct {
	fs           source.FileSource
	offset       int64
	config       harvesterConfig
	lastTimeRead time.Time
	backoff      time.Duration
	done         chan struct{}
	singleClose  sync.Once
}

func NewLogFile(
	fs source.FileSource,
	config harvesterConfig,
) (*LogFile, error) {
	var offset int64
	if seeker, ok := fs.(io.Seeker); ok {
		var err error
		offset, err = seeker.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, err
		}
	}

	return &LogFile{
		fs:           fs,
		offset:       offset,
		config:       config,
		lastTimeRead: time.Now(),
		backoff:      config.Backoff,
		done:         make(chan struct{}),
	}, nil
}

// Read reads from the reader and updates the offset
// The total number of bytes read is returned.
func (r *LogFile) Read(buf []byte) (int, error) {

	totalN := 0

	for {
		select {
		case <-r.done:
			return 0, ErrClosed
		default:
		}

		n, err := r.fs.Read(buf)
		if n > 0 {
			r.offset += int64(n)
			r.lastTimeRead = time.Now()
		}
		totalN += n

		// Read from source completed without error
		// Either end reached or buffer full
		if err == nil {
			// reset backoff for next read
			r.backoff = r.config.Backoff
			return totalN, nil
		}

		// Move buffer forward for next read
		buf = buf[n:]

		// Checks if an error happened or buffer is full
		// If buffer is full, cannot continue reading.
		// Can happen if n == bufferSize + io.EOF error
		err = r.errorChecks(err)
		if err != nil || len(buf) == 0 {
			return totalN, err
		}

		logp.Debug("harvester", "End of file reached: %s; Backoff now.", r.fs.Name())
		r.wait()
	}
}

// errorChecks checks how the given error should be handled based on the config options
func (r *LogFile) errorChecks(err error) error {

	if err != io.EOF {
		logp.Err("Unexpected state reading from %s; error: %s", r.fs.Name(), err)
		return err
	}

	// Stdin is not continuable
	if !r.fs.Continuable() {
		logp.Debug("harvester", "Source is not continuable: %s", r.fs.Name())
		return err
	}

	if err == io.EOF && r.config.CloseEOF {
		return err
	}

	// Refetch fileinfo to check if the file was truncated or disappeared.
	// Errors if the file was removed/rotated after reading and before
	// calling the stat function
	info, statErr := r.fs.Stat()
	if statErr != nil {
		logp.Err("Unexpected error reading from %s; error: %s", r.fs.Name(), statErr)
		return statErr
	}

	// check if file was truncated
	if info.Size() < r.offset {
		logp.Debug("harvester",
			"File was truncated as offset (%d) > size (%d): %s", r.offset, info.Size(), r.fs.Name())
		return ErrFileTruncate
	}

	// Check file wasn't read for longer then CloseInactive
	age := time.Since(r.lastTimeRead)
	if age > r.config.CloseInactive {
		return ErrInactive
	}

	if r.config.CloseRenamed {
		// Check if the file can still be found under the same path
		if !file.IsSameFile(r.fs.Name(), info) {
			return ErrRenamed
		}
	}

	if r.config.CloseRemoved {
		// Check if the file name exists. See https://github.com/elastic/filebeat/issues/93
		_, statErr := os.Stat(r.fs.Name())

		// Error means file does not exist.
		if statErr != nil {
			return ErrRemoved
		}
	}

	return nil
}

func (r *LogFile) wait() {

	// Wait before trying to read file again. File reached EOF.
	select {
	case <-r.done:
		return
	case <-time.After(r.backoff):
	}

	// Increment backoff up to maxBackoff
	if r.backoff < r.config.MaxBackoff {
		r.backoff = r.backoff * time.Duration(r.config.BackoffFactor)
		if r.backoff > r.config.MaxBackoff {
			r.backoff = r.config.MaxBackoff
		}
	}
}

func (r *LogFile) Close() {
	// Make sure reader is only closed once
	r.singleClose.Do(func() {
		close(r.done)
		// Note: File reader is not closed here because that leads to race conditions
	})
}
