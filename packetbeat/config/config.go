package config

import (
	"github.com/elastic/beats/libbeat/common/droppriv"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/libbeat/publisher"
	"github.com/elastic/beats/packetbeat/procs"
)

// Config is a composed structure of Packetbeat configuration
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

// InterfacesConfig contains the information about the active machine's
// interfaces
type InterfacesConfig struct {
	Device       string
	Type         string
	File         string
	WithVlans    bool
	BpfFilter    string
	Snaplen      int
	BufferSizeMb int
	TopSpeed     bool
	Dumpfile     string
	OneAtATime   bool
	Loop         int
}

// Protocols holds the different supported Packetbeat protocol information
type Protocols struct {
	Icmp     Icmp
	DNS      DNS
	HTTP     HTTP
	Memcache Memcache
	Mysql    Mysql
	Mongodb  Mongodb
	Pgsql    Pgsql
	Redis    Redis
	Thrift   Thrift
}

// ProtocolCommon are the common information associated with a protocol
type ProtocolCommon struct {
	Ports              []int `yaml:"ports"`
	SendRequest        *bool `yaml:"send_request"`
	SendResponse       *bool `yaml:"send_response"`
	TransactionTimeout *int  `yaml:"transaction_timeout"`
}

// Icmp holds ICMPv4/v6 specific configuration informaiton
type Icmp struct {
	Enabled            bool
	SendRequest        *bool `yaml:"send_request"`
	SendResponse       *bool `yaml:"send_response"`
	TransactionTimeout *int  `yaml:"transaction_timeout"`
}

// DNS holds DNS specific configuration information
type DNS struct {
	ProtocolCommon     `yaml:",inline"`
	IncludeAuthorities *bool
	IncludeAdditionals *bool
}

// HTTP holds HTTP specific configuration informaiton
type HTTP struct {
	ProtocolCommon      `yaml:",inline"`
	SendAllHeaders      *bool
	SendHeaders         []string
	SplitCookie         *bool
	RealIPHeader        *string
	IncludeBodyFor      []string
	HideKeywords        []string
	RedactAuthorization *bool
}

// Memcache holds Memcached specific configuration informaiton
type Memcache struct {
	ProtocolCommon        `yaml:",inline"`
	MaxValues             int
	MaxBytesPerValue      int
	UDPTransactionTimeout *int
	ParseUnknown          bool
}

// Mysql holds MySQL specific configuration informaiton
type Mysql struct {
	ProtocolCommon `yaml:",inline"`
	MaxRowLength   *int
	MaxRows        *int
}

// Mongodb holds MongoDB specific configuration informaiton
type Mongodb struct {
	ProtocolCommon `yaml:",inline"`
	MaxDocLength   *int
	MaxDocs        *int
}

// Pgsql holds PostgreSQL specific configuration informaiton
type Pgsql struct {
	ProtocolCommon `yaml:",inline"`
	MaxRowLength   *int
	MaxRows        *int
}

// Thrift holds Apache Thrift specific configuration informaiton
type Thrift struct {
	ProtocolCommon         `yaml:",inline"`
	StringMaxSize          *int
	CollectionMaxSize      *int
	DropAfterNStructFields *int
	TransportType          *string
	ProtocolType           *string
	CaptureReply           *bool
	ObfuscateStrings       *bool
	IdlFiles               []string
}

// Redis holds Redis specific configuration informaiton
type Redis struct {
	ProtocolCommon `yaml:",inline"`
}

// ConfigSingleton holds the configuration as a singleton object
var ConfigSingleton Config
