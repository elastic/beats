package outputs

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/op"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/monitoring"
)

type Options struct {
	Guaranteed bool
}

// Data contains the Event and additional values shared/populated by outputs
// to share state internally in output plugins for example between retries.
//
// Values of type Data are pushed by value inside the publisher chain up to the
// outputs. If multiple outputs are configured, each will receive a copy of Data
// elemets.
type Data struct {
	// Holds the beats published event and MUST be used read-only manner only in
	// output plugins.
	Event common.MapStr

	// `Values` can be used to store additional context-dependent metadata
	// within Data. With `Data` being copied to each output, it is safe to update
	// `Data.Values` itself in outputs, but access to actually stored values must
	// be thread-safe: read-only if key might be shared or read/write if value key
	// is local to output plugin.
	Values *Values
}

type Outputer interface {
	// Publish event
	PublishEvent(sig op.Signaler, opts Options, data Data) error

	Close() error
}

// BulkOutputer adds BulkPublish to publish batches of events without looping.
// Outputers still might loop on events or use more efficient bulk-apis if present.
type BulkOutputer interface {
	Outputer
	BulkPublish(sig op.Signaler, opts Options, data []Data) error
}

// Create and initialize the output plugin
type OutputBuilder func(beat common.BeatInfo, config *common.Config) (Outputer, error)

// Functions to be exported by a output plugin
type OutputInterface interface {
	Outputer
}

type OutputPlugin struct {
	Name   string
	Config *common.Config
	Output Outputer
}

type bulkOutputAdapter struct {
	Outputer
}

var outputsPlugins = make(map[string]OutputBuilder)

var (
	Metrics = monitoring.Default.NewRegistry("output")

	AckedEvents = monitoring.NewInt(Metrics, "events.acked", monitoring.Report)
	WriteBytes  = monitoring.NewInt(Metrics, "write.bytes", monitoring.Report)
	WriteErrors = monitoring.NewInt(Metrics, "write.errors", monitoring.Report)
)

func RegisterOutputPlugin(name string, builder OutputBuilder) {
	outputsPlugins[name] = builder
}

func FindOutputPlugin(name string) OutputBuilder {
	return outputsPlugins[name]
}

func InitOutputs(
	beat common.BeatInfo,
	configs map[string]*common.Config,
) ([]OutputPlugin, error) {
	var plugins []OutputPlugin
	for name, plugin := range outputsPlugins {
		config, exists := configs[name]
		if !exists {
			continue
		}

		config.PrintDebugf("Configure output plugin '%v' with:", name)
		if !config.Enabled() {
			continue
		}

		output, err := plugin(beat, config)
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
	signal op.Signaler,
	opts Options,
	data []Data,
) error {
	signal = op.SplitSignaler(signal, len(data))
	for _, d := range data {
		err := b.PublishEvent(signal, opts, d)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *Data) AddValue(key, value interface{}) {
	d.Values = d.Values.Append(key, value)
}

type EventEncoder interface {
	// Encode event
	Encode(event common.MapStr, options interface{}) ([]byte, error)
}

type EventFormatter interface {
	// Format event
	Format(event common.MapStr, format string) ([]byte, error)
}
