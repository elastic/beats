package sip

import (
    "github.com/elastic/beats/packetbeat/config"
)

type sipConfig struct {
    config.ProtocolCommon       `config:",inline"`
}

var (
    defaultConfig = sipConfig{
        ProtocolCommon: config.ProtocolCommon{},
    }
)
