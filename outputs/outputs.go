package outputs

import (
	"sync/atomic"
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
	PublishEvent(trans Transactioner, ts time.Time, event common.MapStr) error
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
	BulkPublish(trans Transactioner, ts time.Time, event []common.MapStr) error
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

type Transactioner interface {
	// Completed is called by publish/output plugin when all events have been
	// send
	Completed()
	Failed()
}

type OutputPlugin struct {
	Name   string
	Config MothershipConfig
	Output Outputer
}

// MultiOutputTransaction guards one transaction from multiple calls
// by using a simple reference counting scheme. If one Transactioner consumer
// reports a Failed event, the Failed event will be send to the guarded Transactioner
// once the reference count becomes zero.
//
// Example use cases:
//   - Push transaction to multiple outputers
//   - split data to be send into smaller transactions
type MultiOutputTransaction struct {
	count       int32
	failed      bool
	transaction Transactioner
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
	trans Transactioner,
	ts time.Time,
	events []common.MapStr,
) error {
	trans = NewMultiOutputTransaction(trans, len(events))
	for _, evt := range events {
		err := b.PublishEvent(trans, ts, evt)
		if err != nil {
			return err
		}
	}
	return nil
}

// NewMultiOutputTransaction create a new MultiOutputTransaction if trans is not nil.
// If trans is nil, nil will be returned. The count is the number of events to be
// received before publishing the final event to the guarded Transactioner.
func NewMultiOutputTransaction(
	trans Transactioner,
	count int,
) *MultiOutputTransaction {
	if trans == nil {
		return nil
	}

	return &MultiOutputTransaction{
		count:       int32(count),
		transaction: trans,
	}
}

// Completed signals a Completed event to m.
func (m *MultiOutputTransaction) Completed() {
	m.onEvent()
}

// Failed signals a Failed event to m.
func (m *MultiOutputTransaction) Failed() {
	m.failed = true
	m.onEvent()
}

func (m *MultiOutputTransaction) onEvent() {
	res := atomic.AddInt32(&m.count, -1)
	if res == 0 {
		if m.failed {
			m.transaction.Failed()
		} else {
			m.transaction.Completed()
		}
	}
}

// CompleteTransaction sends the Completed event to trans if trans is not nil.
func CompleteTransaction(trans Transactioner) {
	if trans != nil {
		trans.Completed()
	}
}

// FailTransaction sends the Failed event to trans if trans is not nil
func FailTransaction(trans Transactioner) {
	if trans != nil {
		trans.Failed()
	}
}

// FinishTransaction will send the Completed or Failed event to trans depending
// on err being set if trans is not nil.
func FinishTransaction(trans Transactioner, err error) {
	if trans != nil {
		if err == nil {
			trans.Completed()
		} else {
			trans.Failed()
		}
	}
}

// FinishTransactions send the Completed or Failed event to all given transactions
// depending on err being set.
func FinishTransactions(transactions []Transactioner, err error) {
	if err == nil {
		for _, t := range transactions {
			t.Completed()
		}
	} else {
		for _, t := range transactions {
			t.Failed()
		}
	}
}
