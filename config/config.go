package config

import (
	"github.com/elastic/libbeat/common/droppriv"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/publisher"
	"github.com/elastic/packetbeat/procs"
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
	Dns      Dns
	Http     Http
	Memcache Memcache
	Mysql    Mysql
	Mongodb  Mongodb
	Pgsql    Pgsql
	Redis    Redis
	Thrift   Thrift
}

type Dns struct {
	Ports               []int
	Send_request        *bool
	Send_response       *bool
	Include_authorities *bool
	Include_additionals *bool
}

type Http struct {
	Ports                []int
	Send_all_headers     *bool
	Send_headers         []string
	Split_cookie         *bool
	Real_ip_header       *string
	Include_body_for     []string
	Hide_keywords        []string
	Redact_authorization *bool
	Send_request         *bool
	Send_response        *bool
}

type Memcache struct {
	Ports                 []int
	MaxValues             int
	MaxBytesPerValue      int
	UdpTransactionTimeout uint
	TcpTransactionTimeout uint
	ParseUnknown          bool
}

type Mysql struct {
	Ports          []int
	Max_row_length *int
	Max_rows       *int
	Send_request   *bool
	Send_response  *bool
}

type Mongodb struct {
	Ports          []int
	Max_doc_length *int
	Max_docs       *int
	Send_request   *bool
	Send_response  *bool
}

type Pgsql struct {
	Ports          []int
	Max_row_length *int
	Max_rows       *int
	Send_request   *bool
	Send_response  *bool
}

type Thrift struct {
	Ports                      []int
	String_max_size            *int
	Collection_max_size        *int
	Drop_after_n_struct_fields *int
	Transport_type             *string
	Protocol_type              *string
	Capture_reply              *bool
	Obfuscate_strings          *bool
	Idl_files                  []string
	Send_request               *bool
	Send_response              *bool
}

type Redis struct {
	Ports         []int
	Send_request  *bool
	Send_response *bool
}

// Config Singleton
var ConfigSingleton Config
