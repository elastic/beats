package config

import (
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/procs"
	"github.com/urso/ucfg"
)

type Config struct {
	Interfaces InterfacesConfig
	Flows      *Flows
	Protocols  map[string]*ucfg.Config
	Shipper    publisher.ShipperConfig
	Procs      procs.ProcsConfig
	RunOptions droppriv.RunOptions
	Logging    logp.Logging
	Filter     map[string]interface{}
}

type InterfacesConfig struct {
	Device         string
	Type           string
	File           string
	With_vlans     bool
	Bpf_filter     string
	Snaplen        int
	Buffer_size_mb int
	TopSpeed       bool
	Dumpfile       string
	OneAtATime     bool
	Loop           int
}

type Flows struct {
	Timeout string
	Period  string
}

type ProtocolCommon struct {
	Ports              []int `config:"ports"`
	SendRequest        bool  `config:"send_request"`
	SendResponse       bool  `config:"send_response"`
	TransactionTimeout int   `config:"transaction_timeout"`
}

// Config Singleton
var ConfigSingleton Config
