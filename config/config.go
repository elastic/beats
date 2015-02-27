package config

import "github.com/BurntSushi/toml"

type Config struct {
	Interfaces InterfacesConfig
	Protocols  map[string]Protocol
	Output     map[string]MothershipConfig
	Input      Input
	RunOptions RunOptions
	Procs      Procs
	Agent      Agent
	Logging    Logging
	Passwords  Passwords
	Thrift     Thrift
	Http       Http
	Geoip      Geoip
	Udpjson    Udpjson
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

type RunOptions struct {
	Uid int
	Gid int
}

type Logging struct {
	Selectors []string
}

type Passwords struct {
	Hide_keywords       []string
	Strip_authorization bool
}

type Geoip struct {
	Paths []string
}

type Protocol struct {
	Ports         []int
	Send_request  bool
	Send_response bool
}

type Http struct {
	Send_all_headers bool
	Send_headers     []string
	Split_cookie     bool
	Real_ip_header   string
	Include_body_for []string
}

type Thrift struct {
	String_max_size            int
	Collection_max_size        int
	Drop_after_n_struct_fields int
	Transport_type             string
	Protocol_type              string
	Capture_reply              bool
	Obfuscate_strings          bool
	Idl_files                  []string
}

type Agent struct {
	Name                  string
	Refresh_topology_freq int
	Ignore_outgoing       bool
	Topology_expire       int
	Tags                  []string
}

type Procs struct {
	Dont_read_from_proc bool
	Max_proc_read_freq  int
	Monitored           map[string]Proc
	Refresh_pids_freq   int
}

type Proc struct {
	Cmdline_grep string
}

type MothershipConfig struct {
	Enabled            bool
	Save_topology      bool
	Host               string
	Port               int
	Protocol           string
	Username           string
	Password           string
	Index              string
	Path               string
	Db                 int
	Db_topology        int
	Timeout            int
	Reconnect_interval int
	Filename           string
	Rotate_every_kb    int
	Number_of_files    int
	DataType           string
	Flush_interval     int
}

type Udpjson struct {
	Bind_ip string
	Port    int
	Timeout int
}

// Config Singleton
var ConfigSingleton Config

// Config metadata singleton
var ConfigMeta toml.MetaData
