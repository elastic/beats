package kafka

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-viper/mapstructure/v2"

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

	config := map[string]interface{}{
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

		config["kerberos"] = map[string]interface{}{
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
			config["use_keytab"] = true
			config["keytab_file"] = kCfg.Kerberos.KeyTabPath

		case "password":
			config["username"] = kCfg.Kerberos.Username
			config["password"] = kCfg.Kerberos.Password

		// kCfg.Kerberos.Validate() should have covered it already,
		// but better safe than sorry
		default:
			return nil, fmt.Errorf("invalid kerberos auth type: %s", authType)
		}

	}

	// TODO: TLS:
	otelTLS, err := outputs.TLSCommonToOTel(kCfg.TLS)
	if err != nil {
		return nil, fmt.Errorf("could not parse TLS/SSL config: %s", err)
	}

	var configMapTLS map[string]any
	err = mapstructure.Decode(otelTLS, &configMapTLS)
	if err != nil {
		return nil, fmt.Errorf("could not decode TLS/SSL config: %s", err)
	}
	config["tls"] = configMapTLS

	// assume topic does not use templating and reject topics.
	// TODO: check if topic/topics match a single event attribute, if so, set
	// OTel 'topic_from_attribute'
	if len(kCfg.Topics) != 0 {
		return nil, errors.New("topics isn't supported in OTel mode")
	}
	config["topic"] = kCfg.Topic

	return config, nil
}
