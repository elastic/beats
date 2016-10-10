package http

import (
	"fmt"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/outputs"

	"github.com/elastic/beats/heartbeat/monitors"
)

type Config struct {
	Name string `config:"name"`

	URLs         []string      `config:"urls" validate:"required"`
	ProxyURL     string        `config:"proxy_url"`
	Timeout      time.Duration `config:"timeout"`
	MaxRedirects int           `config:"max_redirects"`

	Mode monitors.IPSettings `config:",inline"`

	// authentication
	Username string `config:"username"`
	Password string `config:"password"`

	// configure tls (if not configured HTTPS will use system defaults)
	TLS *outputs.TLSConfig `config:"ssl"`

	// http(s) ping validation
	Check checkConfig `config:"check"`
}

type checkConfig struct {
	// HTTP request configuration
	Method      string            `config:"request.method"`      // http request method
	SendHeaders map[string]string `config:"request.headers"`     // http request headers
	SendBody    string            `config:"request.body"`        // send body payload
	Compression compressionConfig `config:"request.compression"` // optionally compress payload

	// expected HTTP response configuration
	Status      uint16            `config:"response.status" verify:"min=0, max=699"`
	RecvHeaders map[string]string `config:"response.headers"`
	RecvBody    string            `config:"response.body"`

	// TODO:
	//  - add support for cookies
	//  - select HTTP version. golang lib will either use 1.1 or 2.0 if HTTPS is used, otherwise HTTP 1.1 . => implement/use specific http.RoundTripper implementation to change wire protocol/version being used
}

type compressionConfig struct {
	Type  string `config:"type"`
	Level int    `config:"level"`
}

var defaultConfig = Config{
	Name:         "http",
	Timeout:      16 * time.Second,
	MaxRedirects: 10,
	Mode:         monitors.DefaultIPSettings,
}

func (c *checkConfig) Validate() error {
	switch strings.ToUpper(c.Method) {
	case "HEAD", "GET", "POST":
	default:
		return fmt.Errorf("HTTP method '%v' not supported", c.Method)
	}

	return nil
}

func (c *compressionConfig) Validate() error {
	t := strings.ToLower(c.Type)
	if t != "" && t != "gzip" {
		return fmt.Errorf("compression type '%v' not supported", c.Type)
	}

	if t == "" {
		return nil
	}

	if !(0 <= c.Level && c.Level <= 9) {
		return fmt.Errorf("compression level %v invalid", c.Level)
	}

	return nil
}
