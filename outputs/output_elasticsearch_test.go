package outputs

import (
	"testing"
	"time"
)

const elasticsearchAddr = "localhost"
const elasticsearchPort = 9200

func createElasticsearchConnection() ElasticsearchOutputType {

	var elasticsearchOutput ElasticsearchOutputType
	elasticsearchOutput.Init(tomlMothership{
		Enabled:       true,
		Save_topology: true,
		Host:          elasticsearchAddr,
		Port:          elasticsearchPort,
		Username:      "",
		Password:      "",
		Path:          "",
		Index:         "packetbeat",
		Protocol:      "",
	}, 10)

	return elasticsearchOutput
}

func TestTopologyInES(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping topology tests in short mode, because they require Elasticsearch")
	}

	elasticsearchOutput1 := createElasticsearchConnection()
	elasticsearchOutput2 := createElasticsearchConnection()
	elasticsearchOutput3 := createElasticsearchConnection()

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9",
		"fe80::4e8d:79ff:fef2:de6a"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"10.1.0.10"})

	// give some time to Elasticsearch to add the IPs
	// TODO: just needs _refresh=true instead?
	time.Sleep(1 * time.Second)

	elasticsearchOutput3.UpdateLocalTopologyMap()

	name2 := elasticsearchOutput3.GetNameByIP("10.1.0.9")
	if name2 != "proxy2" {
		t.Errorf("Failed to update proxy2 in topology: name=%s", name2)
	}

	elasticsearchOutput1.PublishIPs("proxy1", []string{"10.1.0.4"})
	elasticsearchOutput2.PublishIPs("proxy2", []string{"10.1.0.9"})
	elasticsearchOutput3.PublishIPs("proxy3", []string{"192.168.1.2"})

	// give some time to Elasticsearch to add the IPs
	time.Sleep(1 * time.Second)

	elasticsearchOutput3.UpdateLocalTopologyMap()

	name3 := elasticsearchOutput3.GetNameByIP("192.168.1.2")
	if name3 != "proxy3" {
		t.Errorf("Failed to add a new IP")
	}

	name3 = elasticsearchOutput3.GetNameByIP("10.1.0.10")
	if name3 != "" {
		t.Errorf("Failed to delete old IP of proxy3: %s", name3)
	}

	name2 = elasticsearchOutput3.GetNameByIP("fe80::4e8d:79ff:fef2:de6a")
	if name2 != "" {
		t.Errorf("Failed to delete old IP of proxy2: %s", name2)
	}
}
