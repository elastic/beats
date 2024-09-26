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

package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-viper/mapstructure/v2"

	"github.com/elastic/beats/v7/libbeat/beat"
	"github.com/elastic/beats/v7/libbeat/common/fmtstr"
	"github.com/elastic/beats/v7/libbeat/outputs"
	"github.com/elastic/elastic-agent-libs/config"
)

// ToOTelConfig ...
// it received only the kafka output config: `outputs.kafka`.
// It returns the config ready to be placed inside `exporters.kafka`
func ToOTelConfig(cfg *config.C) (map[string]any, error) {
	// assuming enabled: true
	kCfg, err := readConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("could not read kafka config: %s", err)
	}
	if err := kCfg.Version.Validate(); err != nil {
		return nil, err
	}

	if kCfg.Codec.Namespace.Name() != "json" {
		return nil, fmt.Errorf(
			"only 'json' codec is supported in OTel mode, found %s",
			kCfg.Codec.Namespace.Name())
	}

	otelCfg := map[string]interface{}{
		"brokers": map[string]interface{}{
			"protocol_version": string(kCfg.Version),
		},
		"auth": map[string]interface{}{
			"plain_text": map[string]interface{}{
				"username": kCfg.Username,
				"password": kCfg.Password,
			},

			"sasl": map[string]interface{}{ // TODO: only set if needed
				"mechanism": kCfg.Sasl.SaslMechanism,
				"username":  kCfg.Username,
				"password":  kCfg.Password,
				"version":   "1", // TODO: set to 0 for Azure EventHub, see https://github.com/elastic/sarama/blob/beats-fork/config.go#L68-L70
			},
		},
		"client_id": kCfg.ClientID,
		"retry_on_failure": map[string]interface{}{
			"enabled":          true,
			"initial_interval": kCfg.Backoff.Init,
			"max_interval":     kCfg.Backoff.Max,
			"max_elapsed_time": 290 * 360 * 24 * time.Hour, // 290 years, approximately the largest representable duration
		},
		"timeout": kCfg.Timeout,
		"producer": map[string]interface{}{
			"compression":       kCfg.Compression,
			"max_message_bytes": *kCfg.MaxMessageBytes, // there is validation, therefore, it should be safe
			"required_acks":     *kCfg.RequiredACKs,    // there is validation, therefore, it should be safe
		},
	}

	if kCfg.Kerberos != nil {
		err = kCfg.Kerberos.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid kerberos configuration: %s", err)
		}

		otelCfg["kerberos"] = map[string]interface{}{
			"config_file":              kCfg.Kerberos.ConfigPath,
			"realm":                    kCfg.Kerberos.Realm,
			"disable_fast_negotiation": !kCfg.Kerberos.EnableFAST,
		}

		authType, err := kCfg.Kerberos.AuthType.String()
		if err != nil {
			return nil, fmt.Errorf("invalid kerberos auth type: %s", err)
		}

		switch authType {
		case "keytab":
			otelCfg["use_keytab"] = true
			otelCfg["keytab_file"] = kCfg.Kerberos.KeyTabPath

		case "password":
			otelCfg["username"] = kCfg.Kerberos.Username
			otelCfg["password"] = kCfg.Kerberos.Password

		// kCfg.Kerberos.Validate() should have covered it already,
		// but better safe than sorry
		default:
			return nil, fmt.Errorf("invalid kerberos auth type: %s", authType)
		}

	}

	otelTLS, err := outputs.TLSCommonToOTel(kCfg.TLS)
	if err != nil {
		return nil, fmt.Errorf("could not parse TLS/SSL config: %s", err)
	}

	var configMapTLS map[string]any
	err = mapstructure.Decode(otelTLS, &configMapTLS)
	if err != nil {
		return nil, fmt.Errorf("could not decode TLS/SSL config: %s", err)
	}
	otelCfg["tls"] = configMapTLS

	// we do not support more than one topic
	if len(kCfg.Topics) != 0 {
		return nil, errors.New("topics isn't supported in OTel mode")
	}

	topic, err := extractSingleTopic(kCfg.Topic)
	if err != nil {
		return nil, fmt.Errorf("could not extract topic: %s", err)
	}
	otelCfg["topic"] = topic

	return otelCfg, nil
}

func extractSingleTopic(tmpl string) (string, error) {
	st, err := fmtstr.CompileEvent(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to compile topic template")
	}
	if st.IsConst() {
		return st.Run(&beat.Event{})
	}

	if len(st.Fields()) > 1 {
		return "", errors.New("only one attribute supported")
	}

	attr := st.Fields()[0]

	ev := beat.Event{
		Fields: map[string]any{
			attr: "value",
		},
	}

	topic, err := st.Run(&ev)
	if topic != "value" {
		return "", fmt.Errorf("topic template is more than just a event attribute")
	}

	return attr, nil
}
