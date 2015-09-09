package outputs

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
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
	TLS                *bool
	Certificate        string
	CertificateKey     string
	CAs                []string
}

type Outputer interface {
	// Publish event
	PublishEvent(ts time.Time, event common.MapStr) error
}

type TopologyOutputer interface {
	// Register the agent name and its IPs to the topology map
	PublishIPs(name string, localAddrs []string) error

	// Get the agent name with a specific IP from the topology map
	GetNameByIP(ip string) string
}

type OutputBuilder interface {
	// Create and initialize the output plugin
	NewOutput(
		beat string,
		config MothershipConfig,
		topology_expire int) (Outputer, error)
}

// Functions to be exported by a output plugin
type OutputInterface interface {
	Outputer
	TopologyOutputer
}

type OutputPlugin struct {
	Name   string
	Config MothershipConfig
	Output Outputer
}

var enabledOutputPlugins = make(map[string]OutputBuilder)

func RegisterOutputPlugin(name string, builder OutputBuilder) {
	enabledOutputPlugins[name] = builder
}

func FindOutputPlugin(name string) OutputBuilder {
	return enabledOutputPlugins[name]
}

func InitOutputs(
	beat string,
	configs map[string]MothershipConfig,
	topologyExpire int,
) ([]OutputPlugin, error) {
	var plugins []OutputPlugin = nil
	for name, plugin := range enabledOutputPlugins {
		config, exists := configs[name]
		if !exists || !config.Enabled {
			continue
		}

		output, err := plugin.NewOutput(beat, config, topologyExpire)
		if err != nil {
			logp.Err("failed to initialize %s plugin as output: %s", name, err)
			return nil, err
		}

		plugin := OutputPlugin{Name: name, Config: config, Output: output}
		plugins = append(plugins, plugin)
	}
	return plugins, nil
}
