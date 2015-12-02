package elasticsearch

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/elastic/libbeat/logp"
	"github.com/elastic/libbeat/outputs/mode"
)

type topology struct {
	clients []mode.ProtocolClient

	TopologyExpire int
	TopologyMap    atomic.Value // Value holds a map[string][string]
	ttlEnabled     bool
}

type publishedTopology struct {
	Name string
	IPs  string
}

func (t *topology) randomClient() *Client {
	switch len(t.clients) {
	case 0:
		return nil
	case 1:
		return t.clients[0].(*Client).Clone()
	default:
		return t.clients[rand.Intn(len(t.clients))].(*Client).Clone()
	}
}

// Enable using ttl as paramters in a server-ip doc type
func (t *topology) EnableTTL() error {
	client := t.randomClient()
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
	_, _, _ = client.CreateIndex(index, nil)
	_, _, err := client.Index(index, "server-ip", "_mapping", nil, setting)
	if err != nil {
		return err
	}

	t.ttlEnabled = true
	return nil
}

// Get the name of a shipper by its IP address from the local topology map
func (t *topology) GetNameByIP(ip string) string {
	topologyMap, ok := t.TopologyMap.Load().(map[string]string)
	if ok {
		name, exists := topologyMap[ip]
		if exists {
			return name
		}
	}
	return ""
}

// Each shipper publishes a list of IPs together with its name to Elasticsearch
func (t *topology) PublishIPs(name string, localAddrs []string) error {
	if !t.ttlEnabled {
		logp.Debug("output_elasticsearch",
			"Not publishing IPs because TTL was not yet confirmed to be enabled")
		return nil
	}

	client := t.randomClient()
	if client == nil {
		return ErrNotConnected
	}

	logp.Debug("output_elasticsearch",
		"Publish IPs %s with expiration time %d", localAddrs, t.TopologyExpire)

	params := map[string]string{
		"ttl":     fmt.Sprintf("%dms", t.TopologyExpire),
		"refresh": "true",
	}
	_, _, err := client.Index(
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

	newMap, err := loadTopolgyMap(client)
	if err != nil {
		return err
	}
	t.TopologyMap.Store(newMap)

	return nil
}

// Update the local topology map
func loadTopolgyMap(client *Client) (map[string]string, error) {
	// get all shippers IPs from Elasticsearch

	index := ".packetbeat-topology"
	docType := "server-ip"

	// get number of entries in index for search query to return all entries in one query
	_, cntRes, err := client.CountSearchURI(index, docType, nil)
	if err != nil {
		logp.Err("Getting topology map fails with: %s", err)
		return nil, err
	}

	params := map[string]string{"size": strconv.Itoa(cntRes.Count)}
	_, res, err := client.SearchURI(index, docType, params)
	if err != nil {
		logp.Err("Getting topology map fails with: %s", err)
		return nil, err
	}

	topology := make(map[string]string)
	for _, obj := range res.Hits.Hits {
		var result QueryResult
		err = json.Unmarshal(obj, &result)
		if err != nil {
			logp.Err("Failed to read response: %v", err)
			return nil, err
		}

		var pub publishedTopology
		err = json.Unmarshal(result.Source, &pub)
		if err != nil {
			logp.Err("json.Unmarshal fails with: %s", err)
			return nil, err
		}

		// add mapping
		for _, addr := range strings.Split(pub.IPs, ",") {
			topology[addr] = pub.Name
		}
	}

	logp.Debug("output_elasticsearch", "Topology map %s", topology)
	return topology, nil
}
