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

    TopologyExpire time.Duration
    ReconnectInterval time.Duration
    Hostname        string
    Password        string
    Db              int
    DbTopology      int
    Timeout         time.Duration

    TopologyMap map[string]string
}

var RedisOutput RedisOutputType

func (out *RedisOutputType) Init(config tomlMothership) error {

    out.Hostname = fmt.Sprintf("%s:%d", config.Host, config.Port)

    if config.Password != "" {
        out.Password = config.Password
    }

    if config.Db != 0 {
        out.Db = config.Db
    }

    out.DbTopology = 1
    if config.Db_topology != 0 {
        out.DbTopology = config.Db_topology
    }

    out.Timeout = time.Duration(5) * time.Second
    if config.Timeout != 0 {
        out.Timeout = time.Duration(config.Timeout) * time.Second
    }

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

    INFO("[RedisOutput] Using Redis server %s", out.Hostname)
    if out.Password != "" {
        INFO("[RedisOutput] Using password to connect to Redis")
    }
    INFO("[RedisOutput] Redis connection timeout %s", out.Timeout)
    INFO("[RedisOutput] Using index pattern [%s-]YYYY.MM.DD", out.Index)
    INFO("[RedisOutput] Topology expires after %s", out.TopologyExpire)
    INFO("[RedisOutput] Using db %d for storing events", out.Db)
    INFO("[RedisOutput] Using db %d for storing topology", out.DbTopology)

    // connect to db
    client := redis.NewTCPClient(&redis.Options{
        Addr:   out.Hostname,
        Password: out.Password,
        DB: int64(out.Db),
        DialTimeout: out.Timeout,
    })

    _, err := client.Ping().Result()
    if err != nil {
        ERR("Failed connection to Redis. ping returns an error: %s", err)
        return err
    }
    out.Client = client

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

    // connect to db
     client := redis.NewTCPClient(&redis.Options{
        Addr:   out.Hostname,
        Password: out.Password,
        DialTimeout: out.Timeout,
    })
    client.Select(int64(out.DbTopology))

    defer client.Close()


    _, err := client.HSet(name, "ipaddrs", strings.Join(localAddrs, ",")).Result()
    if err != nil {
        ERR("[%s] Fail to set the IP addresses: %s", name, err)
        return err
    }

    _, err = client.Expire(name, out.TopologyExpire).Result()
    if err != nil {
        ERR("[%s] Fail to set the expiration time: %s", name, err)
        return err
    }

    out.UpdateLocalTopologyMap(client)

    return nil
}

func (out *RedisOutputType) UpdateLocalTopologyMap(client *redis.Client) {

    TopologyMapTmp := make(map[string]string)

    res, err := client.Keys("*").Result()
    if err != nil {
        ERR("Fail to get the all agents from the topology map %s", err)
        return
    }
    for _, hostname := range res {
        res, err := client.HGet(hostname, "ipaddrs").Result()
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
