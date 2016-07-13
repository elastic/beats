package config

import (
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/packetbeat/procs"
)

type Config struct {
	Interfaces InterfacesConfig          `config:"interfaces"`
	Flows      *Flows                    `config:"flows"`
	Protocols  map[string]*common.Config `config:"protocols"`
	Procs      procs.ProcsConfig         `config:"procs"`
	RunOptions droppriv.RunOptions
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
