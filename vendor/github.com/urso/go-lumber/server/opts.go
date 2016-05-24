package server

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"time"

	"github.com/urso/go-lumber/lj"
)

type Option func(*options) error

type options struct {
	timeout   time.Duration
	keepalive time.Duration
	decoder   jsonDecoder
	tls       *tls.Config
	v1        bool
	v2        bool
	ch        chan *lj.Batch
}

type jsonDecoder func([]byte, interface{}) error

func Keepalive(kl time.Duration) Option {
	return func(opt *options) error {
		if kl < 0 {
			return errors.New("keepalive must not be negative")
		}
		opt.keepalive = kl
		return nil
	}
}

func Timeout(to time.Duration) Option {
	return func(opt *options) error {
		if to < 0 {
			return errors.New("timeouts must not be negative")
		}
		opt.timeout = to
		return nil
	}
}

func TLS(tls *tls.Config) Option {
	return func(opt *options) error {
		opt.tls = tls
		return nil
	}
}

func Channel(c chan *lj.Batch) Option {
	return func(opt *options) error {
		opt.ch = c
		return nil
	}
}

func JSONDecoder(decoder func([]byte, interface{}) error) Option {
	return func(opt *options) error {
		opt.decoder = decoder
		return nil
	}
}

func V1(b bool) Option {
	return func(opt *options) error {
		opt.v1 = b
		return nil
	}
}

func V2(b bool) Option {
	return func(opt *options) error {
		opt.v2 = b
		return nil
	}
}

func applyOptions(opts []Option) (options, error) {
	o := options{
		decoder:   json.Unmarshal,
		timeout:   30 * time.Second,
		keepalive: 3 * time.Second,
		v1:        true,
		v2:        true,
		tls:       nil,
	}

	for _, opt := range opts {
		if err := opt(&o); err != nil {
			return o, err
		}
	}
	return o, nil
}
