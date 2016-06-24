package reader

import (
	"errors"
	"io"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/harvester/source"
	"github.com/elastic/beats/filebeat/input/file"
	"github.com/elastic/beats/libbeat/logp"
)

var (
	ErrFileTruncate = errors.New("detected file being truncated")
	ErrForceClose   = errors.New("file must be closed")
	ErrInactive     = errors.New("file inactive")
)

type logFileReader struct {
	fs           source.FileSource
	offset       int64
	config       LogFileReaderConfig
	lastTimeRead time.Time
	backoff      time.Duration
	done         chan struct{}
}

type LogFileReaderConfig struct {
	ForceClose         bool
	CloseOlder         time.Duration
	BackoffDuration    time.Duration
	MaxBackoffDuration time.Duration
	BackoffFactor      int
}

func NewLogFileReader(
	fs source.FileSource,
	config LogFileReaderConfig,
	done chan struct{},
) (*logFileReader, error) {
	var offset int64
	if seeker, ok := fs.(io.Seeker); ok {
		var err error
		offset, err = seeker.Seek(0, os.SEEK_CUR)
		if err != nil {
			return nil, err
		}
	}

	return &logFileReader{
		fs:           fs,
		offset:       offset,
		config:       config,
		lastTimeRead: time.Now(),
		backoff:      config.BackoffDuration,
		done:         done,
	}, nil
}

func (r *logFileReader) Read(buf []byte) (int, error) {

	for {
		select {
		case <-r.done:
			return 0, nil
		default:
		}

		n, err := r.fs.Read(buf)
		if n > 0 {
			r.offset += int64(n)
			r.lastTimeRead = time.Now()
		}
		if err == nil {
			// reset backoff
			r.backoff = r.config.BackoffDuration
			return n, nil
		}

		continuable := r.fs.Continuable()
		if err == io.EOF && !continuable {
			logp.Info("Reached end of file: %s", r.fs.Name())
			return n, err
		}

		if err != io.EOF || !continuable {
			logp.Err("Unexpected state reading from %s; error: %s", r.fs.Name(), err)
			return n, err
		}

		// Refetch fileinfo to check if the file was truncated or disappeared.
		// Errors if the file was removed/rotated after reading and before
		// calling the stat function
		info, statErr := r.fs.Stat()
		if statErr != nil {
			logp.Err("Unexpected error reading from %s; error: %s", r.fs.Name(), statErr)
			return n, statErr
		}

		// handle fails if file was truncated
		if info.Size() < r.offset {
			logp.Debug("harvester",
				"File was truncated as offset (%s) > size (%s): %s",
				r.offset, info.Size(), r.fs.Name())
			return n, ErrFileTruncate
		}

		age := time.Since(r.lastTimeRead)
		if age > r.config.CloseOlder {
			// If the file hasn't change for longer then maxInactive, harvester stops
			// and file handle will be closed.
			return n, ErrInactive
		}

		if r.config.ForceClose {
			// Check if the file name exists (see #93)
			_, statErr := os.Stat(r.fs.Name())

			// Error means file does not exist. If no error, check if same file. If
			// not close as rotated.
			if statErr != nil || !file.IsSameFile(r.fs.Name(), info) {
				logp.Info("Force close file: %s; error: %s", r.fs.Name(), statErr)
				// Return directly on windows -> file is closing
				return n, ErrForceClose
			}
		}

		if err != io.EOF {
			logp.Err("Unexpected state reading from %s; error: %s", r.fs.Name(), err)
		}

		logp.Debug("harvester", "End of file reached: %s; Backoff now.", r.fs.Name())
		buf = buf[n:]
		if len(buf) == 0 {
			return n, nil
		}
		r.wait()
	}
}

func (r *logFileReader) wait() {
	// Wait before trying to read file wr.ch reached EOF again
	time.Sleep(r.backoff)

	// Increment backoff up to maxBackoff
	if r.backoff < r.config.MaxBackoffDuration {
		r.backoff = r.backoff * time.Duration(r.config.BackoffFactor)
		if r.backoff > r.config.MaxBackoffDuration {
			r.backoff = r.config.MaxBackoffDuration
		}
	}
}
