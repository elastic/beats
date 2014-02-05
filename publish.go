package main

import (
    "fmt"
    "labix.org/v2/mgo/bson"
    "github.com/mattbaird/elastigo/api"
    "github.com/mattbaird/elastigo/core"
    "os"
    "time"
)

type PublisherType struct {
    name         string

    url             string
    mother_host     string
    mother_port     string
}

var Publisher PublisherType

// Config
type tomlAgent struct {
    Name        string
}
type tomlMothership struct {
    Host string
    Port int
}

type Event struct {
    Timestamp time.Time `json:"@timestamp"`
    Agent string `json:"agent"`
    Type string `json:"type"`
    Src_ip string `json:"src_ip"`
    Src_port uint16 `json:"src_port"`
    Dst_ip string `json:"dst_ip"`
    Dst_port uint16 `json:"dst_port"`
    ResponseTime int32 `json:"responsetime"`
    Status string `json:"status"`
    RequestRaw string `json:"request_raw"`
    ResponseRaw string `json:"response_raw"`

    Mysql bson.M `json:"mysql"`
    Http bson.M `json:"http"`
    Redis bson.M `json:"redis"`
}


func (publisher *PublisherType) PublishHttpTransaction(t *HttpTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())
    src_ip := Ipv4_Ntoa(t.tuple.Src_ip)
    dst_ip := Ipv4_Ntoa(t.tuple.Dst_ip)

    status := t.Http["response"].(bson.M)["phrase"].(string)

    _, err := core.Index(true, index, "http","", Event{
        t.ts, "tiny", "http", src_ip, t.tuple.Src_port,
        dst_ip, t.tuple.Dst_port, t.ResponseTime,
        status, t.Request_raw, t.Response_raw,
        nil, t.Http, nil})

    DEBUG("publish", "Sent Http transaction:\n%s", t.Http)
    return err

}

func (publisher *PublisherType) PublishMysqlTransaction(t *MysqlTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())
    src_ip := Ipv4_Ntoa(t.tuple.Src_ip)
    dst_ip := Ipv4_Ntoa(t.tuple.Dst_ip)

    status := t.Mysql["error_message"].(string)
    if len(status) == 0 {
        status = "OK"
    }

    _, err := core.Index(true, index, "mysql", "", Event{
        t.ts, "tiny", "mysql", src_ip, t.tuple.Src_port,
        dst_ip,t.tuple.Dst_port, t.ResponseTime,
        status, t.Request_raw, t.Response_raw,
        t.Mysql, nil, nil})

    DEBUG("publish", "Sent MySQL transaction:\n%s", t.Mysql)
    return err

}

func (publisher *PublisherType) PublishRedisTransaction(t *RedisTransaction) error {
    // Set the Elasticsearch Host to Connect to
    api.Domain = publisher.mother_host
    api.Port = publisher.mother_port

    // add single go struct entity
    index := fmt.Sprintf("packetbeat-%d.%02d.%02d", t.ts.Year(), t.ts.Month(), t.ts.Day())
    src_ip := Ipv4_Ntoa(t.tuple.Src_ip)
    dst_ip := Ipv4_Ntoa(t.tuple.Dst_ip)

    status := "OK"

    _, err := core.Index(true, index, "redis","", Event{
        t.ts, "tiny", "redis", src_ip, t.tuple.Src_port,
        dst_ip, t.tuple.Dst_port, t.ResponseTime,
        status, t.Request_raw, t.Response_raw,
        nil, nil, t.Redis})

    DEBUG("publish", "Sent Redis transaction:\n%s", t.Redis)
    return err

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

    return nil
}
