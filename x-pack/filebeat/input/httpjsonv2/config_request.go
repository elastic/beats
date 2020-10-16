package httpjsonv2

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transport/tlscommon"
	"github.com/elastic/beats/v7/x-pack/filebeat/input/httpjsonv2/internal/transforms"
)

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

func (c retryConfig) Validate() error {
	switch {
	case c.MaxAttempts != nil && *c.MaxAttempts <= 0:
		return errors.New("max_attempts must be greater than 0")
	case c.WaitMin != nil && *c.WaitMin <= 0:
		return errors.New("wait_min must be greater than 0")
	case c.WaitMax != nil && *c.WaitMax <= 0:
		return errors.New("wait_max must be greater than 0")
	}
	return nil
}

type rateLimitConfig struct {
	Limit     *transforms.Template `config:"limit"`
	Reset     *transforms.Template `config:"reset"`
	Remaining *transforms.Template `config:"remaining"`
}

func (c rateLimitConfig) Validate() error {
	if c.Limit == nil || c.Reset == nil || c.Remaining == nil {
		return errors.New("all rate_limit fields must have a value")
	}

	return nil
}

type urlConfig struct {
	*url.URL
}

func (u *urlConfig) Unpack(in string) error {
	parsed, err := url.Parse(in)
	if err != nil {
		return err
	}

	*u = urlConfig{URL: parsed}

	return nil
}

type requestConfig struct {
	URL        *urlConfig        `config:"url" validate:"required"`
	Method     string            `config:"method" validate:"required"`
	Body       *common.MapStr    `config:"body"`
	Timeout    *time.Duration    `config:"timeout"`
	SSL        *tlscommon.Config `config:"ssl"`
	Retry      retryConfig       `config:"retry"`
	RateLimit  *rateLimitConfig  `config:"rate_limit"`
	Transforms transforms.Config `config:"transforms"`
}

func (c requestConfig) Validate() error {

	switch strings.ToUpper(c.Method) {
	case "POST":
	case "GET":
		if c.Body != nil {
			return errors.New("body can't be used with method: \"GET\"")
		}
	default:
		return fmt.Errorf("unsupported method %q", c.Method)
	}

	if c.Timeout != nil && *c.Timeout <= 0 {
		return errors.New("timeout must be greater than 0")
	}

	if _, err := transforms.New(c.Transforms, "request"); err != nil {
		return err
	}

	return nil
}
