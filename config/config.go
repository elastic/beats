package config

import (
	"github.com/elastic/infrabeat/common/droppriv"
	"github.com/elastic/infrabeat/outputs"
	"github.com/elastic/packetbeat/procs"
)

type Config struct {
	Interfaces InterfacesConfig
	Protocols  map[string]Protocol
	Output     map[string]outputs.MothershipConfig
	Agent      outputs.AgentConfig
	Input      Input
	Procs      procs.ProcsConfig
	RunOptions droppriv.RunOptions
	Logging    Logging
	Thrift     Thrift
	Http       Http
	Mysql      Mysql
	Pgsql      Pgsql
	Redis      Redis
	Geoip      outputs.Geoip
	Udpjson    Udpjson
	GoBeacon   GoBeacon
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

type Input struct {
	Inputs []string
}

type Logging struct {
	Selectors []string
}

type Protocol struct {
	Protocol string
	Ports    []int
}

type Http struct {
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
	Max_row_length *int
	Max_rows       *int
	Send_request   *bool
	Send_response  *bool
}

type Pgsql struct {
	Max_row_length *int
	Max_rows       *int
	Send_request   *bool
	Send_response  *bool
}

type Thrift struct {
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
	Send_request  *bool
	Send_response *bool
}

type Udpjson struct {
	Bind_ip string
	Port    int
	Timeout int
}

type GoBeacon struct {
	Listen_addr string
	Tracker     string
}

// Config Singleton
var ConfigSingleton Config
