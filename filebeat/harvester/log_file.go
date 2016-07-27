package harvester

import (
	"io"
	"os"
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
}

func NewLogFile(
	fs source.FileSource,
	config harvesterConfig,
	done chan struct{},
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
		done:         done,
	}, nil
}

func (r *LogFile) Read(buf []byte) (int, error) {

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

		// reset backoff
		if err == nil {
			r.backoff = r.config.Backoff
			return n, nil
		}

		if err != io.EOF {
			logp.Err("Unexpected state reading from %s; error: %s", r.fs.Name(), err)
			return n, err
		}

		// Stdin is not continuable
		if !r.fs.Continuable() {
			logp.Debug("harvester", "Source is not continuable: %s", r.fs.Name())
			return n, err
		}

		err = r.errorChecks(err)
		if err != nil {
			return n, err
		}

		logp.Debug("harvester", "End of file reached: %s; Backoff now.", r.fs.Name())
		buf = buf[n:]
		if len(buf) == 0 {
			return n, nil
		}
		r.wait()
	}
}

// errorChecks checks how the given error should be handled based on the config options
func (r *LogFile) errorChecks(err error) error {
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
			"File was truncated as offset (%s) > size (%s): %s", r.offset, info.Size(), r.fs.Name())
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
