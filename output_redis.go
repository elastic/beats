package main

import (
    "encoding/json"
    "fmt"
    "github.com/garyburd/redigo/redis"
    "strings"
)

type RedisOutputType struct {
    OutputInterface
    Index          string
    Conn           redis.Conn
    TopologyExpire int

    TopologyMap map[string]string
}

var RedisOutput RedisOutputType

func (out *RedisOutputType) Init(config tomlMothership) error {

    hostname := fmt.Sprintf("%s:%d", config.Host, config.Port)
    c, err := redis.Dial("tcp", hostname)
    if err != nil {
        ERR("Fail to connect to Redis server %s", hostname)
        return err
    }
    out.Conn = c
    //defer c.Close()

    if config.Index != "" {
        out.Index = config.Index
    } else {
        out.Index = "packetbeat"
    }

    out.TopologyExpire = 15
    if _Config.Agent.Topology_expire != 0 {
        out.TopologyExpire = _Config.Agent.Topology_expire
    }

    INFO("[RedisOutput] Using Redis server %s", hostname)
    INFO("[RedisOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
    INFO("[RedisOutput] Topology expires after %d seconds", out.TopologyExpire)

    return nil
}

func (out *RedisOutputType) GetNameByIP(ip string) string {
    name, exists := out.TopologyMap[ip]
    if !exists {
        return ""
    }
    return name
}

func (out *RedisOutputType) PublishIPs(name string, localAddrs []string) error {

    DEBUG("output_redis", "[%s] Publish the IPs %s", name, localAddrs)

    _, err := out.Conn.Do("SELECT", 1)
    if err != nil {
        ERR("[%s] Fail to select redis database: %s", name, err)
        return err
    }

    _, err = out.Conn.Do("HSET", name, "ipaddrs", strings.Join(localAddrs, ","))
    if err != nil {
        ERR("[%s] Fail to set the IP addresses: %s", name, err)
        return err
    }

    _, err = out.Conn.Do("EXPIRE", name, out.TopologyExpire)
    if err != nil {
        ERR("[%s] Fail to set the expiration time: %s", name, err)
        return err
    }

    out.UpdateLocalTopologyMap()

    return nil
}

func (out *RedisOutputType) UpdateLocalTopologyMap() {

    TopologyMapTmp := make(map[string]string)
    out.Conn.Do("SELECT", 1)

    res, err := redis.Strings(out.Conn.Do("KEYS", "*"))
    if err != nil {
        ERR("Fail to get the all agents from the topology map %s", err)
        return
    }
    for _, hostname := range res {
        res, err := redis.String(out.Conn.Do("HGET", hostname, "ipaddrs"))
        if err != nil {
            ERR("[%s] Fail to get the IPs: %s", hostname, err)
        } else {
            ipaddrs := strings.Split(res, ",")
            for _, addr := range ipaddrs {
                TopologyMapTmp[addr] = hostname
            }
        }
    }

    out.TopologyMap = TopologyMapTmp

    DEBUG("output_redis", "Topology %s", TopologyMapTmp)
}

func (out *RedisOutputType) PublishEvent(event *Event) error {

    out.Conn.Do("SELECT", 0)

    index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, event.Timestamp.Year(), event.Timestamp.Month(), event.Timestamp.Day())

    json_event, err := json.Marshal(event)
    if err != nil {
        ERR("Fail to convert the event to JSON: %s", err)
        return err
    }

    _, err = out.Conn.Do("RPUSH", index, json_event)
    if err != nil {
        ERR("Fail to publish event to REDIS: %s", err)
        return err
    }
    DEBUG("output_redis", "Publish event")
    return nil
}
