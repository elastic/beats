package harvester

import (
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/elastic/beats/filebeat/input"
	"github.com/elastic/beats/libbeat/logp"
)

type logFileReader struct {
	fs        FileSource
	offset    int64
	config    logFileReaderConfig
	truncated bool

	lastTimeRead time.Time
	backoff      time.Duration
}

type logFileReaderConfig struct {
	forceClose         bool
	maxInactive        time.Duration
	backoffDuration    time.Duration
	maxBackoffDuration time.Duration
	backoffFactor      int
}

var (
	errFileTruncate = errors.New("detected file being truncated")
	errForceClose   = errors.New("file must be closed")
	errInactive     = errors.New("file inactive")
)

func newLogFileReader(
	fs FileSource,
	config logFileReaderConfig,
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
		backoff:      config.backoffDuration,
	}, nil
}

func (r *logFileReader) Read(buf []byte) (int, error) {
	fmt.Println("call Read")

	if r.truncated {
		var offset int64
		if seeker, ok := r.fs.(io.Seeker); ok {
			var err error
			offset, err = seeker.Seek(0, os.SEEK_CUR)
			if err != nil {
				return 0, err
			}
		}
		r.offset = offset
		r.truncated = false
	}

	for {
		n, err := r.fs.Read(buf)
		if n > 0 {
			fmt.Printf("did read(%v): '%s'\n", n, buf[:n])

			r.offset += int64(n)
			r.lastTimeRead = time.Now()
		}
		if err == nil {
			// reset backoff
			r.backoff = r.config.backoffDuration
			fmt.Printf("return size: %v\n", n)
			return n, nil
		}

		continuable := r.fs.Continuable()
		fmt.Printf("error: %v, continuable: %v\n", err, continuable)

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
				"File was truncated as offset (%s) > size (%s). Begin reading file from offset 0: %s",
				r.offset, info.Size(), r.fs.Name())
			r.truncated = true
			return n, errFileTruncate
		}

		age := time.Since(r.lastTimeRead)
		if age > r.config.maxInactive {
			// If the file hasn't change for longer then maxInactive, harvester stops
			// and file handle will be closed.
			return n, errInactive
		}

		if r.config.forceClose {
			// Check if the file name exists (see #93)
			_, statErr := os.Stat(r.fs.Name())

			// Error means file does not exist. If no error, check if same file. If
			// not close as rotated.
			if statErr != nil || !input.IsSameFile(r.fs.Name(), info) {
				logp.Info("Force close file: %s; error: %s", r.fs.Name(), statErr)
				// Return directly on windows -> file is closing
				return n, errForceClose
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
	if r.backoff < r.config.maxBackoffDuration {
		r.backoff = r.backoff * time.Duration(r.config.backoffFactor)
		if r.backoff > r.config.maxBackoffDuration {
			r.backoff = r.config.maxBackoffDuration
		}
	}
}
