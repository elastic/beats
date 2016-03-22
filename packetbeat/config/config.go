package config

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/procs"
)

type Config struct {
	Interfaces InterfacesConfig
	Flows      *Flows
	Protocols  map[string]*common.Config
	Shipper    publisher.ShipperConfig
	Procs      procs.ProcsConfig
	RunOptions droppriv.RunOptions
	Logging    logp.Logging
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
	Ports              []int         `config:"ports"`
	SendRequest        bool          `config:"send_request"`
	SendResponse       bool          `config:"send_response"`
	TransactionTimeout time.Duration `config:"transaction_timeout"`
}

// Config Singleton
var ConfigSingleton Config
