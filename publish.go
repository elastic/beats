package main

import (
    "encoding/json"
    "fmt"
    "github.com/mattbaird/elastigo/api"
    "github.com/mattbaird/elastigo/core"
    "labix.org/v2/mgo/bson"
    "os"
    "strconv"
    "strings"
    "time"
)

type PublisherType struct {
    name     string
    disabled bool

    RefreshTopologyTimer <-chan time.Time
    TopologyMap          map[string]string
}

var Publisher PublisherType

// Config
type tomlAgent struct {
    Name                  string
    Refresh_topology_freq int
    Ignore_outgoing       bool
}
type tomlMothership struct {
    Host     string
    Port     int
    Protocol string
    Username string
    Password string
}

type Event struct {
    Timestamp    time.Time `json:"@timestamp"`
    Type         string    `json:"type"`
    Src_ip       string    `json:"src_ip"`
    Src_port     uint16    `json:"src_port"`
    Src_proc     string    `json:"src_proc"`
    Src_country  string    `json:"src_country"`
    Src_server   string    `json:"src_server"`
    Dst_ip       string    `json:"dst_ip"`
    Dst_port     uint16    `json:"dst_port"`
    Dst_proc     string    `json:"dst_proc"`
    Dst_server   string    `json:"dst_server"`
    ResponseTime int32     `json:"responsetime"`
    Status       string    `json:"status"`
    RequestRaw   string    `json:"request_raw"`
    ResponseRaw  string    `json:"response_raw"`

    Mysql bson.M `json:"mysql"`
    Http  bson.M `json:"http"`
    Redis bson.M `json:"redis"`
}

type Topology struct {
    Name string `json:"name"`
    Ip   string `json:"ip"`
}

func PrintPublishEvent(event *Event) {
    json, err := json.MarshalIndent(event, "", "  ")
    if err != nil {
        ERR("json.Marshal: %s", err)
    } else {
        DEBUG("publish", "Publish: %s", string(json))
    }
}

func (publisher *PublisherType) GetServerName(ip string) string {
    // in case the IP is localhost, return current agent name
    islocal, err := IsLoopback(ip)
    if err != nil {
        ERR("Parsing IP %s fails with: %s", ip, err)
        return ""
    } else {
        if islocal {
            return publisher.name
        }
    }
    // find the agent with the desired IP
    name, exists := publisher.TopologyMap[ip]
    if !exists {
        return ""
    }
    return name
}

func (publisher *PublisherType) PublishHttpTransaction(t *HttpTransaction) error {
    var err error
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := t.Http["response"].(bson.M)["phrase"].(string)

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if _Config.Agent.Ignore_outgoing && dst_server != "" &&
        dst_server != publisher.name {
        // duplicated transaction -> ignore it
        DEBUG("publish", "Ignore duplicated HTTP transaction on %s: %s -> %s", publisher.name, src_server, dst_server)
        return nil
    }

    var src_country = ""
    if _GeoLite != nil {
        if len(src_server) == 0 { // only for external IP addresses
            loc := _GeoLite.GetLocationByIP(t.Src.Ip)
            if loc != nil {
                src_country = loc.CountryCode
            }
        }
    }

    event := Event{
        t.ts, "http", t.Src.Ip, t.Src.Port, t.Src.Proc, src_country, src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
        t.ResponseTime, status, t.Request_raw, t.Response_raw,
        nil, t.Http, nil}

    if IS_DEBUG("publish") {
        PrintPublishEvent(&event)
    }

    // add Http transaction
    if !publisher.disabled {
        _, err = core.Index(index, "http", "", nil, event)
    }

    return err

}

func (publisher *PublisherType) PublishMysqlTransaction(t *MysqlTransaction) error {
    var err error
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := t.Mysql["error_message"].(string)
    if len(status) == 0 {
        status = "OK"
    }

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if _Config.Agent.Ignore_outgoing && dst_server != "" &&
        dst_server != publisher.name {
        // duplicated transaction -> ignore it
        DEBUG("publish", "Ignore duplicated MySQL transaction on %s: %s -> %s", publisher.name, src_server, dst_server)
        return nil
    }

    event := Event{
        t.ts, "mysql", t.Src.Ip, t.Src.Port, t.Src.Proc, "", src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
        t.ResponseTime, status, t.Request_raw, t.Response_raw,
        t.Mysql, nil, nil}

    if IS_DEBUG("publish") {
        PrintPublishEvent(&event)
    }

    // add Mysql transaction
    if !publisher.disabled {
        _, err = core.Index(index, "mysql", "", nil, event)
    }

    return err

}

func (publisher *PublisherType) PublishRedisTransaction(t *RedisTransaction) error {
    var err error
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := "OK"

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if _Config.Agent.Ignore_outgoing && dst_server != "" &&
        dst_server != publisher.name {
        // duplicated transaction -> ignore it
        DEBUG("publish", "Ignore duplicated REDIS transaction on %s: %s -> %s", publisher.name, src_server, dst_server)
        return nil
    }

    event := Event{
        t.ts, "redis", t.Src.Ip, t.Src.Port, t.Src.Proc, "", src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
        t.ResponseTime, status, t.Request_raw, t.Response_raw,
        nil, nil, t.Redis}

    if IS_DEBUG("publish") {
        PrintPublishEvent(&event)
    }

    // add Redis transaction
    if !publisher.disabled {
        _, err = core.Index(index, "redis", "", nil, event)
    }

    return err
}

func (publisher *PublisherType) UpdateTopologyPeriodically() {
    for _ = range publisher.RefreshTopologyTimer {
        publisher.UpdateTopology()
    }
}

func (publisher *PublisherType) UpdateTopology() {

    DEBUG("publish", "Updating Topology")

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
    publisher.TopologyMap = TopologyMapTmp

    DEBUG("publish", "[%s] Map: %s", publisher.name, publisher.TopologyMap)
}

func (publisher *PublisherType) PublishTopology(params ...string) error {

    var localAddrs []string = params

    if len(params) == 0 {
        addrs, err := LocalAddrs()
        if err != nil {
            ERR("Getting local IP addresses fails with: %s", err)
            return err
        }
        localAddrs = addrs
    }

    // delete old IP addresses
    searchJson := fmt.Sprintf("{query: {term: {name: %s}}}", strconv.Quote(publisher.name))
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
                    Topology{publisher.name, addr})
                if err != nil {
                    return err
                }
                if !res.Ok {
                    ERR("Fail to add new topology entry")
                }
            }
        }
    }

    DEBUG("publish", "Topology: name=%s, ips=%s", publisher.name, strings.Join(localAddrs, " "))

    // initialize local topology map
    publisher.TopologyMap = make(map[string]string)

    return nil
}

func (publisher *PublisherType) Init(publishDisabled bool) error {
    var err error

    // Set the Elasticsearch Host to Connect to
    api.Domain = _Config.Elasticsearch.Host
    api.Port = fmt.Sprintf("%d", _Config.Elasticsearch.Port)
    api.Username = _Config.Elasticsearch.Username
    api.Password = _Config.Elasticsearch.Password

    if _Config.Elasticsearch.Protocol != "" {
        api.Protocol = _Config.Elasticsearch.Protocol
    }

    INFO("Use %s://%s:%s as publisher", api.Protocol, api.Domain, api.Port)

    publisher.name = _Config.Agent.Name
    if len(publisher.name) == 0 {
        // use the hostname
        publisher.name, err = os.Hostname()
        if err != nil {
            return err
        }

        INFO("No agent name configured, using hostname '%s'", publisher.name)
    }

    publisher.disabled = publishDisabled
    if publisher.disabled {
        INFO("Dry run mode. Elasticsearch won't be updated or queried.")
    }

    RefreshTopologyFreq := 10 * time.Second
    if _Config.Agent.Refresh_topology_freq != 0 {
        RefreshTopologyFreq = time.Duration(_Config.Agent.Refresh_topology_freq) * time.Second
    }
    publisher.RefreshTopologyTimer = time.Tick(RefreshTopologyFreq)

    if !publisher.disabled {
        // register agent and its public IP addresses
        err = publisher.PublishTopology()
        if err != nil {
            ERR("Failed to publish topology: %s", err)
            return err
        }

        // update topology periodically
        go publisher.UpdateTopologyPeriodically()
    }

    return nil
}
