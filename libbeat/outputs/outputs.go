package outputs

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
)

type MothershipConfig struct {
	SaveTopology      bool `config:"save_topology"`
	Host              string
	Port              int
	Hosts             []string
	LoadBalance       *bool `config:"loadbalance"`
	Protocol          string
	Username          string
	Password          string
	ProxyURL          string `config:"proxy_url"`
	Index             string
	Path              string
	Template          Template
	Params            map[string]string `config:"parameters"`
	Db                int
	DbTopology        int `config:"db_topology"`
	Timeout           int
	ReconnectInterval int    `config:"reconnect_interval"`
	Filename          string `config:"filename"`
	RotateEveryKb     int    `config:"rotate_every_kb"`
	NumberOfFiles     int    `config:"number_of_files"`
	DataType          string
	FlushInterval     *int  `config:"flush_interval"`
	BulkMaxSize       *int  `config:"bulk_max_size"`
	MaxRetries        *int  `config:"max_retries"`
	Pretty            *bool `config:"pretty"`
	TLS               *TLSConfig
	Worker            int
	CompressionLevel  *int   `config:"compression_level"`
	KeepAlive         string `config:"keep_alive"`
	MaxMessageBytes   *int   `config:"max_message_bytes"`
	RequiredACKs      *int   `config:"required_acks"`
	BrokerTimeout     string `config:"broker_timeout"`
	Compression       string `config:"compression"`
	ClientID          string `config:"client_id"`
	Topic             string `config:"topic"`
	UseType           *bool  `config:"use_type"`
}

type Template struct {
	Name      string
	Path      string
	Overwrite bool
}

type Options struct {
	Guaranteed bool
}

type Outputer interface {
	// Publish event

	PublishEvent(trans Signaler, opts Options, event common.MapStr) error
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
	BulkPublish(trans Signaler, opts Options, event []common.MapStr) error
}

type OutputBuilder interface {
	// Create and initialize the output plugin
	NewOutput(
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
	beatName string,
	configs map[string]MothershipConfig,
	topologyExpire int,
) ([]OutputPlugin, error) {
	var plugins []OutputPlugin = nil
	for name, plugin := range enabledOutputPlugins {
		config, exists := configs[name]
		if !exists {
			continue
		}

		if config.Index == "" {
			config.Index = beatName
		}

		output, err := plugin.NewOutput(&config, topologyExpire)
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
	opts Options,
	events []common.MapStr,
) error {
	signal = NewSplitSignaler(signal, len(events))
	for _, evt := range events {
		err := b.PublishEvent(signal, opts, evt)
		if err != nil {
			return err
		}
	}
	return nil
}
