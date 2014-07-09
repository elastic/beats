package main

import (
    "testing"
    "time"
)

const redisAddr = ":6379"

func TestTopologyInRedis(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
    }
    LogInit(LOG_DEBUG, "" /*!toSyslog*/, true, []string{})


    var publisher1 PublisherType = PublisherType{name: "proxy1"}
    var publisher2 PublisherType = PublisherType{name: "proxy2"}
    var publisher3 PublisherType = PublisherType{name: "proxy3"}

    var redisOutput1 = RedisOutputType{
        Index: "packetbeat",
        Hostname: redisAddr,
        Password: "",
        DbTopology: 1,
        Timeout: time.Duration(5) * time.Second,
        TopologyExpire: time.Duration(15) * time.Second,
    }

    var redisOutput2 = RedisOutputType{
        Index: "packetbeat",
        Hostname: redisAddr,
        Password: "",
        DbTopology: 1,
        Timeout: time.Duration(5) * time.Second,
        TopologyExpire: time.Duration(15) * time.Second,
    }

    var redisOutput3 = RedisOutputType{
        Index: "packetbeat",
        Hostname: redisAddr,
        Password: "",
        DbTopology: 1,
        Timeout: time.Duration(5) * time.Second,
        TopologyExpire: time.Duration(15) * time.Second,
    }

    publisher1.TopologyOutput = OutputInterface(&redisOutput1)
    publisher2.TopologyOutput = OutputInterface(&redisOutput2)
    publisher3.TopologyOutput = OutputInterface(&redisOutput3)

    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9", "fe80::4e8d:79ff:fef2:de6a")
    publisher3.PublishTopology("10.1.0.10")

    name2 := publisher3.GetServerName("10.1.0.9")
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

    name3 := publisher3.GetServerName("192.168.1.2")
    if name3 != "proxy3" {
        t.Error("Failed to add a new IP")
    }

    name3 = publisher3.GetServerName("10.1.0.10")
    if name3 != "" {
        t.Error("Failed to delete old IP of proxy3: %s", name3)
    }

    name2 = publisher3.GetServerName("fe80::4e8d:79ff:fef2:de6a")
    if name2 != "" {
        t.Error("Failed to delete old IP of proxy2: %s", name2)
    }
}
