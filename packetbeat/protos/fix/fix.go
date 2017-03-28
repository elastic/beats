package fix

import (
    "time"
    "strings"
    "strconv"

    "github.com/elastic/beats/libbeat/common"
    "github.com/elastic/beats/libbeat/logp"

    //"github.com/elastic/beats/packetbeat/procs"
    "github.com/elastic/beats/packetbeat/protos"
    //"github.com/elastic/beats/packetbeat/protos/tcp"
    "github.com/elastic/beats/packetbeat/publish"
)

var debugf = logp.Info

type fixPlugin struct {
    // config
    ports        []int
    sendRequest  bool
    sendResponse bool

    transactionTimeout time.Duration

    results publish.Transactions
}

func init() {
    protos.Register("fix", New)
}

func New(testMode bool, results publish.Transactions, cfg *common.Config) (protos.Plugin, error) {
    p := &fixPlugin{}
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

func (fix *fixPlugin) init(results publish.Transactions, config *fixConfig) error {
    fix.setFromConfig(config)

    fix.results = results

    return nil
}

func (fix *fixPlugin) setFromConfig(config *fixConfig) {
    fix.ports = config.Ports
    fix.sendRequest = config.SendRequest
    fix.sendResponse = config.SendResponse
    fix.transactionTimeout = config.TransactionTimeout
}

func (fix *fixPlugin) GetPorts() []int {
    return fix.ports
}


func (fix *fixPlugin) ConnectionTimeout() time.Duration {
    return fix.transactionTimeout
}

func (fix *fixPlugin) Parse(pkt *protos.Packet, tcptuple *common.TCPTuple,
    dir uint8, private protos.ProtocolData) protos.ProtocolData {

    parts := strings.Split(string(pkt.Payload), string(rune(01)))

    parts = parts[:len(parts)-1]

    debugf("stream add data: %q (dir=%v, len=%v) {", parts, dir, len(pkt.Payload))

    event := common.MapStr{}

    event["@timestamp"] = common.Time(pkt.Ts)

    event["type"] = "fix"

    for _, part := range parts {
        q := strings.Split(part, "=")
        if len(q) > 1 {
            key, _ := strconv.Atoi(q[0])
            value := q[1]

            field := fixFields[key]


            if field.dtype == "string" {
                event[field.name] = value
            }
            if field.dtype == "int" {
                castVal, _ := strconv.Atoi(value)
                event[field.name] = castVal
            }
            if field.dtype == "float" {
                castVal, _ := strconv.ParseFloat(value, 64)
                event[field.name] = castVal
            }

        }
    }

    fix.results.PublishTransaction(event)

    return nil
}

func (fix *fixPlugin) GapInStream(tcptuple *common.TCPTuple, dir uint8,
    nbytes int, private protos.ProtocolData) (priv protos.ProtocolData, drop bool) {

    return private, true
}

func (fix *fixPlugin) ReceivedFin(tcptuple *common.TCPTuple, dir uint8,
    private protos.ProtocolData) protos.ProtocolData {

    return private
}

