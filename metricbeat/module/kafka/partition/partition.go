package partition

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/mb/parse"

	"github.com/Shopify/sarama"
)

// init registers the partition MetricSet with the central registry.
func init() {
	if err := mb.Registry.AddMetricSet("kafka", "partition", New, parse.PassThruHostParser); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the partition MetricSet
type MetricSet struct {
	mb.BaseMetricSet

	broker *sarama.Broker
	cfg    *sarama.Config
	id     int32
	topics []string
}

const noID int32 = -1

var errFailQueryOffset = errors.New("operation failed")

var debugf = logp.MakeDebug("kafka")

// New create a new instance of the partition MetricSet
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	config := defaultConfig
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	tls, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = base.Module().Config().Timeout
	cfg.Net.ReadTimeout = base.Module().Config().Timeout
	cfg.ClientID = config.ClientID
	cfg.Metadata.Retry.Max = config.Retries
	cfg.Metadata.Retry.Backoff = config.Backoff
	if tls != nil {
		cfg.Net.TLS.Enable = true
		cfg.Net.TLS.Config = tls.BuildModuleConfig("")
	}
	if config.Username != "" {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.User = config.Username
		cfg.Net.SASL.Password = config.Password
	}

	broker := sarama.NewBroker(base.Host())
	return &MetricSet{
		BaseMetricSet: base,
		broker:        broker,
		cfg:           cfg,
		id:            noID,
		topics:        config.Topics,
	}, nil
}

func (m *MetricSet) connect() (*sarama.Broker, error) {
	b := m.broker
	if err := b.Open(m.cfg); err != nil {
		return nil, err
	}

	if m.id != noID {
		return b, nil
	}

	// current broker is bootstrap only. Get metadata to find id:
	meta, err := queryMetadataWithRetry(b, m.cfg, m.topics)
	if err != nil {
		closeBroker(b)
		return nil, err
	}

	other := findMatchingBroker(brokerAddress(b), meta.Brokers)
	if other == nil { // no broker found
		closeBroker(b)
		return nil, fmt.Errorf("No advertised broker with address %v found", b.Addr())
	}

	debugf("found matching broker %v with id %v", other.Addr(), other.ID())

	m.id = other.ID()
	return b, nil
}

// Fetch partition stats list from kafka
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	b, err := m.connect()
	if err != nil {
		return nil, err
	}

	defer closeBroker(b)
	response, err := queryMetadataWithRetry(b, m.cfg, m.topics)
	if err != nil {
		return nil, err
	}

	events := []common.MapStr{}
	evtBroker := common.MapStr{
		"id":      m.id,
		"address": b.Addr(),
	}

	for _, topic := range response.Topics {
		evtTopic := common.MapStr{
			"name": topic.Name,
		}

		if topic.Err != 0 {
			evtTopic["error"] = common.MapStr{
				"code": topic.Err,
			}
		}

		for _, partition := range topic.Partitions {
			// partition offsets can be queried from leader only
			if m.id != partition.Leader {
				continue
			}

			// collect offsets for all replicas
			for _, id := range partition.Replicas {

				// Get oldest and newest available offsets
				offOldest, offNewest, offOK, err := queryOffsetRange(b, id, topic.Name, partition.ID)

				if !offOK {
					if err == nil {
						err = errFailQueryOffset
					}

					logp.Err("Failed to query kafka partition (%v:%v) offsets: %v",
						topic.Name, partition.ID, err)
					continue
				}

				partitionEvent := common.MapStr{
					"id":             partition.ID,
					"leader":         partition.Leader,
					"replica":        id,
					"insync_replica": hasID(id, partition.Isr),
				}

				if partition.Err != 0 {
					partitionEvent["error"] = common.MapStr{
						"code": partition.Err,
					}
				}

				// create event
				event := common.MapStr{
					"topic":     evtTopic,
					"broker":    evtBroker,
					"partition": partitionEvent,
					"offset": common.MapStr{
						"newest": offNewest,
						"oldest": offOldest,
					},
				}

				events = append(events, event)
			}
		}
	}

	return events, nil
}

func hasID(id int32, lst []int32) bool {
	for _, other := range lst {
		if id == other {
			return true
		}
	}
	return false
}

// queryOffsetRange queries the broker for the oldest and the newest offsets in
// a kafka topics partition for a given replica.
func queryOffsetRange(
	b *sarama.Broker,
	replicaID int32,
	topic string,
	partition int32,
) (int64, int64, bool, error) {
	oldest, okOld, err := queryOffset(b, replicaID, topic, partition, sarama.OffsetOldest)
	if err != nil {
		return -1, -1, false, err
	}

	newest, okNew, err := queryOffset(b, replicaID, topic, partition, sarama.OffsetNewest)
	if err != nil {
		return -1, -1, false, err
	}

	return oldest, newest, okOld && okNew, nil
}

func queryOffset(
	b *sarama.Broker,
	replicaID int32,
	topic string,
	partition int32,
	time int64,
) (int64, bool, error) {
	req := &sarama.OffsetRequest{}
	if replicaID != noID {
		req.SetReplicaID(replicaID)
	}
	req.AddBlock(topic, partition, time, 1)
	resp, err := b.GetAvailableOffsets(req)
	if err != nil {
		return -1, false, err
	}

	block := resp.GetBlock(topic, partition)
	if len(block.Offsets) == 0 {
		return -1, false, nil
	}

	return block.Offsets[0], true, nil
}

func closeBroker(b *sarama.Broker) {
	if ok, _ := b.Connected(); ok {
		b.Close()
	}
}

func queryMetadataWithRetry(
	b *sarama.Broker,
	cfg *sarama.Config,
	topics []string,
) (r *sarama.MetadataResponse, err error) {
	err = withRetry(b, cfg, func() (e error) {
		r, e = b.GetMetadata(&sarama.MetadataRequest{topics})
		return
	})
	return
}

func withRetry(
	b *sarama.Broker,
	cfg *sarama.Config,
	f func() error,
) error {
	var err error
	for max := 0; max < cfg.Metadata.Retry.Max; max++ {
		if ok, _ := b.Connected(); !ok {
			if err = b.Open(cfg); err == nil {
				err = f()
			}
		} else {
			err = f()
		}

		if err == nil {
			return nil
		}

		retry, reconnect := checkRetryQuery(err)
		if !retry {
			return err
		}

		time.Sleep(cfg.Metadata.Retry.Backoff)
		if reconnect {
			closeBroker(b)
		}
	}
	return err
}

func checkRetryQuery(err error) (retry, reconnect bool) {
	if err == nil {
		return false, false
	}

	if err == io.EOF {
		return true, true
	}

	k, ok := err.(sarama.KError)
	if !ok {
		return false, false
	}

	switch k {
	case sarama.ErrLeaderNotAvailable, sarama.ErrReplicaNotAvailable,
		sarama.ErrOffsetsLoadInProgress, sarama.ErrRebalanceInProgress:
		return true, false
	case sarama.ErrRequestTimedOut, sarama.ErrBrokerNotAvailable,
		sarama.ErrNetworkException:
		return true, true
	}

	return false, false
}

func findMatchingBroker(
	addr string,
	brokers []*sarama.Broker,
) *sarama.Broker {
	lst := brokerAddresses(brokers)
	if idx, found := findMatchingAddress(addr, lst); found {
		return brokers[idx]
	}
	return nil
}

func findMatchingAddress(
	addr string,
	brokers []string,
) (int, bool) {
	debugf("Try to match broker to: %v", addr)

	// compare connection address to list of broker addresses
	if i, found := indexOf(addr, brokers); found {
		return i, true
	}

	// get connection 'port'
	_, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		port = "9092"
	}

	// lookup local machines ips for comparing with broker addresses
	localIPs, err := common.LocalIPAddrs()
	if err != nil || len(localIPs) == 0 {
		return -1, false
	}
	debugf("local machine ips: %v", localIPs)

	// try to find broker by comparing the fqdn for each known ip to list of
	// brokers
	localHosts := lookupHosts(localIPs)
	debugf("local machine addresses: %v", localHosts)
	for _, host := range localHosts {
		debugf("try to match with fqdn: %v (%v)", host, port)
		if i, found := indexOf(net.JoinHostPort(host, port), brokers); found {
			return i, true
		}
	}

	// try to find broker id by comparing the machines local hostname to
	// broker hostnames in metadata
	if host, err := os.Hostname(); err == nil {
		debugf("try to match with hostname only: %v (%v)", host, port)

		tmp := net.JoinHostPort(strings.ToLower(host), port)
		if i, found := indexOf(tmp, brokers); found {
			return i, true
		}
	}

	// lookup ips for all brokers
	debugf("match by ips")
	for i, b := range brokers {
		debugf("test broker address: %v", b)
		bh, bp, err := net.SplitHostPort(b)
		if err != nil {
			continue
		}

		// port numbers do not match
		if bp != port {
			continue
		}

		// lookup all ips for brokers host:
		ips, err := net.LookupIP(bh)
		debugf("broker %v ips: %v, %v", bh, ips, err)
		if err != nil {
			continue
		}

		debugf("broker (%v) ips: %v", bh, ips)

		// check if ip is known
		if anyIPsMatch(ips, localIPs) {
			return i, true
		}
	}

	return -1, false
}

func lookupHosts(ips []net.IP) []string {
	set := map[string]struct{}{}
	for _, ip := range ips {
		txt, err := ip.MarshalText()
		if err != nil {
			continue
		}

		hosts, err := net.LookupAddr(string(txt))
		debugf("lookup %v => %v, %v", string(txt), hosts, err)
		if err != nil {
			continue
		}

		for _, host := range hosts {
			h := strings.ToLower(strings.TrimSuffix(host, "."))
			set[h] = struct{}{}
		}
	}

	hosts := make([]string, 0, len(set))
	for host := range set {
		hosts = append(hosts, host)
	}
	return hosts
}

func anyIPsMatch(as, bs []net.IP) bool {
	for _, a := range as {
		for _, b := range bs {
			if bytes.Equal(a, b) {
				return true
			}
		}
	}
	return false
}

func brokerAddresses(brokers []*sarama.Broker) []string {
	addresses := make([]string, len(brokers))
	for i, b := range brokers {
		addresses[i] = brokerAddress(b)
	}
	return addresses
}

func brokerAddress(b *sarama.Broker) string {
	return strings.ToLower(b.Addr())
}

func indexOf(s string, lst []string) (int, bool) {
	for i, v := range lst {
		if s == v {
			return i, true
		}
	}
	return -1, false
}
