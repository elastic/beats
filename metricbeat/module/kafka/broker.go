// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kafka

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/Shopify/sarama"

	"github.com/menderesk/beats/v7/libbeat/common"
	"github.com/menderesk/beats/v7/libbeat/common/kafka"
)

// Version returns a kafka version from its string representation
func Version(version string) kafka.Version {
	return kafka.Version(version)
}

// Broker provides functionality for communicating with a single kafka broker
type Broker struct {
	broker *sarama.Broker
	cfg    *sarama.Config
	client sarama.Client

	advertisedAddr string
	id             int32
	matchID        bool
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
	Version                  kafka.Version
	Sasl                     kafka.SaslConfig
}

type GroupDescription struct {
	Members map[string]MemberDescription
}

type MemberDescription struct {
	Err        error
	ClientID   string
	ClientHost string
	Topics     map[string][]int32
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
		settings.Sasl.ConfigureSarama(cfg)
	}
	cfg.Version, _ = settings.Version.Get()

	return &Broker{
		broker:  sarama.NewBroker(host),
		cfg:     cfg,
		client:  nil,
		id:      noID,
		matchID: settings.MatchID,
	}
}

// Close the broker connection
func (b *Broker) Close() error {
	closeBroker(b.broker)
	b.client.Close()
	return nil
}

// Connect connects the broker to the configured host
func (b *Broker) Connect() error {
	if err := b.broker.Open(b.cfg); err != nil {
		return errors.Wrap(err, "broker.Open failed")
	}

	c, err := getClusterWideClient(b.Addr(), b.cfg)
	if err != nil {
		closeBroker(b.broker)
		return fmt.Errorf("getting cluster client for advertised broker with address %v: %w", b.Addr(), err)
	}
	b.client = c

	if b.id != noID || !b.matchID {
		return nil
	}

	// current broker is bootstrap only. Get metadata to find id:
	meta, err := queryMetadataWithRetry(b.broker, b.cfg, nil)
	if err != nil {
		closeBroker(b.broker)
		return errors.Wrap(err, "failed to query metadata")
	}

	finder := brokerFinder{Net: &defaultNet{}}
	other := finder.findBroker(brokerAddress(b.broker), meta.Brokers)
	if other == nil { // no broker found
		closeBroker(b.broker)
		return fmt.Errorf("No advertised broker with address %v found", b.Addr())
	}

	debugf("found matching broker %v with id %v", other.Addr(), other.ID())
	b.id = other.ID()
	b.advertisedAddr = other.Addr()

	return nil
}

// Addr returns the configured broker endpoint.
func (b *Broker) Addr() string {
	return b.broker.Addr()
}

// AdvertisedAddr returns the advertised broker address in case of
// matching broker has been found.
func (b *Broker) AdvertisedAddr() string {
	return b.advertisedAddr
}

// GetMetadata fetches most recent cluster metadata from the broker.
func (b *Broker) GetMetadata(topics ...string) (*sarama.MetadataResponse, error) {
	return queryMetadataWithRetry(b.broker, b.cfg, topics)
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
	resp, err := b.broker.GetAvailableOffsets(req)
	if err != nil {
		return -1, errors.Wrap(err, "get available offsets failed")
	}

	block := resp.GetBlock(topic, partition)
	if len(block.Offsets) == 0 {
		return -1, errors.Wrap(block.Err, "block offsets is empty")
	}

	return block.Offsets[0], nil
}

// ListGroups lists all groups managed by the broker. Other consumer
// groups might be managed by other brokers.
func (b *Broker) ListGroups() ([]string, error) {
	resp, err := b.broker.ListGroups(&sarama.ListGroupsRequest{})
	if err != nil {
		return nil, err
	}

	if resp.Err != sarama.ErrNoError {
		return nil, resp.Err
	}

	if len(resp.Groups) == 0 {
		return nil, nil
	}

	groups := make([]string, 0, len(resp.Groups))
	for name := range resp.Groups {
		groups = append(groups, name)
	}
	return groups, nil
}

// DescribeGroups fetches group details from broker.
func (b *Broker) DescribeGroups(
	queryGroups []string,
) (map[string]GroupDescription, error) {
	requ := &sarama.DescribeGroupsRequest{Groups: queryGroups}
	resp, err := b.broker.DescribeGroups(requ)
	if err != nil {
		return nil, err
	}

	if len(resp.Groups) == 0 {
		return nil, nil
	}

	groups := map[string]GroupDescription{}
	for _, descr := range resp.Groups {
		if len(descr.Members) == 0 {
			groups[descr.GroupId] = GroupDescription{}
			continue
		}

		members := map[string]MemberDescription{}
		for memberID, memberDescr := range descr.Members {
			assignment, err := memberDescr.GetMemberAssignment()
			if err != nil {
				members[memberID] = MemberDescription{
					ClientID:   memberDescr.ClientId,
					ClientHost: memberDescr.ClientHost,
					Err:        err,
				}
				continue
			}

			members[memberID] = MemberDescription{
				ClientID:   memberDescr.ClientId,
				ClientHost: memberDescr.ClientHost,
				Topics:     assignment.Topics,
			}
		}
		groups[descr.GroupId] = GroupDescription{Members: members}
	}

	return groups, nil
}

// FetchGroupOffsets fetches the consume offset of group.
// The partitions is a MAP mapping from topic name to partitionid array.
func (b *Broker) FetchGroupOffsets(group string, partitions map[string][]int32) (*sarama.OffsetFetchResponse, error) {
	requ := &sarama.OffsetFetchRequest{
		ConsumerGroup: group,
		Version:       1,
	}
	for topic, partition := range partitions {
		for _, partitionID := range partition {
			requ.AddPartition(topic, partitionID)
		}
	}
	return b.broker.FetchOffset(requ)
}

// FetchPartitionOffsetFromTheLeader fetches the OffsetNewest from the leader.
func (b *Broker) FetchPartitionOffsetFromTheLeader(topic string, partitionID int32) (int64, error) {
	offset, err := b.client.GetOffset(topic, partitionID, sarama.OffsetNewest)
	if err != nil {
		return -1, err
	}
	return offset, nil
}

// ID returns the broker ID or -1 if the broker id is unknown.
func (b *Broker) ID() int32 {
	if b.id == noID {
		return b.broker.ID()
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

// NetInfo can be used to obtain network information
type NetInfo interface {
	LookupIP(string) ([]net.IP, error)
	LookupAddr(string) ([]string, error)
	LocalIPAddrs() ([]net.IP, error)
	Hostname() (string, error)
}

type defaultNet struct{}

// LookupIP looks up a host using the local resolver
func (m *defaultNet) LookupIP(addr string) ([]net.IP, error) {
	return net.LookupIP(addr)
}

// LookupAddr returns the list of hosts resolving to an specific address
func (m *defaultNet) LookupAddr(address string) ([]string, error) {
	return net.LookupAddr(address)
}

// LocalIPAddrs return the list of IP addresses configured in local network interfaces
func (m *defaultNet) LocalIPAddrs() ([]net.IP, error) {
	return common.LocalIPAddrs()
}

// Hostname returns the hostname reported by the OS
func (m *defaultNet) Hostname() (string, error) {
	return os.Hostname()
}

type brokerFinder struct {
	Net NetInfo
}

func (m *brokerFinder) findBroker(addr string, brokers []*sarama.Broker) *sarama.Broker {
	lst := brokerAddresses(brokers)
	if idx, found := m.findAddress(addr, lst); found {
		return brokers[idx]
	}
	return nil
}

func (m *brokerFinder) findAddress(addr string, brokers []string) (int, bool) {
	debugf("Try to match broker to: %v", addr)

	// get connection 'port'
	host, port, err := net.SplitHostPort(addr)
	if err != nil || port == "" {
		host = addr
		port = "9092"
	}

	// compare connection address to list of broker addresses
	if i, found := indexOf(net.JoinHostPort(host, port), brokers); found {
		return i, true
	}

	// lookup local machines ips for comparing with broker addresses
	localIPs, err := m.Net.LocalIPAddrs()
	if err != nil || len(localIPs) == 0 {
		return -1, false
	}
	debugf("local machine ips: %v", localIPs)

	// try to find broker by comparing the fqdn for each known ip to list of
	// brokers
	localHosts := m.lookupHosts(localIPs)
	debugf("local machine addresses: %v", localHosts)
	for _, host := range localHosts {
		debugf("try to match with fqdn: %v (%v)", host, port)
		if i, found := indexOf(net.JoinHostPort(host, port), brokers); found {
			return i, true
		}
	}

	// try matching ip of configured host with broker list, this would
	// match if hosts of advertised addresses are IPs, but configured host
	// is a hostname
	ips, err := m.Net.LookupIP(host)
	if err == nil {
		for _, ip := range ips {
			addr := net.JoinHostPort(ip.String(), port)
			if i, found := indexOf(addr, brokers); found {
				return i, true
			}
		}
	}

	// try to find broker id by comparing the machines local hostname to
	// broker hostnames in metadata
	if host, err := m.Net.Hostname(); err == nil {
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
		ips, err := m.Net.LookupIP(bh)
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

func (m *brokerFinder) lookupHosts(ips []net.IP) []string {
	set := map[string]struct{}{}
	for _, ip := range ips {
		txt, err := ip.MarshalText()
		if err != nil {
			continue
		}

		hosts, err := m.Net.LookupAddr(string(txt))
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
			if a.Equal(b) {
				return true
			}
		}
	}
	return false
}

func getClusterWideClient(addr string, cfg *sarama.Config) (sarama.Client, error) {
	client, err := sarama.NewClient([]string{addr}, cfg)
	if err != nil {
		return nil, err
	}
	return client, nil
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
