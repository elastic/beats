package elasticsearch

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
	"github.com/elastic/libbeat/outputs/mode"
)

var debug = logp.MakeDebug("elasticsearch")

var (
	// ErrNoHostsConfigured indicates missing host or hosts configuration
	ErrNoHostsConfigured = errors.New("no host configuration found")

	// ErrNotConnected indicates failure due to client having no valid connection
	ErrNotConnected = errors.New("not connected")

	// ErrJsonEncodeFailed indicates encoding failures
	ErrJsonEncodeFailed = errors.New("json encode failed")
)

const (
	defaultEsOpenTimeout = 3000 * time.Millisecond

	defaultMaxRetries = 3

	elasticsearchDefaultTimeout = 30 * time.Second
)

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", elasticsearchOutputPlugin{})
}

type elasticsearchOutputPlugin struct{}

type elasticsearchOutput struct {
	index   string
	mode    mode.ConnectionMode
	clients []mode.ProtocolClient

	TopologyExpire int
	TopologyMap    atomic.Value // Value holds a map[string][string]
	ttlEnabled     bool
}

type requestExecutor interface {
	request(method, path string, params map[string]string, body interface{}) ([]byte, error)
}

type bulkMeta struct {
	Index bulkMetaIndex `json:"index"`
}

type bulkMetaIndex struct {
	Index   string `json:"_index"`
	DocType string `json:"_type"`
}

// NewOutput instantiates a new output plugin instance publishing to elasticsearch.
func (f elasticsearchOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	topologyExpire int,
) (outputs.Outputer, error) {
	output := &elasticsearchOutput{}
	err := output.init(beat, *config, topologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type publishedTopology struct {
	Name string
	IPs  string
}

func (out *elasticsearchOutput) init(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) error {

	clients, err := makeClients(beat, config)
	if err != nil {
		return err
	}

	timeout := elasticsearchDefaultTimeout
	if config.Timeout != 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	maxRetries := defaultMaxRetries
	if config.Max_retries != nil {
		maxRetries = *config.Max_retries
	}

	var waitRetry = time.Duration(1) * time.Second

	var m mode.ConnectionMode
	out.clients = clients
	if len(clients) == 1 {
		client := clients[0]
		m, err = mode.NewSingleConnectionMode(client, maxRetries, waitRetry, timeout)
	} else {
		loadBalance := config.LoadBalance == nil || *config.LoadBalance
		if loadBalance {
			m, err = mode.NewLoadBalancerMode(clients, maxRetries, waitRetry, timeout)
		} else {
			m, err = mode.NewFailOverConnectionMode(clients, maxRetries, waitRetry, timeout)
		}
	}
	if err != nil {
		return err
	}

	if config.Save_topology {
		err := out.EnableTTL()
		if err != nil {
			logp.Err("Fail to set _ttl mapping: %s", err)
			// keep trying in the background
			go func() {
				for {
					err := out.EnableTTL()
					if err == nil {
						break
					}
					logp.Err("Fail to set _ttl mapping: %s", err)
					time.Sleep(5 * time.Second)
				}
			}()
		}
	}

	out.TopologyExpire = 15000
	if topologyExpire != 0 {
		out.TopologyExpire = topologyExpire * 1000 // millisec
	}

	out.mode = m
	if config.Index != "" {
		out.index = config.Index
	} else {
		out.index = beat
	}
	return nil
}

func makeClients(
	beat string,
	config outputs.MothershipConfig,
) ([]mode.ProtocolClient, error) {
	var urls []string
	if len(config.Hosts) > 0 {
		// use hosts setting
		for _, host := range config.Hosts {
			url, err := getURL(config.Protocol, config.Path, host)
			if err != nil {
				logp.Err("Invalid host param set: %s, Error: %v", host, err)
			}

			urls = append(urls, url)
		}
	} else {
		// usage of host and port is deprecated as it is replaced by hosts
		url := fmt.Sprintf("%s://%s:%d%s", config.Protocol, config.Host, config.Port, config.Path)
		urls = append(urls, url)
	}

	index := beat
	if config.Index != "" {
		index = config.Index
	}

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return nil, err
	}

	var clients []mode.ProtocolClient
	for _, url := range urls {
		client := NewClient(url, index, tlsConfig, config.Username, config.Password)
		clients = append(clients, client)
	}
	return clients, nil
}

func (out *elasticsearchOutput) PublishEvent(
	signaler outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	return out.mode.PublishEvent(signaler, event)
}

func (out *elasticsearchOutput) BulkPublish(
	trans outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	return out.mode.PublishEvents(trans, events)
}

// Enable using ttl as paramters in a server-ip doc type
func (out *elasticsearchOutput) EnableTTL() error {
	client := out.randomClient()
	if client == nil {
		return ErrNotConnected
	}

	setting := map[string]interface{}{
		"server-ip": map[string]interface{}{
			"_ttl": map[string]string{"enabled": "true", "default": "15s"},
		},
	}

	// make sure the .packetbeat-topology index exists
	// Ignore error here, as CreateIndex will error (400 Bad Request) if index
	// already exists. If index could not be created, next api call to index will
	// fail anyway.
	index := ".packetbeat-topology"
	_, _ = client.CreateIndex(index, nil)
	_, err := client.Index(index, "server-ip", "_mapping", nil, setting)
	if err != nil {
		return err
	}

	out.ttlEnabled = true
	return nil
}

// Get the name of a shipper by its IP address from the local topology map
func (out *elasticsearchOutput) GetNameByIP(ip string) string {
	topologyMap, ok := out.TopologyMap.Load().(map[string]string)
	if ok {
		name, exists := topologyMap[ip]
		if exists {
			return name
		}
	}
	return ""
}

// Each shipper publishes a list of IPs together with its name to Elasticsearch
func (out *elasticsearchOutput) PublishIPs(name string, localAddrs []string) error {
	if !out.ttlEnabled {
		logp.Debug("output_elasticsearch", "Not publishing IPs because TTL was not yet confirmed to be enabled")
		return nil
	}

	client := out.randomClient()
	if client == nil {
		return ErrNotConnected
	}

	logp.Debug("output_elasticsearch", "Publish IPs %s with expiration time %d", localAddrs, out.TopologyExpire)
	params := map[string]string{
		"ttl":     fmt.Sprintf("%dms", out.TopologyExpire),
		"refresh": "true",
	}
	_, err := client.Index(
		".packetbeat-topology", //index
		"server-ip",            //type
		name,                   // id
		params,                 // parameters
		publishedTopology{name, strings.Join(localAddrs, ",")}, // body
	)

	if err != nil {
		logp.Err("Fail to publish IP addresses: %s", err)
		return err
	}

	out.UpdateLocalTopologyMap(client)

	return nil
}

// Update the local topology map
func (out *elasticsearchOutput) UpdateLocalTopologyMap(client *Client) {

	// get all shippers IPs from Elasticsearch
	topologyMapTmp := make(map[string]string)

	res, err := SearchURI(client, ".packetbeat-topology", "server-ip", nil)
	if err == nil {
		for _, obj := range res.Hits.Hits {
			var result QueryResult
			err = json.Unmarshal(obj, &result)
			if err != nil {
				return
			}

			var pub publishedTopology
			err = json.Unmarshal(result.Source, &pub)
			if err != nil {
				logp.Err("json.Unmarshal fails with: %s", err)
			}
			// add mapping
			ipaddrs := strings.Split(pub.IPs, ",")
			for _, addr := range ipaddrs {
				topologyMapTmp[addr] = pub.Name
			}
		}
	} else {
		logp.Err("Getting topology map fails with: %s", err)
	}

	// update topology map
	out.TopologyMap.Store(topologyMapTmp)

	logp.Debug("output_elasticsearch", "Topology map %s", topologyMapTmp)
}

func (out *elasticsearchOutput) randomClient() *Client {
	switch len(out.clients) {
	case 0:
		return nil
	case 1:
		return out.clients[0].(*Client).Clone()
	default:
		return out.clients[rand.Intn(len(out.clients))].(*Client).Clone()
	}
}
