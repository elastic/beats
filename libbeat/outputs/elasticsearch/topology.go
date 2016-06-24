package elasticsearch

import (
	"encoding/json"
	"math/rand"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/outputs/mode"
)

type topology struct {
	clients []mode.ProtocolClient

	TopologyMap atomic.Value // Value holds a map[string][string]
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
	client := t.randomClient()
	if client == nil {
		return ErrNotConnected
	}

	debugf("Publish IPs: %s", localAddrs)

	params := map[string]string{
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

	debugf("Topology map %s", topology)
	return topology, nil
}
