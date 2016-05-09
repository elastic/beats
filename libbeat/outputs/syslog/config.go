package syslog

import (
	"time"

	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/outputs/transport"
)

type config struct {
	Path           string                `config:"path"`
	Host           string                `config:"host"`
	Port           int                   `config:"port"`
	MaxRetries     int                   `config:"max_retries"`
	SyslogProgram  string                `config:"default_syslog_program"`
	SyslogPriority uint64                `config:"default_syslog_priority"`
	SyslogSeverity uint64                `config:"default_syslog_program"`
	TLS            *outputs.TLSConfig    `config:"tls"`
	Timeout        time.Duration         `config:"timeout"`
	Proxy          transport.ProxyConfig `config:",inline"`
}

// We set the default values for program, priority and severity here, and
// override them in PublishEvents if they're set on individual files.
//   Priority 1: user-level messages.
//   Severity 6: informational messages.
var (
	defaultConfig = config{
		Port:           514,
		MaxRetries:     3,
		SyslogProgram:  "filebeat",
		SyslogPriority: 1,
		SyslogSeverity: 6,
	}
)
