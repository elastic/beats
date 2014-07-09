package main

import (
    "testing"
    "time"
    "fmt"
)

const elasticsearchAddr = "localhost"
const elasticsearchPort = 9200

func createElasticsearchConnection() ElasticsearchOutputType {

    var elasticsearchOutput ElasticsearchOutputType
    elasticsearchOutput.Init(tomlMothership{
        Enabled: true,
        Save_topology: true,
        Host: elasticsearchAddr,
        Port: elasticsearchPort,
        Username: "",
        Password: "",
        Path: "",
        Index: "packetbeat",
        Protocol: "",
    })

    return elasticsearchOutput
}

func TestTopologyInES(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
    }

    LogInit(LOG_DEBUG, "" /*!toSyslog*/, true, []string{})

    var publisher1 PublisherType = PublisherType{name: "proxy1"}
    var publisher2 PublisherType = PublisherType{name: "proxy2"}
    var publisher3 PublisherType = PublisherType{name: "proxy3"}

    elasticsearchOutput1 := createElasticsearchConnection()
    elasticsearchOutput2 := createElasticsearchConnection()
    elasticsearchOutput3 := createElasticsearchConnection()

    publisher1.TopologyOutput = OutputInterface(&elasticsearchOutput1)
    publisher2.TopologyOutput = OutputInterface(&elasticsearchOutput2)
    publisher3.TopologyOutput = OutputInterface(&elasticsearchOutput3)

    publisher1.PublishTopology("10.1.0.4")
    publisher2.PublishTopology("10.1.0.9", "fe80::4e8d:79ff:fef2:de6a")
    publisher3.PublishTopology("10.1.0.10")

    // give some time to Elasticsearch to add the IPs
    time.Sleep(1 * time.Second)

    elasticsearchOutput3.UpdateLocalTopologyMap()
    fmt.Println(elasticsearchOutput3.TopologyMap)

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

    // give some time to Elasticsearch to add the IPs
    time.Sleep(1 * time.Second)

    elasticsearchOutput3.UpdateLocalTopologyMap()

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


