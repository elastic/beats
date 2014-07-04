package main

import (
    "github.com/go-redis/redis"
    "testing"
    "time"
)
const redisAddr = ":6379"

var redisOutput RedisOutputType

func initOutput() {

    redisOutput = RedisOutputType{Index: "packetbeat"}

    redisOutput.Client = redis.NewTCPClient(&redis.Options{
	Addr: redisAddr,
    })

    redisOutput.TopologyClient = redis.NewTCPClient(&redis.Options{
        Addr: redisAddr,
    })
    redisOutput.TopologyExpire = time.Duration(15) * time.Second

}

func closeOutput() {
    redisOutput.Client.Close()
    redisOutput.TopologyClient.Close()
}

func TestTopologyInRedis(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
    }

    initOutput()

    var publisher1 PublisherType = PublisherType{name: "proxy1"}
    var publisher2 PublisherType = PublisherType{name: "proxy2"}
    var publisher3 PublisherType = PublisherType{name: "proxy3"}

    publisher1.Output = append(publisher1.Output, OutputInterface(&redisOutput))
    publisher2.Output = append(publisher2.Output, OutputInterface(&redisOutput))
    publisher3.Output = append(publisher3.Output, OutputInterface(&redisOutput))

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
    closeOutput()
}
