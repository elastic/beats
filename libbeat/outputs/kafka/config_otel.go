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

	otelCfg := map[string]any{
		"brokers":          kCfg.Hosts,
		"protocol_version": string(kCfg.Version),
		"auth":             map[string]any{},
		"client_id":        kCfg.ClientID,
		"encoding":         kCfg.Codec.Namespace.Name(),
		"retry_on_failure": map[string]any{
			"enabled":          true,
			"initial_interval": kCfg.Backoff.Init,
			"max_interval":     kCfg.Backoff.Max,
			"max_elapsed_time": 290 * 360 * 24 * time.Hour, // 290 years, approximately the largest representable duration
		},
		"timeout": kCfg.Timeout,
		"producer": map[string]any{
			"compression":       kCfg.Compression,
			"max_message_bytes": *kCfg.MaxMessageBytes, // there is validation, therefore, it should be safe
			"required_acks":     *kCfg.RequiredACKs,    // there is validation, therefore, it should be safe
		},
	}

	// handle auth
	auth := otelCfg["auth"]
	authM, ok := auth.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("could not convert 'auth' node to map to add kerberos auth: %T", auth)
	}

	switch {
	case kCfg.Sasl.SaslMechanism != "":
		authM["sasl"] = map[string]any{ // TODO: only set if needed
			"mechanism": kCfg.Sasl.SaslMechanism,
			"username":  kCfg.Username,
			"password":  kCfg.Password,
			"version":   "1", // TODO: set to 0 for Azure EventHub, see https://github.com/elastic/sarama/blob/beats-fork/config.go#L68-L70
		}
	case kCfg.Kerberos.IsEnabled():
		err = kCfg.Kerberos.Validate()
		if err != nil {
			return nil, fmt.Errorf("invalid kerberos configuration: %s", err)
		}

		krb := map[string]any{
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
			krb["use_keytab"] = true
			krb["keytab_file"] = kCfg.Kerberos.KeyTabPath

		case "password":
			krb["username"] = kCfg.Kerberos.Username
			krb["password"] = kCfg.Kerberos.Password

		// kCfg.Kerberos.Validate() should have covered it already,
		// but better safe than sorry
		default:
			return nil, fmt.Errorf("invalid kerberos auth type: %s", authType)
		}

		authM["kerberos"] = krb
	default:
		authM["plain_text"] = map[string]any{
			"username": kCfg.Username,
			"password": kCfg.Password,
		}
	}

	if kCfg.TLS.IsEnabled() {
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
	}

	// we do not support more than one topic
	if len(kCfg.Topics) != 0 {
		return nil, errors.New("topics isn't supported in OTel mode")
	}

	topic, fromAttr, err := extractSingleTopic(kCfg.Topic)
	if err != nil {
		return nil, fmt.Errorf("could not extract topic: %s", err)
	}
	// topic is to be extracted from an attribute value on the event
	if fromAttr {
		otelCfg["topic_from_attribute"] = topic
		topic = ""
	}
	otelCfg["topic"] = topic

	// TODO: partially supported and unsupported fields. See https://docs.google.com/spreadsheets/d/1FlVVVzQsH5iRGAlPrMbpqNw23kQOn20WhtONSmM3UUI/edit?usp=sharing
	// for details
	return otelCfg, nil
}

// extractSingleTopic extracts the topic name or the attribute which value
// should be used as the topic name.
// It receives the Beats' topic template and returns a topic name and false or
// the event attribute name and true.
func extractSingleTopic(tmpl string) (string, bool, error) {
	st, err := fmtstr.CompileEvent(tmpl)
	if err != nil {
		return "", false, fmt.Errorf("failed to compile topic template")
	}
	if st.IsConst() {
		topic, err := st.Run(&beat.Event{})
		if err != nil {
			return "", false, fmt.Errorf("failed to topic template is a constant but failed to compile")
		}
		return topic, false, nil
	}

	if len(st.Fields()) > 1 {
		return "", false, errors.New("only one attribute supported")
	}

	attr := st.Fields()[0]

	ev := beat.Event{
		Fields: map[string]any{
			attr: "value",
		},
	}

	topic, err := st.Run(&ev)
	if topic != "value" {
		return "", false, fmt.Errorf("topic template is more than just a event attribute")
	}

	return attr, true, nil
}
