package v2

import (
	"encoding/json"
	"errors"
	"time"
)

// Option type to be passed to New/Dial functions.
type Option func(*options) error

type options struct {
	timeout     time.Duration
	encoder     jsonEncoder
	compressLvl int
}

type jsonEncoder func(interface{}) ([]byte, error)

// JSONEncoder client option configuring the encoder used to convert events
// to json. The default is `json.Marshal`.
func JSONEncoder(encoder func(interface{}) ([]byte, error)) Option {
	return func(opt *options) error {
		opt.encoder = encoder
		return nil
	}
}

// Timeout client option configuring read/write timeout.
func Timeout(to time.Duration) Option {
	return func(opt *options) error {
		if to < 0 {
			return errors.New("timeouts must not be negative")
		}
		opt.timeout = to
		return nil
	}
}

// CompressionLevel client option setting the gzip compression level (0 to 9).
func CompressionLevel(l int) Option {
	return func(opt *options) error {
		if !(0 <= l && l <= 9) {
			return errors.New("compression level must be within 0 and 9")
		}
		opt.compressLvl = l
		return nil
	}
}

func applyOptions(opts []Option) (options, error) {
	o := options{
		encoder: json.Marshal,
		timeout: 30 * time.Second,
	}

	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return o, err
		}
	}
	return o, nil
}
