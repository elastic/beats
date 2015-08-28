package outputs

import (
	"time"

	"github.com/elastic/libbeat/common"
)

type MothershipConfig struct {
	Enabled            bool
	Save_topology      bool
	Host               string
	Port               int
	Hosts              []string
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
	Flush_interval     *int
	Bulk_size          *int
	Max_retries        *int
}

// Functions to be exported by a output plugin
type OutputInterface interface {
	// Initialize the output plugin
	Init(beat string, config MothershipConfig, topology_expire int) error

	// Register the agent name and its IPs to the topology map
	PublishIPs(name string, localAddrs []string) error

	// Get the agent name with a specific IP from the topology map
	GetNameByIP(ip string) string

	// Publish event
	PublishEvent(ts time.Time, event common.MapStr) error
}

// Output identifier
type OutputPlugin uint16

// Output constants
const (
	UnknownOutput OutputPlugin = iota
	RedisOutput
	ElasticsearchOutput
	FileOutput
)

// Output names
var OutputNames = []string{
	"unknown",
	"redis",
	"elasticsearch",
	"file",
}

func (o OutputPlugin) String() string {
	if int(o) >= len(OutputNames) {
		return "impossible"
	}
	return OutputNames[o]
}
