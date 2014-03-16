package main

import (
    "fmt"
    "labix.org/v2/mgo/bson"
    "github.com/mattbaird/elastigo/api"
    "github.com/mattbaird/elastigo/core"
    "encoding/json"
    "os"
    "time"
    "strings"
    "strconv"
)

type PublisherType struct {
    name         string

    url             string
    mother_host     string
    mother_port     string

    RefreshTopologyTimer <-chan time.Time
    TopologyMap map[string]TopologyMapping
}

var Publisher PublisherType

// Config
type tomlAgent struct {
    Name        string
	Refresh_topology_freq int
}
type tomlMothership struct {
    Host string
    Port int
}

type Event struct {
    Timestamp time.Time `json:"@timestamp"`
    Type string `json:"type"`
    Src_ip string `json:"src_ip"`
    Src_port uint16 `json:"src_port"`
    Src_proc string `json:"src_proc"`
    Src_country string `json:"src_country"`
    Src_server string `json:"src_server"`
    Dst_ip string `json:"dst_ip"`
    Dst_port uint16 `json:"dst_port"`
    Dst_proc string `json:"dst_proc"`
    Dst_server string `json:"dst_server"`
    ResponseTime int32 `json:"responsetime"`
    Status string `json:"status"`
    RequestRaw string `json:"request_raw"`
    ResponseRaw string `json:"response_raw"`

    Mysql bson.M `json:"mysql"`
    Http bson.M `json:"http"`
    Redis bson.M `json:"redis"`
}

type Topology struct {
    Name string `json:"name"`
    Ip string `json:"ip"`
}

type TopologyMapping struct {
    Name string
    RefreshTime time.Time
}

func (publisher *PublisherType) GetServerName(ip string) string {
    mapping, exists := publisher.TopologyMap[ip]

    if !exists {
        return ""
    }
    return mapping.Name
}

func (publisher *PublisherType) PublishHttpTransaction(t *HttpTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := t.Http["response"].(bson.M)["phrase"].(string)

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if dst_server != publisher.name {
        // duplicated transaction -> ignore it
        return nil
    }

    var src_country = ""
    if len(src_server) == 0 {
            loc := _GeoLite.GetLocationByIP(t.Src.Ip)
            if loc != nil {
                    src_country = loc.CountryCode
            }
    }

    _, err := core.Index(true, index, "http","", Event{
        t.ts, "http", t.Src.Ip, t.Src.Port, t.Src.Proc, src_country, src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
	t.ResponseTime, status, t.Request_raw, t.Response_raw,
        nil, t.Http, nil})

    DEBUG("publish", "Sent Http transaction [%s->%s]:\n%s", t.Src.Proc, t.Dst.Proc, t.Http)
    return err

}

func (publisher *PublisherType) PublishMysqlTransaction(t *MysqlTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := t.Mysql["error_message"].(string)
    if len(status) == 0 {
        status = "OK"
    }

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if dst_server != publisher.name {
        // duplicated transaction -> ignore it
        return nil
    }

    _, err := core.Index(true, index, "mysql", "", Event{
        t.ts, "mysql", t.Src.Ip, t.Src.Port, t.Src.Proc, "", src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
	t.ResponseTime, status, t.Request_raw, t.Response_raw,
        t.Mysql, nil, nil})

    DEBUG("publish", "Sent MySQL transaction [%s->%s]:\n%s", t.Src.Proc, t.Dst.Proc, t.Mysql)

    return err

}

func (publisher *PublisherType) PublishRedisTransaction(t *RedisTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())

    status := "OK"

    src_server := publisher.GetServerName(t.Src.Ip)
    dst_server := publisher.GetServerName(t.Dst.Ip)

    if dst_server != publisher.name {
        // duplicated transaction -> ignore it
        return nil
    }

    _, err := core.Index(true, index, "redis","", Event{
        t.ts, "redis", t.Src.Ip, t.Src.Port, t.Src.Proc, "", src_server,
        t.Dst.Ip, t.Dst.Port, t.Dst.Proc, dst_server,
	t.ResponseTime, status, t.Request_raw, t.Response_raw,
        nil, nil, t.Redis})

    DEBUG("publish", "Sent Redis transaction [%s->%s]:\n%s", t.Src.Proc, t.Dst.Proc, t.Redis)
    return err

}

func (publisher *PublisherType) UpdateTopology() {

    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port


    for _ = range publisher.RefreshTopologyTimer {

		DEBUG("publish", "Updating Topology")

        searchJson := `{
            "query": {
                "match_all": {}
            }
        }`
        res, err := core.SearchRequest(true, "packetbeat-topology", "server-ip", searchJson, "", 0)
        refreshTime := time.Now()
        if err == nil {
            for _, server := range res.Hits.Hits {
                var top Topology
                err = json.Unmarshal(server.Source, &top)
                if err != nil {
                    ERR("Failed to unmarshal json data: %s", err)
                }

                // refresh time or add new server ip
                entry := TopologyMapping{Name: top.Name, RefreshTime: refreshTime}
                publisher.TopologyMap[top.Ip] = entry

            }
            // delete old data from map
            for ip, mapping := range publisher.TopologyMap {
                if !refreshTime.Equal(mapping.RefreshTime) {
                   delete(publisher.TopologyMap, ip)
                }
            }
            DEBUG("publish", "Map: %s", publisher.TopologyMap)
        } else {
            ERR("Failed to fetch packetbeat-topology data")
        }
    }
}

func (publisher *PublisherType) PublishTopology() error {

    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    localAddrs, err := LocalAddrs()
    if err != nil {
        ERR("Failed to get local IP addresses: %s", err)
    }

    for _, addr := range localAddrs {

        // check if the IP is already in the elasticsearch, before adding it 
        searchJson := fmt.Sprintf("{query: {term: {ip: %s}}}",strconv.Quote(addr))
        res, err := core.SearchRequest(true, "packetbeat-topology", "server-ip", searchJson, "", 0)
        found := true
        if err != nil  {
            found = false
        } else {
            if res.Hits.Total == 0 {
                found = false
            }
        }

        if !found {
            _, err = core.Index(true, "packetbeat-topology", "server-ip", "",
                Topology{publisher.name, addr})
            if err != nil {
                return err
            }
        }
    }

    DEBUG("publish", "Topology: name=%s, ips=%s", publisher.name, strings.Join(localAddrs, " "))

    publisher.TopologyMap = make(map[string]TopologyMapping)

    go publisher.UpdateTopology()

    return nil
}

func (publisher *PublisherType) Init() error {
    var err error

    publisher.mother_host = _Config.Elasticsearch.Host
    publisher.mother_port = fmt.Sprintf("%d", _Config.Elasticsearch.Port)

    publisher.url = fmt.Sprintf("%s:%s", publisher.mother_host, publisher.mother_port)
    INFO("Use %s as publisher", publisher.url)

    publisher.name = _Config.Agent.Name
    if len(publisher.name) == 0 {
        // use the hostname
        publisher.name, err = os.Hostname()
        if err != nil {
            return err
        }

        INFO("No agent name configured, using hostname '%s'", publisher.name)
    }

	RefreshTopologyFreq := 10 * time.Second
	if _Config.Agent.Refresh_topology_freq != 0 {
		RefreshTopologyFreq = time.Duration(_Config.Agent.Refresh_topology_freq) * time.Second
	}
    publisher.RefreshTopologyTimer = time.Tick( RefreshTopologyFreq )
	DEBUG("publish", "RefreshTopologyFreq=%d", _Config.Agent.Refresh_topology_freq)

    // register agent and its public IP addresses
    err = publisher.PublishTopology()
    if err != nil {
        ERR("Failed to publish topology: %s", err)
        return err
    }
    return nil
}
