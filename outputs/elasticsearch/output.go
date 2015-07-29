package elasticsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

type ElasticsearchOutput struct {
	Index          string
	TopologyExpire int
	Conn           *Elasticsearch
	FlushInterval  time.Duration
	BulkMaxSize    int

	TopologyMap  map[string]string
	sendingQueue chan EventMsg

	ttlEnabled bool
}

type PublishedTopology struct {
	Name string
	IPs  string
}

// Initialize Elasticsearch as output
func (out *ElasticsearchOutput) Init(config outputs.MothershipConfig, topology_expire int) error {

	if len(config.Protocol) == 0 {
		config.Protocol = "http"
	}

	var urls []string

	if len(config.Hosts) > 0 {
		// use hosts setting
		for _, host := range config.Hosts {
			url := fmt.Sprintf("%s://%s%s", config.Protocol, host, config.Path)
			urls = append(urls, url)
		}
	} else {
		// use host and port settings
		url := fmt.Sprintf("%s://%s:%d%s", config.Protocol, config.Host, config.Port, config.Path)
		urls = append(urls, url)
	}

	es := NewElasticsearch(urls, config.Username, config.Password)
	out.Conn = es

	if config.Index != "" {
		out.Index = config.Index
	} else {
		out.Index = "packetbeat"
	}

	out.TopologyExpire = 15000
	if topology_expire != 0 {
		out.TopologyExpire = topology_expire /*sec*/ * 1000 // millisec
	}

	out.FlushInterval = 1000 * time.Millisecond
	if config.Flush_interval != nil {
		out.FlushInterval = time.Duration(*config.Flush_interval) * time.Millisecond
	}
	out.BulkMaxSize = 10000
	if config.Bulk_size != nil {
		out.BulkMaxSize = *config.Bulk_size
	}

	if config.Max_retries != nil {
		out.Conn.SetMaxRetries(*config.Max_retries)
	}

	logp.Info("[ElasticsearchOutput] Using Elasticsearch %s", urls)
	logp.Info("[ElasticsearchOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
	logp.Info("[ElasticsearchOutput] Topology expires after %ds", out.TopologyExpire/1000)
	if out.FlushInterval > 0 {
		logp.Info("[ElasticsearchOutput] Insert events in batches. Flush interval is %s. Bulk size is %d.", out.FlushInterval, out.BulkMaxSize)
	} else {
		logp.Info("[ElasticsearchOutput] Insert events one by one. This might affect the performance of the shipper.")
	}

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

	out.sendingQueue = make(chan EventMsg, 1000)
	go out.SendMessagesGoroutine()

	return nil
}

// Enable using ttl as paramters in a server-ip doc type
func (out *ElasticsearchOutput) EnableTTL() error {

	// make sure the .packetbeat-topology index exists
	out.Conn.CreateIndex(".packetbeat-topology")

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
func (out *ElasticsearchOutput) GetNameByIP(ip string) string {
	name, exists := out.TopologyMap[ip]
	if !exists {
		return ""
	}
	return name
}

// Insert a list of events in the bulkChannel
func (out *ElasticsearchOutput) InsertBulkMessage(bulkChannel chan interface{}) {
	close(bulkChannel)
	go func(channel chan interface{}) {
		_, err := out.Conn.Bulk("", "", nil, channel)
		if err != nil {
			logp.Err("Fail to perform many index operations in a single API call: %s", err)
		}
	}(bulkChannel)
}

// Goroutine that sends one or multiple events to Elasticsearch.
// If the flush_interval > 0, then the events are sent in batches. Otherwise, one by one.
func (out *ElasticsearchOutput) SendMessagesGoroutine() {
	flushChannel := make(<-chan time.Time)

	if out.FlushInterval > 0 {
		flushTicker := time.NewTicker(out.FlushInterval)
		flushChannel = flushTicker.C
	}

	bulkChannel := make(chan interface{}, out.BulkMaxSize)

	for {
		select {
		case msg := <-out.sendingQueue:
			index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, msg.Ts.Year(), msg.Ts.Month(), msg.Ts.Day())
			if out.FlushInterval > 0 {
				// insert the events in batches
				if len(bulkChannel)+2 > out.BulkMaxSize {
					logp.Debug("output_elasticsearch", "Channel size reached. Calling bulk")
					out.InsertBulkMessage(bulkChannel)
					bulkChannel = make(chan interface{}, out.BulkMaxSize)
				}
				bulkChannel <- map[string]interface{}{
					"index": map[string]interface{}{
						"_index": index,
						"_type":  msg.Event["type"].(string),
					},
				}
				bulkChannel <- msg.Event
			} else {
				// insert the events one by one
				_, err := out.Conn.Index(index, msg.Event["type"].(string), "", nil, msg.Event)
				if err != nil {
					logp.Err("Fail to insert a single event: %s", err)
				}
			}
		case _ = <-flushChannel:
			out.InsertBulkMessage(bulkChannel)
			bulkChannel = make(chan interface{}, out.BulkMaxSize)
		}
	}
}

// Each shipper publishes a list of IPs together with its name to Elasticsearch
func (out *ElasticsearchOutput) PublishIPs(name string, localAddrs []string) error {
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
		PublishedTopology{name, strings.Join(localAddrs, ",")} /* body */)

	if err != nil {
		logp.Err("Fail to publish IP addresses: %s", err)
		return err
	}

	out.UpdateLocalTopologyMap()

	return nil
}

// Update the local topology map
func (out *ElasticsearchOutput) UpdateLocalTopologyMap() {

	// get all shippers IPs from Elasticsearch
	TopologyMapTmp := make(map[string]string)

	res, err := out.Conn.SearchUri(".packetbeat-topology", "server-ip", nil)
	if err == nil {
		for _, obj := range res.Hits.Hits {
			var result QueryResult
			err = json.Unmarshal(obj, &result)
			if err != nil {
				return
			}

			var pub PublishedTopology
			err = json.Unmarshal(result.Source, &pub)
			if err != nil {
				logp.Err("json.Unmarshal fails with: %s", err)
			}
			// add mapping
			ipaddrs := strings.Split(pub.IPs, ",")
			for _, addr := range ipaddrs {
				TopologyMapTmp[addr] = pub.Name
			}
		}
	} else {
		logp.Err("Getting topology map fails with: %s", err)
	}

	// update topology map
	out.TopologyMap = TopologyMapTmp

	logp.Debug("output_elasticsearch", "Topology map %s", out.TopologyMap)
}

// Publish an event by adding it to the queue of events.
func (out *ElasticsearchOutput) PublishEvent(ts time.Time, event common.MapStr) error {

	out.sendingQueue <- EventMsg{Ts: ts, Event: event}

	logp.Debug("output_elasticsearch", "Publish event: %s", event)
	return nil
}
