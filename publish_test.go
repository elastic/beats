package main

import (
    "github.com/mattbaird/elastigo/api"
    "github.com/mattbaird/elastigo/core"
    "log/syslog"
    "testing"
    "time"
)

func TestTopology(t *testing.T) {

    api.Domain = "10.0.50.4"
    api.Port = "9200"

    _, _ = core.Delete("packetbeat-topology", "server-ip", "", nil)
    var publisher1 PublisherType = PublisherType{name: "proxy1"}
    var publisher2 PublisherType = PublisherType{name: "proxy2"}
    var publisher3 PublisherType = PublisherType{name: "proxy3"}

    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9", "fe80::4e8d:79ff:fef2:de6a")
    publisher3.PublishTopology("10.1.0.10")

    // give some time to Elasticsearch to add the IPs
    time.Sleep(1 * time.Second)

    publisher1.UpdateTopology()
    publisher2.UpdateTopology()
    publisher3.UpdateTopology()

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

    // give some time to Elasticsearch to add the IPs
    time.Sleep(1 * time.Second)

    publisher1.UpdateTopology()
    publisher2.UpdateTopology()
    publisher3.UpdateTopology()

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

func TestGetServerName(t *testing.T) {

    LogInit(syslog.LOG_DEBUG, "" /*!toSyslog*/, true, []string{})
    // TODO: delete old topology
    api.Domain = "10.0.50.4"
    api.Port = "9200"

    var publisher PublisherType = PublisherType{name: "proxy1", RefreshTopologyTimer: time.Tick(1 * time.Second)}

    name := publisher.GetServerName("127.0.0.1")
    if name != "proxy1" {
        t.Error("GetServerName should return the agent name in case of localhost: %s", name)
    }
}
