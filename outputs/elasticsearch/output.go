package elasticsearch

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/elastic/libbeat/common"
	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs"
)

var debug = logp.MakeDebug("elasticsearch")

func init() {
	outputs.RegisterOutputPlugin("elasticsearch", ElasticsearchOutputPlugin{})
}

func (f ElasticsearchOutputPlugin) NewOutput(
	beat string,
	config outputs.MothershipConfig,
	topology_expire int,
) (outputs.Outputer, error) {
	output := &elasticsearchOutput{}
	err := output.Init(beat, config, topology_expire)
	if err != nil {
		return nil, err
	}
	return output, nil
}

type ElasticsearchOutputPlugin struct{}

type elasticsearchOutput struct {
	Index          string
	TopologyExpire int
	Conn           *Elasticsearch
	FlushInterval  time.Duration
	BulkMaxSize    int

	TopologyMap  atomic.Value // Value holds a map[string][string]
	sendingQueue chan EventMsg

	ttlEnabled bool
}

type PublishedTopology struct {
	Name string
	IPs  string
}

// Initialize Elasticsearch as output
func (out *elasticsearchOutput) Init(beat string, config outputs.MothershipConfig, topology_expire int) error {

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
		out.Index = beat
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

	out.sendingQueue = make(chan EventMsg, 1000)
	go out.SendMessagesGoroutine()

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

// Insert a list of events in the bulkChannel
func (out *elasticsearchOutput) InsertBulkMessage(
	pendingTrans []outputs.Signaler,
	batch []interface{},
) {
	go func(transactions []outputs.Signaler, data []interface{}) {
		_, err := out.Conn.Bulk("", "", nil, data)
		outputs.SignalAll(pendingTrans, err)
		if err != nil {
			logp.Err("Fail to perform many index operations in a single API call: %s", err)
		}
	}(pendingTrans, batch)
}

// Goroutine that sends one or multiple events to Elasticsearch.
// If the flush_interval > 0, then the events are sent in batches. Otherwise, one by one.
func (out *elasticsearchOutput) SendMessagesGoroutine() {
	flushChannel := make(<-chan time.Time)

	if out.FlushInterval > 0 {
		flushTicker := time.NewTicker(out.FlushInterval)
		flushChannel = flushTicker.C
	}

	batch := make([]interface{}, 0, out.BulkMaxSize)
	var pendingTrans []outputs.Signaler

	for {
		select {
		case msg := <-out.sendingQueue:
			index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, msg.Ts.Year(), msg.Ts.Month(), msg.Ts.Day())
			if out.FlushInterval > 0 {
				// insert the events in batches
				if len(batch)+2 > out.BulkMaxSize {
					logp.Debug("output_elasticsearch", "Channel size reached. Calling bulk")
					out.InsertBulkMessage(pendingTrans, batch)
					pendingTrans = nil
					batch = make([]interface{}, 0, out.BulkMaxSize)
				}

				meta := map[string]interface{}{
					"index": map[string]interface{}{
						"_index": index,
						"_type":  msg.Event["type"].(string),
					},
				}
				batch = append(batch, meta, msg.Event)
				if msg.Trans != nil {
					pendingTrans = append(pendingTrans, msg.Trans)
				}
			} else {
				// insert the events one by one
				_, err := out.Conn.Index(index, msg.Event["type"].(string), "", nil, msg.Event)
				outputs.Signal(msg.Trans, err)
				if err != nil {
					logp.Err("Fail to insert a single event: %s", err)
				}
			}
		case _ = <-flushChannel:
			out.InsertBulkMessage(pendingTrans, batch)
			pendingTrans = pendingTrans[:0]
			batch = make([]interface{}, 0, out.BulkMaxSize)
		}
	}
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
		PublishedTopology{name, strings.Join(localAddrs, ",")} /* body */)

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
	trans outputs.Signaler,
	ts time.Time,
	event common.MapStr,
) error {
	out.sendingQueue <- EventMsg{Trans: trans, Ts: ts, Event: event}

	logp.Debug("output_elasticsearch", "Publish event: %s", event)
	return nil
}

type eventsMetaBuilder struct {
	index string
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
			ts := event["ts"].(time.Time)
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
