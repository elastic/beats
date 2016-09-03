package memcache

// Memcache plugin initialization, message/transaction types and transaction initialization/publishing.

import (
	"encoding/json"
	"expvar"
	"math"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"

	"github.com/elastic/beats/packetbeat/protos"
	"github.com/elastic/beats/packetbeat/protos/applayer"
	"github.com/elastic/beats/packetbeat/publish"
)

// memcache types
type Memcache struct {
	Ports   protos.PortsConfig
	results publish.Transactions
	config  parserConfig

	udpMemcache
	tcpMemcache

	handler MemcacheHandler
}

type MemcacheHandler interface {
	onTransaction(t *transaction)
}

// message actively parsed
type message struct {
	// shared
	applayer.Message
	next       *message
	isComplete bool

	command   *commandType
	isBinary  bool
	errorMsg  memcacheString
	casUnique uint64
	isCas     bool

	// text part
	commandLine memcacheString
	rawCommand  []byte
	rawArgs     []byte
	noreply     bool

	// binary part
	opcode  memcacheOpcode
	status  uint16
	vbucket uint16
	opaque  uint32
	isQuiet bool

	// values
	keys         []memcacheString
	flags        uint32
	exptime      uint32
	value        uint64
	value2       uint64
	ivalue       int64
	ivalue2      int64
	str          memcacheString
	data         memcacheData
	bytes        uint
	bytesLost    uint
	values       []memcacheData
	count_values uint32

	stats []memcacheStat
}

type transaction struct {
	applayer.Transaction

	command *commandType

	request  *message
	response *message
}

type memcacheString struct {
	raw []byte
}

type memcacheData struct {
	data []byte
}

type memcacheStat struct {
	Name  memcacheString `json:"name"`
	Value memcacheString `json:"value"`
}

var debug = logp.MakeDebug("memcache")

var (
	unmatchedRequests      = expvar.NewInt("memcache.unmatched_requests")
	unmatchedResponses     = expvar.NewInt("memcache.unmatched_responses")
	unfinishedTransactions = expvar.NewInt("memcache.unfinished_transactions")
)

func init() {
	protos.Register("memcache", New)
}

func New(
	testMode bool,
	results publish.Transactions,
	cfg *common.Config,
) (protos.Plugin, error) {
	p := &Memcache{}
	config := defaultConfig
	if !testMode {
		if err := cfg.Unpack(&config); err != nil {
			return nil, err
		}
	}

	if err := p.init(results, &config); err != nil {
		return nil, err
	}
	return p, nil
}

// Called to initialize the Plugin
func (mc *Memcache) init(results publish.Transactions, config *memcacheConfig) error {
	debug("init memcache plugin")

	mc.handler = mc
	if err := mc.setFromConfig(config); err != nil {
		return err
	}

	mc.udpConnections = make(map[common.HashableIpPortTuple]*udpConnection)
	mc.results = results
	return nil
}

func (mc *Memcache) setFromConfig(config *memcacheConfig) error {
	if err := mc.Ports.Set(config.Ports); err != nil {
		return err
	}

	mc.config.maxValues = config.MaxValues
	if config.MaxBytesPerValue <= 0 {
		mc.config.maxBytesPerValue = math.MaxInt32
	} else {
		mc.config.maxBytesPerValue = config.MaxBytesPerValue
	}

	mc.config.parseUnkown = config.ParseUnknown

	mc.udpConfig.transTimeout = config.UdpTransactionTimeout
	mc.tcpConfig.tcpTransTimeout = config.TransactionTimeout

	debug("transaction timeout: %v", config.TransactionTimeout)
	debug("udp transaction timeout: %v", config.UdpTransactionTimeout)
	debug("maxValues = %v", mc.config.maxValues)
	debug("maxBytesPerValue = %v", mc.config.maxBytesPerValue)

	return nil
}

// GetPorts return the configured memcache application ports.
func (mc *Memcache) GetPorts() []int {
	return mc.Ports.Ports
}

func (mc *Memcache) finishTransaction(t *transaction) error {
	mc.handler.onTransaction(t)
	return nil
}

func (mc *Memcache) onTransaction(t *transaction) {
	event := common.MapStr{}
	t.Event(event)
	debug("publish event: %s", event)
	mc.results.PublishTransaction(event)
}

func newMessage(ts time.Time) *message {
	msg := message{}
	msg.Ts = ts
	return &msg
}

func (m *message) String() string {
	return commandCodeStrings[m.command.code]
}

func (m *message) Event(event common.MapStr) error {
	if m.command == nil {
		return ErrInvalidMessage
	}
	return m.command.event(m, event)
}

func (m *message) SubEvent(
	name string,
	event common.MapStr,
) (common.MapStr, error) {
	if m == nil {
		return nil, nil
	}
	msg_event := common.MapStr{}
	event[name] = msg_event
	return msg_event, m.Event(msg_event)
}

func tryMergeResponses(mc *Memcache, prev, msg *message) (bool, error) {
	if msg != nil {
		msg.isComplete = checkResponseComplete(msg)
	}

	if prev == nil || msg == nil {
		return false, nil
	}

	if prev.isBinary != msg.isBinary {
		return false, ErrMixOfBinaryAndText
	}

	if !msg.isBinary {
		// merge text protocol value/stats message
		if prev.command.code == MemcacheResValue {
			return mergeValueMessages(mc, prev, msg)
		} else if prev.command.code == MemcacheResStat {
			return mergeStatsMessages(mc, prev, msg)
		}

		return false, nil
	} else {
		// merge binary protocol stats messages
		if prev.opcode != opcodeStat || msg.opcode != opcodeStat {
			return false, nil
		}
		if prev.opaque != msg.opaque {
			return false, nil
		}

		return mergeStatsMessages(mc, prev, msg)
	}
}

func mergeValueMessages(mc *Memcache, prev, msg *message) (bool, error) {
	debug("try to merge value messages")

	valueMessages := prev.command.code == MemcacheResValue &&
		(msg.command.code == MemcacheResValue ||
			msg.command.code == MemcacheResEnd)
	if !valueMessages {
		err := ErrExpectedValueForMerge
		debug("%v", err)
		return false, nil
	}

	prev.Size += msg.Size
	prev.bytes += msg.bytes
	prev.keys = append(prev.keys, msg.keys...)
	prev.AddNotes(msg.Notes...)
	prev.count_values += msg.count_values
	if msg.command.code == MemcacheResValue {
		delta := 0
		if mc.config.maxValues < 0 {
			delta = len(msg.values)
		} else if len(prev.values) < mc.config.maxValues {
			delta = mc.config.maxValues - len(prev.values)
			if delta > len(prev.values) {
				delta = len(prev.values)
			}
		}

		prev.values = append(prev.values, msg.values[0:delta]...)
	}

	prev.isComplete = prev.isComplete || msg.isComplete
	return true, nil
}

func mergeStatsMessages(mc *Memcache, prev, msg *message) (bool, error) {
	debug("try to merge stats message: %v", msg.stats)

	statsMessages := prev.command.typ == MemcacheStatsMsg &&
		(msg.command.typ == MemcacheStatsMsg ||
			msg.command.code == MemcacheResEnd)
	if !statsMessages {
		err := ErrExpectedStatsForMerge
		debug("%v", err)
		return false, nil
	}

	prev.AddNotes(msg.Notes...)
	prev.stats = append(prev.stats, msg.stats...)
	prev.Size += msg.Size
	prev.isComplete = prev.isComplete || msg.isComplete
	return true, nil
}

func checkResponseComplete(msg *message) bool {
	if msg.isBinary {
		if msg.opcode != opcodeStat {
			return true
		}
		return len(msg.keys) == 0
	} else {
		cont := msg.command.code == MemcacheResValue ||
			msg.command.code == MemcacheResStat
		return !cont
	}
}

func newTransaction(requ, resp *message) *transaction {

	if requ == nil && resp == nil {
		return nil
	}

	t := &transaction{}
	t.request = requ
	t.response = resp
	t.Status = computeTransactionStatus(requ, resp)

	switch {
	case requ != nil && resp != nil:
		t.Init(requ)
		t.BytesOut = requ.Size
		t.BytesIn = resp.Size
		t.ResponseTime = int32(resp.Ts.Sub(requ.Ts).Nanoseconds() / 1e6) // [ms]
		t.Notes = append(t.Notes, requ.Notes...)
		t.Notes = append(t.Notes, resp.Notes...)
	case requ != nil && resp == nil:
		t.Init(requ)
		t.BytesOut = requ.Size
		t.ResponseTime = -1
		t.Notes = append(t.Notes, requ.Notes...)
	case requ == nil && resp != nil:
		t.Init(resp)
		t.BytesIn = resp.Size
		t.ResponseTime = -1
		t.Notes = append(t.Notes, resp.Notes...)
	}

	return t
}

func (t *transaction) Init(msg *message) {
	t.Transaction.InitWithMsg("memcache", &msg.Message)
	t.command = msg.command
	if msg.IsRequest {
		t.BytesOut = msg.Size
	} else {
		t.BytesIn = msg.Size
	}
}

func (t *transaction) Event(event common.MapStr) error {
	debug("count event notes: %v", len(t.Notes))
	if err := t.Transaction.Event(event); err != nil {
		logp.Warn("error filling generic transaction fields: %v", err)
		return err
	}

	mc := common.MapStr{}
	event["memcache"] = mc

	if t.request != nil {
		_, err := t.request.SubEvent("request", mc)
		if err != nil {
			logp.Warn("error filling transaction request: %v", err)
			return err
		}
	}
	if t.response != nil {
		_, err := t.response.SubEvent("response", mc)
		if err != nil {
			logp.Warn("error filling transaction reponse: %v", err)
			return err
		}
	}

	msg := t.request
	if msg == nil {
		msg = t.response
	}
	if msg == nil {
		mc["protocol_type"] = "unknown"
	} else {
		if msg.isBinary {
			mc["protocol_type"] = "binary"
		} else {
			mc["protocol_type"] = "text"
		}
	}

	return nil
}

func computeTransactionStatus(requ, resp *message) string {
	switch {
	case requ == nil && resp != nil:
		return common.CLIENT_ERROR_STATUS
	case requ != nil && resp == nil:
		if requ.isQuiet || requ.noreply {
			return common.OK_STATUS
		} else if requ.command.code == MemcacheCmdQuit {
			return common.OK_STATUS
		} else {
			return common.SERVER_ERROR_STATUS
		}
	case requ != nil && resp != nil && requ.isBinary:
		if resp.status == uint16(statusCodeNoError) {
			return common.OK_STATUS
		} else if resp.status > 0x80 {
			return common.SERVER_ERROR_STATUS
		} else {
			return common.ERROR_STATUS
		}
	case requ != nil && resp != nil && !requ.isBinary:
		if resp.command.typ != MemcacheFailResp {
			return common.OK_STATUS
		}

		switch resp.command.code {
		case MemcacheErrClientError, MemcacheErrSame:
			return common.CLIENT_ERROR_STATUS
		case MemcacheErrServerError,
			MemcacheErrBusy,
			MemcacheErrBadClass,
			MemcacheErrNoSpare,
			MemcacheErrNotFull,
			MemcacheErrUnsafe:
			return common.SERVER_ERROR_STATUS
		default:
			return common.ERROR_STATUS
		}
	default:
		return common.ERROR_STATUS
	}
}

func (mc memcacheString) String() string {
	return string(mc.raw)
}

func (mc memcacheString) MarshalText() ([]byte, error) {
	return mc.raw, nil
}

func (mc memcacheData) String() string {
	return string(mc.data)
}

func (mc memcacheData) MarshalJSON() ([]byte, error) {
	return json.Marshal(mc.data)
}

func (mc memcacheData) IsSet() bool {
	return mc.data != nil
}
