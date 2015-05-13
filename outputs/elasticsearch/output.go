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

	TopologyMap map[string]string
}

type PublishedTopology struct {
	Name string
	IPs  string
}

func (out *ElasticsearchOutput) Init(config outputs.MothershipConfig, topology_expire int) error {

	url := fmt.Sprintf("http://%s:%d", config.Host, config.Port)
	con := NewElasticsearch(url)
	out.Conn = con

	// TODO:
	//api.Username = config.Username
	//api.Password = config.Password
	//api.BasePath = config.Path

	//if config.Protocol != "" {
	//	api.Protocol = config.Protocol
	//}

	if config.Index != "" {
		out.Index = config.Index
	} else {
		out.Index = "packetbeat"
	}

	out.TopologyExpire = 15000
	if topology_expire != 0 {
		out.TopologyExpire = topology_expire /*sec*/ * 1000 // millisec
	}

	err := out.EnableTTL()
	if err != nil {
		logp.Err("Fail to set _ttl mapping: %s", err)
		return err
	}

	logp.Info("[ElasticsearchOutput] Using Elasticsearch %s", url)
	logp.Info("[ElasticsearchOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
	logp.Info("[ElasticsearchOutput] Topology expires after %ds", out.TopologyExpire/1000)

	return nil
}

func (out *ElasticsearchOutput) EnableTTL() error {
	setting := map[string]interface{}{
		"server-ip": map[string]interface{}{
			"_ttl": map[string]string{"enabled": "true", "default": "15000"},
		},
	}

	_, err := out.Conn.Index(".packetbeat-topology", "server-ip", "_mapping", nil, setting)
	if err != nil {
		return err
	}
	return nil
}

func (out *ElasticsearchOutput) GetNameByIP(ip string) string {
	name, exists := out.TopologyMap[ip]
	if !exists {
		return ""
	}
	return name
}
func (out *ElasticsearchOutput) PublishIPs(name string, localAddrs []string) error {
	logp.Debug("output_elasticsearch", "Publish IPs %s with expiration time %d", localAddrs, out.TopologyExpire)
	params := map[string]string{
		"ttl":     fmt.Sprintf("%d", out.TopologyExpire),
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

func (out *ElasticsearchOutput) PublishEvent(ts time.Time, event common.MapStr) error {

	index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, ts.Year(), ts.Month(), ts.Day())
	_, err := out.Conn.Index(index, event["type"].(string), "", nil, event)
	logp.Debug("output_elasticsearch", "Publish event")
	return err
}
