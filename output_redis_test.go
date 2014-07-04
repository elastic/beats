package main

import (
    "github.com/garyburd/redigo/redis"
    "testing"
)

func TestTopologyInRedis(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
    }

    c, err := redis.Dial("tcp", "127.0.0.1:6379")
    if err != nil {
        t.Error("Fail to connect to the Redis server 127.0.0.1:6379")
        return
    }
    defer c.Close()

    var redisoutput1 RedisOutputType = RedisOutputType{Index: "packetbeat", Conn: c, TopologyExpire: 15}

    var publisher1 PublisherType = PublisherType{name: "proxy1"}
    var publisher2 PublisherType = PublisherType{name: "proxy2"}
    var publisher3 PublisherType = PublisherType{name: "proxy3"}

    publisher1.Output = append(publisher1.Output, OutputInterface(&redisoutput1))
    publisher2.Output = append(publisher2.Output, OutputInterface(&redisoutput1))
    publisher3.Output = append(publisher3.Output, OutputInterface(&redisoutput1))

    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9", "fe80::4e8d:79ff:fef2:de6a")
    publisher3.PublishTopology("10.1.0.10")

    name2 := publisher1.GetServerName("10.1.0.9")
    if name2 != "proxy2" {
        t.Error("Failed to update proxy2 in topology: name=%s", name2)
    }

    name2 = publisher3.GetServerName("10.1.0.9")
    if name2 != "proxy2" {
        t.Error("Failed to update proxy2 in topology: name=%s", name2)
    }

    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9")
    publisher3.PublishTopology("192.168.1.2")

    name3 := publisher1.GetServerName("192.168.1.2")
    if name3 != "proxy3" {
        t.Error("Failed to add a new IP")
    }

    name3 = publisher1.GetServerName("10.1.0.10")
    if name3 != "" {
        t.Error("Failed to delete old IP of proxy3: %s", name3)
    }

    name2 = publisher1.GetServerName("fe80::4e8d:79ff:fef2:de6a")
    if name2 != "" {
        t.Error("Failed to delete old IP of proxy2: %s", name2)
    }
}
