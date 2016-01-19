package config

import (
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/procs"
)

type Config struct {
	Interfaces InterfacesConfig
	Protocols  Protocols
	Output     map[string]outputs.MothershipConfig
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

type Protocols struct {
	Icmp     Icmp
	Amqp     Amqp
	Dns      Dns
	Http     Http
	Memcache Memcache
	Mysql    Mysql
	Mongodb  Mongodb
	Pgsql    Pgsql
	Redis    Redis
	Thrift   Thrift
}

type ProtocolCommon struct {
	Ports              []int `yaml:"ports"`
	SendRequest        *bool `yaml:"send_request"`
	SendResponse       *bool `yaml:"send_response"`
	TransactionTimeout *int  `yaml:"transaction_timeout"`
}

type Icmp struct {
	Enabled            bool
	SendRequest        *bool `yaml:"send_request"`
	SendResponse       *bool `yaml:"send_response"`
	TransactionTimeout *int  `yaml:"transaction_timeout"`
}

type Amqp struct {
	ProtocolCommon              `yaml:",inline"`
	ParseHeaders               *bool `yaml:"parse_headers"`
	ParseArguments             *bool `yaml:"parse_arguments"`
	MaxBodyLength             *int `yaml:"max_body_length"`
	HideConnectionInformation *bool `yaml:"hide_connection_information"`
}

type Dns struct {
	ProtocolCommon      `yaml:",inline"`
	Include_authorities *bool
	Include_additionals *bool
}

type Http struct {
	ProtocolCommon       `yaml:",inline"`
	Send_all_headers     *bool
	Send_headers         []string
	Split_cookie         *bool
	Real_ip_header       *string
	Include_body_for     []string
	Hide_keywords        []string
	Redact_authorization *bool
}

type Memcache struct {
	ProtocolCommon        `yaml:",inline"`
	MaxValues             int
	MaxBytesPerValue      int
	UdpTransactionTimeout *int
	ParseUnknown          bool
}

type Mysql struct {
	ProtocolCommon `yaml:",inline"`
	Max_row_length *int
	Max_rows       *int
}

type Mongodb struct {
	ProtocolCommon `yaml:",inline"`
	Max_doc_length *int
	Max_docs       *int
}

type Pgsql struct {
	ProtocolCommon `yaml:",inline"`
	Max_row_length *int
	Max_rows       *int
}

type Thrift struct {
	ProtocolCommon             `yaml:",inline"`
	String_max_size            *int
	Collection_max_size        *int
	Drop_after_n_struct_fields *int
	Transport_type             *string
	Protocol_type              *string
	Capture_reply              *bool
	Obfuscate_strings          *bool
	Idl_files                  []string
}

type Redis struct {
	ProtocolCommon `yaml:",inline"`
}

// Config Singleton
var ConfigSingleton Config
