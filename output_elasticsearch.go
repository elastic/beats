package main

import (
    "encoding/json"
    "fmt"
    "github.com/packetbeat/elastigo/api"
    "github.com/packetbeat/elastigo/core"
    "strconv"
)

type ElasticsearchOutputType struct {
    OutputInterface
    Index string

    TopologyMap map[string]string
}

var ElasticsearchOutput ElasticsearchOutputType

func (out *ElasticsearchOutputType) Init(config tomlMothership) error {

    api.Domain = config.Host
    api.Port = fmt.Sprintf("%d", config.Port)
    api.Username = config.Username
    api.Password = config.Password
    api.BasePath = config.Path

    if config.Protocol != "" {
        api.Protocol = config.Protocol
    }

    if config.Index != "" {
        out.Index = config.Index
    } else {
        out.Index = "packetbeat"
    }

    INFO("[ElasticsearchOutput] Using Elasticsearch %s://%s:%s%s", api.Protocol, api.Domain, api.Port, api.BasePath)
    INFO("[ElasticsearchOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)

    return nil
}

func (out *ElasticsearchOutputType) GetNameByIP(ip string) string {
    name, exists := out.TopologyMap[ip]
    if !exists {
        return ""
    }
    return name
}
func (out *ElasticsearchOutputType) PublishIPs(name string, localAddrs []string) error {
    // delete old IP addresses
    searchJson := fmt.Sprintf("{query: {term: {name: %s}}}", strconv.Quote(name))
    res, err := core.SearchRequest("packetbeat-topology", "server-ip", nil, searchJson)
    if err == nil {
        for _, server := range res.Hits.Hits {

            var top Topology
            err = json.Unmarshal([]byte(*server.Source), &top)
            if err != nil {
                ERR("Failed to unmarshal json data: %s", err)
            }
            if !stringInSlice(top.Ip, localAddrs) {
                res, err := core.Delete("packetbeat-topology", "server-ip" /*id*/, top.Ip, nil)
                if err != nil {
                    ERR("Failed to delete the old IP address from packetbeat-topology")
                }
                if !res.Ok {
                    ERR("Fail to delete old topology entry")
                }
            }

        }
    }

    // add new IP addresses
    for _, addr := range localAddrs {

        // check if the IP is already in the elasticsearch, before adding it
        found, err := core.Exists("packetbeat-topology", "server-ip" /*id*/, addr, nil)
        if err != nil {
            ERR("core.Exists fails with: %s", err)
        } else {

            if !found {
                res, err := core.Index("packetbeat-topology", "server-ip" /*id*/, addr, nil,
                    Topology{name, addr})
                if err != nil {
                    return err
                }
                if !res.Ok {
                    ERR("Fail to add new topology entry")
                }
            }
        }
    }

    out.UpdateLocalTopologyMap()

    return nil
}

func (out *ElasticsearchOutputType) UpdateLocalTopologyMap() {

    // get all agents IPs from Elasticsearch
    TopologyMapTmp := make(map[string]string)
    res, err := core.SearchUri("packetbeat-topology", "server-ip", nil)
    if err == nil {
        for _, server := range res.Hits.Hits {
            var top Topology
            err = json.Unmarshal([]byte(*server.Source), &top)
            if err != nil {
                ERR("json.Unmarshal fails with: %s", err)
            }
            // add mapping
            TopologyMapTmp[top.Ip] = top.Name
        }
    } else {
        ERR("core.SearchRequest fails with: %s", err)
    }

    // update topology map
    out.TopologyMap = TopologyMapTmp

    DEBUG("output_elasticsearch", "Topology map %s", out.TopologyMap)
}

func (out *ElasticsearchOutputType) PublishEvent(event *Event) error {

    index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, event.Timestamp.Year(), event.Timestamp.Month(), event.Timestamp.Day())
    _, err := core.Index(index, event.Type, "", nil, event)
    return err
}
