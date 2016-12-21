package kafka

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/Shopify/sarama"
	"github.com/elastic/beats/libbeat/common"
)

// Broker provides functionality for communicating with a single kafka broker
type Broker struct {
	b   *sarama.Broker
	cfg *sarama.Config

	id      int32
	matchID bool
}

// BrokerSettings defines common configurations used when connecting to a broker
type BrokerSettings struct {
	MatchID                  bool
	DialTimeout, ReadTimeout time.Duration
	ClientID                 string
	Retries                  int
	Backoff                  time.Duration
	TLS                      *tls.Config
	Username, Password       string
}

const noID = -1

// NewBroker creates a new unconnected kafka Broker connection instance.
func NewBroker(host string, settings BrokerSettings) *Broker {
	cfg := sarama.NewConfig()
	cfg.Net.DialTimeout = settings.DialTimeout
	cfg.Net.ReadTimeout = settings.ReadTimeout
	cfg.ClientID = settings.ClientID
	cfg.Metadata.Retry.Max = settings.Retries
	cfg.Metadata.Retry.Backoff = settings.Backoff
	if tls := settings.TLS; tls != nil {
		cfg.Net.TLS.Enable = true
		cfg.Net.TLS.Config = tls
	}
	if user := settings.Username; user != "" {
		cfg.Net.SASL.Enable = true
		cfg.Net.SASL.User = user
		cfg.Net.SASL.Password = settings.Password
	}

	return &Broker{
		b:       sarama.NewBroker(host),
		cfg:     cfg,
		id:      noID,
		matchID: settings.MatchID,
	}
}

// Close the broker connection
func (b *Broker) Close() error {
	closeBroker(b.b)
	return nil
}

// Connect connects the broker to the configured host
func (b *Broker) Connect() error {
	if err := b.b.Open(b.cfg); err != nil {
		return err
	}

	if b.id != noID || !b.matchID {
		return nil
	}

	// current broker is bootstrap only. Get metadata to find id:
	meta, err := queryMetadataWithRetry(b.b, b.cfg, nil)
	if err != nil {
		closeBroker(b.b)
		return err
	}

	other := findMatchingBroker(brokerAddress(b.b), meta.Brokers)
	if other == nil { // no broker found
		closeBroker(b.b)
		return fmt.Errorf("No advertised broker with address %v found", b.Addr())
	}

	debugf("found matching broker %v with id %v", other.Addr(), other.ID())
	b.id = other.ID()
	return nil
}

// Addr returns the configured broker endpoint.
func (b *Broker) Addr() string {
	return b.b.Addr()
}

// GetMetadata fetches most recent cluster metadata from the broker.
func (b *Broker) GetMetadata(topics ...string) (*sarama.MetadataResponse, error) {
	return queryMetadataWithRetry(b.b, b.cfg, topics)
}

// GetTopicsMetadata fetches most recent topics/partition metadata from the broker.
func (b *Broker) GetTopicsMetadata(topics ...string) ([]*sarama.TopicMetadata, error) {
	r, err := b.GetMetadata(topics...)
	if err != nil {
		return nil, err
	}
	return r.Topics, nil
}

// PartitionOffset fetches the available offset from a partition.
func (b *Broker) PartitionOffset(
	replicaID int32,
	topic string,
	partition int32,
	time int64,
) (int64, error) {
	req := &sarama.OffsetRequest{}
	if replicaID != noID {
		req.SetReplicaID(replicaID)
	}
	req.AddBlock(topic, partition, time, 1)
	resp, err := b.b.GetAvailableOffsets(req)
	if err != nil {
		return -1, err
	}

	block := resp.GetBlock(topic, partition)
	if len(block.Offsets) == 0 {
		return -1, nil
	}

	return block.Offsets[0], nil
}

// ID returns the broker or -1 if the broker id is unknown.
func (b *Broker) ID() int32 {
	if b.id == noID {
		return b.b.ID()
	}
	return b.id
}

func queryMetadataWithRetry(
	b *sarama.Broker,
	cfg *sarama.Config,
	topics []string,
) (r *sarama.MetadataResponse, err error) {
	err = withRetry(b, cfg, func() (e error) {
		requ := &sarama.MetadataRequest{Topics: topics}
		r, e = b.GetMetadata(requ)
		return
	})
	return
}

func closeBroker(b *sarama.Broker) {
	if ok, _ := b.Connected(); ok {
		b.Close()
	}
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
