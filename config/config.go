package config

import (
	"github.com/elastic/libbeat/common/droppriv"
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
	Logging    Logging
	Filter     map[string]interface{}
}

type InterfacesConfig struct {
	Device         string
	Devices        []string
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

type Logging struct {
	Selectors []string
}

type Protocols struct {
	Http   Http
	Mysql  Mysql
	Pgsql  Pgsql
	Redis  Redis
	Thrift Thrift
}

type Http struct {
	Ports               []int
	Send_all_headers    *bool
	Send_headers        []string
	Split_cookie        *bool
	Real_ip_header      *string
	Include_body_for    []string
	Hide_keywords       []string
	Strip_authorization *bool
	Send_request        *bool
	Send_response       *bool
}

type Mysql struct {
	Ports          []int
	Max_row_length *int
	Max_rows       *int
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
