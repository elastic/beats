package elasticsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"net"
	"net/url"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

var debug = logp.MakeDebug("elasticsearch")

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", elasticsearchOutputPlugin{})
}

type elasticsearchOutputPlugin struct{}

// NewOutput instantiates a new output plugin instance publishing to elasticsearch.
func (f elasticsearchOutputPlugin) NewOutput(
	beat string,
	config *outputs.MothershipConfig,
	TopologyExpire int,
) (outputs.Outputer, error) {
	output := &elasticsearchOutput{}
	err := output.Init(beat, *config, TopologyExpire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type elasticsearchOutput struct {
	Index string
	Conn  *Elasticsearch

	TopologyExpire int
	TopologyMap    atomic.Value // Value holds a map[string][string]
	ttlEnabled     bool
}

type publishedTopology struct {
	Name string
	IPs  string
}

// Initialize Elasticsearch as output
func (out *elasticsearchOutput) Init(
	beat string,
	config outputs.MothershipConfig,
	topologyExpire int,
) error {

	if len(config.Protocol) == 0 {
		config.Protocol = "http"
	}

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

	tlsConfig, err := outputs.LoadTLSConfig(config.TLS)
	if err != nil {
		return err
	}

	es := NewElasticsearch(urls, tlsConfig, config.Username, config.Password)
	out.Conn = es

	if config.Index != "" {
		out.Index = config.Index
	} else {
		out.Index = beat
	}

	out.TopologyExpire = 15000
	if topologyExpire != 0 {
		out.TopologyExpire = topologyExpire /*sec*/ * 1000 // millisec
	}

	if config.Max_retries != nil {
		out.Conn.SetMaxRetries(*config.Max_retries)
	}

	logp.Info("[ElasticsearchOutput] Using Elasticsearch %s", urls)
	logp.Info("[ElasticsearchOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
	logp.Info("[ElasticsearchOutput] Topology expires after %ds", out.TopologyExpire/1000)

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

	return nil
}

// Enable using ttl as paramters in a server-ip doc type
func (out *elasticsearchOutput) EnableTTL() error {

	// make sure the .packetbeat-topology index exists
	// Ignore error here, as CreateIndex will error (400 Bad Request) if index
	// already exists. If index could not be created, next api call to index will
	// fail anyway.
	_, _ = out.Conn.CreateIndex(".packetbeat-topology", nil)

	setting := map[string]interface{}{
		"server-ip": map[string]interface{}{
			"_ttl": map[string]string{"enabled": "true", "default": "15s"},
		},
	}

	_, err := out.Conn.Index(".packetbeat-topology", "server-ip", "_mapping", nil, setting)
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

	logp.Debug("output_elasticsearch", "Publish IPs %s with expiration time %d", localAddrs, out.TopologyExpire)
	params := map[string]string{
		"ttl":     fmt.Sprintf("%dms", out.TopologyExpire),
		"refresh": "true",
	}
	_, err := out.Conn.Index(
		".packetbeat-topology", /*index*/
		"server-ip",            /*type*/
		name,                   /* id */
		params,                 /* parameters */
		publishedTopology{name, strings.Join(localAddrs, ",")} /* body */)

	if err != nil {
		logp.Err("Fail to publish IP addresses: %s", err)
		return err
	}

	out.UpdateLocalTopologyMap()

	return nil
}

// Update the local topology map
func (out *elasticsearchOutput) UpdateLocalTopologyMap() {

	// get all shippers IPs from Elasticsearch
	topologyMapTmp := make(map[string]string)

	res, err := out.Conn.SearchURI(".packetbeat-topology", "server-ip", nil)
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

// Publish an event by adding it to the queue of events.
func (out *elasticsearchOutput) PublishEvent(
	signaler outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	index := fmt.Sprintf("%s-%d.%02d.%02d",
		out.Index, ts.Year(), ts.Month(), ts.Day())

	logp.Debug("output_elasticsearch", "Publish event: %s", event)

	// insert the events one by one
	_, err := out.Conn.Index(index, event["type"].(string), "", nil, event)
	outputs.Signal(signaler, err)
	if err != nil {
		logp.Err("Fail to insert a single event: %s", err)
	}

	return nil
}

func (out *elasticsearchOutput) BulkPublish(
	trans outputs.Signaler,
	ts time.Time,
	events []common.MapStr,
) error {
	go func() {
		request, err := out.Conn.startBulkRequest("", "", nil)
		if err != nil {
			logp.Err("Failed to perform many index operations in a single API call: %s", err)
			outputs.Signal(trans, err)
			return
		}

		for _, event := range events {
			ts := time.Time(event["timestamp"].(common.Time)).UTC()
			index := fmt.Sprintf("%s-%d.%02d.%02d",
				out.Index, ts.Year(), ts.Month(), ts.Day())
			meta := common.MapStr{
				"index": map[string]interface{}{
					"_index": index,
					"_type":  event["type"].(string),
				},
			}
			err := request.Send(meta, event)
			if err != nil {
				logp.Err("Fail to encode event: %s", err)
			}
		}

		_, err = request.Flush()
		outputs.Signal(trans, err)
		if err != nil {
			logp.Err("Failed to perform many index operations in a single API call: %s",
				err)
		}
	}()
	return nil
}

// Creates the url based on the url configuration.
// Adds missing parts with defaults (scheme, host, port)
func getURL(defaultScheme string, defaultPath string, rawURL string) (string, error) {
	addr, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	scheme := addr.Scheme
	host := addr.Host
	port := "9200"

	// sanitize parse errors if url does not contain scheme
	// if parse url looks funny, prepend schema and try again:
	if addr.Scheme == "" || (addr.Host == "" && addr.Path == "" && addr.Opaque != "") {
		rawURL = fmt.Sprintf("%v://%v", defaultScheme, rawURL)
		if tmpAddr, err := url.Parse(rawURL); err == nil {
			addr = tmpAddr
			scheme = addr.Scheme
			host = addr.Host
		} else {
			// If url doesn't have a scheme, host is written into path. For example: 192.168.3.7
			scheme = defaultScheme
			host = addr.Path
			addr.Path = ""
		}
	}

	if host == "" {
		host = "localhost"
	} else {
		// split host and optional port
		if splitHost, splitPort, err := net.SplitHostPort(host); err == nil {
			host = splitHost
			port = splitPort
		}

		// Check if ipv6
		if strings.Count(host, ":") > 1 && strings.Count(host, "]") == 0 {
			host = "[" + host + "]"
		}
	}

	// Assign default path if not set
	if addr.Path == "" {
		addr.Path = defaultPath
	}

	// reconstruct url
	addr.Scheme = scheme
	addr.Host = host + ":" + port
	return addr.String(), nil
}
