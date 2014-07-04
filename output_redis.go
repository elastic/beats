package main

import (
    "encoding/json"
    "fmt"
    "time"
    "github.com/go-redis/redis"
    "strings"
)

type RedisOutputType struct {
    OutputInterface
    Index          string
    Client         *redis.Client
    TopologyClient         *redis.Client
    TopologyExpire time.Duration

    TopologyMap map[string]string
}

var RedisOutput RedisOutputType

func (out *RedisOutputType) Init(config tomlMothership) error {

    hostname := fmt.Sprintf("%s:%d", config.Host, config.Port)

    // connect to db
    client := redis.NewTCPClient(&redis.Options{
        Addr:   hostname,
        Password: "",
    })

    _, err := client.Ping().Result()
    if err != nil {
        ERR("Failed connection to Redis. ping returns an error: %s", err)
        return err
    }
    out.Client = client


    // connect to db topology
     out.TopologyClient = redis.NewTCPClient(&redis.Options{
        Addr:   hostname,
        Password: "",
    })
    out.TopologyClient.Select(1)


    if config.Index != "" {
        out.Index = config.Index
    } else {
        out.Index = "packetbeat"
    }

    exp_sec := 15
    if _Config.Agent.Topology_expire != 0 {
        exp_sec = _Config.Agent.Topology_expire
    }
    out.TopologyExpire = time.Duration(exp_sec) * time.Second

    INFO("[RedisOutput] Using Redis server %s", hostname)
    INFO("[RedisOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
    INFO("[RedisOutput] Topology expires after %s", out.TopologyExpire)

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

    _, err := out.TopologyClient.HSet(name, "ipaddrs", strings.Join(localAddrs, ",")).Result()
    if err != nil {
        ERR("[%s] Fail to set the IP addresses: %s", name, err)
        return err
    }

    _, err = out.TopologyClient.Expire(name, out.TopologyExpire).Result()
    if err != nil {
        ERR("[%s] Fail to set the expiration time: %s", name, err)
        return err
    }

    out.UpdateLocalTopologyMap()

    return nil
}

func (out *RedisOutputType) UpdateLocalTopologyMap() {

    TopologyMapTmp := make(map[string]string)

    res, err := out.TopologyClient.Keys("*").Result()
    if err != nil {
        ERR("Fail to get the all agents from the topology map %s", err)
        return
    }
    for _, hostname := range res {
        res, err := out.TopologyClient.HGet(hostname, "ipaddrs").Result()
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

    index := fmt.Sprintf("%s-%d.%02d.%02d", out.Index, event.Timestamp.Year(), event.Timestamp.Month(), event.Timestamp.Day())

    json_event, err := json.Marshal(event)
    if err != nil {
        ERR("Fail to convert the event to JSON: %s", err)
        return err
    }

    _, err = out.Client.RPush(index, string(json_event)).Result()
    if err != nil {
        ERR("Fail to publish event to REDIS: %s", err)
        return err
    }
    DEBUG("output_redis", "Publish event")
    return nil
}
