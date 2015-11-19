package outputs

import (
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
)

type MothershipConfig struct {
	Save_topology      bool
	Host               string
	Port               int
	Hosts              []string
	LoadBalance        *bool
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
	BulkMaxSize        *int `yaml:"bulk_max_size"`
	Max_retries        *int
	Pretty             *bool
	TLS                *TLSConfig
	Worker             int
}

type Outputer interface {
	// Publish event
	PublishEvent(trans Signaler, ts time.Time, event common.MapStr) error
}

type TopologyOutputer interface {
	// Register the agent name and its IPs to the topology map
	PublishIPs(name string, localAddrs []string) error

	// Get the agent name with a specific IP from the topology map
	GetNameByIP(ip string) string
}

// BulkOutputer adds BulkPublish to publish batches of events without looping.
// Outputers still might loop on events or use more efficient bulk-apis if present.
type BulkOutputer interface {
	Outputer
	BulkPublish(trans Signaler, ts time.Time, event []common.MapStr) error
}

type OutputBuilder interface {
	// Create and initialize the output plugin
	NewOutput(
		beat string,
		config *MothershipConfig,
		topologyExpire int) (Outputer, error)
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

type bulkOutputAdapter struct {
	Outputer
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
		if !exists {
			continue
		}

		output, err := plugin.NewOutput(beat, &config, topologyExpire)
		if err != nil {
			logp.Err("failed to initialize %s plugin as output: %s", name, err)
			return nil, err
		}

		plugin := OutputPlugin{Name: name, Config: config, Output: output}
		plugins = append(plugins, plugin)
		logp.Info("Activated %s as output plugin.", name)
	}
	return plugins, nil
}

// CastBulkOutputer casts out into a BulkOutputer if out implements
// the BulkOutputer interface. If out does not implement the interface an outputer
// wrapper implementing the BulkOutputer interface is returned.
func CastBulkOutputer(out Outputer) BulkOutputer {
	if bo, ok := out.(BulkOutputer); ok {
		return bo
	}
	return &bulkOutputAdapter{out}
}

func (b *bulkOutputAdapter) BulkPublish(
	signal Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	signal = NewSplitSignaler(signal, len(events))
	for _, evt := range events {
		err := b.PublishEvent(signal, ts, evt)
		if err != nil {
			return err
		}
	}
	return nil
}
