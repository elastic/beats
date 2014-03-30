package main

import (
    "testing"
    "log/syslog"
    "time"
    "github.com/mattbaird/elastigo/api"
    "github.com/mattbaird/elastigo/core"
)

func TestTopology(t *testing.T) {

    LogInit(syslog.LOG_DEBUG, "" /*!toSyslog*/, true, []string{})
    // TODO: delete old topology
    api.Domain = "10.0.50.4"
    api.Port = "9200"

    _, _ = core.Delete("packetbeat-topology", "server-ip", "", nil)
    var publisher1 PublisherType = PublisherType{name: "proxy1", mother_host: api.Domain, mother_port: api.Port, RefreshTopologyTimer: time.Tick(1 * time.Second)}
    var publisher2 PublisherType = PublisherType{name: "proxy2", mother_host: api.Domain, mother_port: api.Port, RefreshTopologyTimer: time.Tick(1 * time.Second)}
    var publisher3 PublisherType = PublisherType{name: "proxy3", mother_host: api.Domain, mother_port: api.Port, RefreshTopologyTimer: time.Tick(1 * time.Second)}


    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9", "fe80::4e8d:79ff:fef2:de6a")
    publisher3.PublishTopology("10.1.0.10")

    time.Sleep(1 * time.Second)

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

    time.Sleep(1 * time.Second)

    name3 := publisher1.GetServerName("192.168.1.2")
    if name3 != "proxy3" {
        t.Error("Failed to add a new IP")
    }

    name3 = publisher1.GetServerName("10.1.0.10")
    if name3 != "" {
        t.Error("Failed to delete old IP of proxy3")
    }

    name2 = publisher1.GetServerName("fe80::4e8d:79ff:fef2:de6a")
    if name3 != "" {
        t.Error("Failed to delete old IP of proxy2")
    }
}
