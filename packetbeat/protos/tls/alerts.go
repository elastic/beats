// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package tls

import (
	"errors"
	"fmt"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type alertSeverity uint8
type alertCode uint8

type alert struct {
	severity alertSeverity
	code     alertCode
}

var alertNames = map[alertCode]string{
	0:   "close_notify",
	10:  "unexpected_message",
	20:  "bad_record_mac",
	21:  "decryption_failed",
	22:  "record_overflow",
	30:  "decompression_failure",
	40:  "handshake_failure",
	41:  "no_certificate_RESERVED",
	42:  "bad_certificate",
	43:  "unsupported_certificate",
	44:  "certificate_revoked",
	45:  "certificate_expired",
	46:  "certificate_unknown",
	47:  "illegal_parameter",
	48:  "unknown_ca",
	49:  "access_denied",
	50:  "decode_error",
	51:  "decrypt_error",
	60:  "export_restriction_RESERVED",
	70:  "protocol_version",
	71:  "insufficient_security",
	80:  "internal_error",
	86:  "inappropriate_fallback",
	90:  "user_canceled",
	100: "no_renegotiation",
	110: "unsupported_extension",
	111: "certificate_unobtainable",
	112: "unrecognized_name",
	113: "bad_certificate_status_response",
	114: "bad_certificate_hash_value",
	115: "unknown_psk_identity",
}

var (
	errRead = errors.New("Buffer read error")
)

func (severity alertSeverity) String() string {
	switch severity {
	case 1:
		return "warning"
	case 2:
		return "fatal"
	}
	return fmt.Sprintf("(unknown:0x%02x)", int(severity))
}

func (alertCode alertCode) String() string {
	if str, ok := alertNames[alertCode]; ok {
		return str
	}
	return fmt.Sprintf("(unknown:0x%02x)", int(alertCode))
}

func (alert alert) toMap(source string) common.MapStr {
	return common.MapStr{
		"severity": alert.severity.String(),
		"code":     int(alert.code),
		"type":     alert.code.String(),
		"source":   source,
	}
}

func (parser *parser) parseAlert(buf *bufferView) error {
	if buf.length() != 2 {
		if isDebug {
			debugf("ignoring encrypted alert")
		}
		return nil
	}

	var severity, code uint8
	if !buf.read8(0, &severity) || !buf.read8(1, &code) {
		return errRead
	}
	if severity < 1 || severity > 2 {
		logp.Warn("invalid severity in alert: %v", severity)
	}
	alert := alert{alertSeverity(severity), alertCode(code)}
	if isDebug {
		debugf("Got alert %v", alert)
	}
	parser.alerts = append(parser.alerts, alert)
	return nil
}
