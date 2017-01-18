package jolokia

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigParser(t *testing.T) {
	metricSetup := []MetricSetup {
		{"java.lang:type=Runtime","Uptime","uptime","integer"},
		{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep","CollectionTime","gc.cms_collection_time","integer"},
		{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep","CollectionCount","gc.cms_collection_count","integer"},
	}
	jolokiaConfigInput := []MetricSetConfigInput{
		{"localhost:4008", metricSetup, "application1", "instance1"},
/*		{"localhost:4002", "{\"org.apache.cassandra.metrics:type=ClientRequest,scope=Read," +
			"name=Latency:::OneMinuteRate\":\"client_request.read_latency_one_min_rate\",\"org.apache
			.cassandra.metrics:type=ClientRequest,scope=Read,name=Latency:::Count\":\"client_request
			.read_latency\",\"org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency:::OneMinuteRate\":\"client_request.write_latency_one_min_rate\",\"org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency:::Count\":\"client_request.write_latency\",\"org.apache.cassandra.metrics:type=Compaction,name=CompletedTasks:::Value\":\"compaction.completed_tasks\",\"org.apache.cassandra.metrics:type=Compaction,name=PendingTasks:::Value\":\"compaction.pending_tasks\"}", "cassandra", ""},*/
	}

	jolokiaConfig, err := parseConfig(jolokiaConfigInput)
	assert.Nil(t, err)

	mappingConfigOne := map[string]string{
		"java.lang:type=Runtime_Uptime": "uptime",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionTime": "gc.cms_collection_time",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionCount": "gc.cms_collection_count",
	}
	//mappingConfigTwo := map[string]string{"org.apache.cassandra.metrics:type=ClientRequest,scope=Read,"name=Latency:::OneMinuteRate": "client_request.read_latency_one_min_rate", "org.apache.cassandra.metrics:type=ClientRequest,scope=Read,name=Latency:::Count": "client_request.read_latency", "org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency:::Count": "client_request.write_latency", "org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency:::OneMinuteRate": "client_request.write_latency_one_min_rate", "org.apache.cassandra.metrics:type=Compaction,name=CompletedTasks:::Value": "compaction.completed_tasks", "org.apache.cassandra.metrics:type=Compaction,name=PendingTasks:::Value": "compaction.pending_tasks"}

	var expectedBodyOne []Entry
	json.Unmarshal([]byte("[{\"type\":\"read\",\"mbean\":\"java.lang:type=Runtime\",\"attribute\":[\"Uptime\"]},"+
		"{\"type\":\"read\",\"mbean\":\"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep\","+
		"\"attribute\":[\"CollectionTime\",\"CollectionCount\"]}]"), &expectedBodyOne)
	/*var expectedBodyTwo []Entry
	json.Unmarshal([]byte("[{\"type\":\"read\",\"mbean\":\"org.apache.cassandra.metrics:type=ClientRequest,"+
		"scope=Read,name=Latency\",\"attribute\":[\"OneMinuteRate\",\"Count\"]},{\"type\":\"read\","+
		"\"mbean\":\"org.apache.cassandra.metrics:type=ClientRequest,scope=Write,name=Latency\",
		\"attribute\":[\"OneMinuteRate\",\"Count\"]},{\"type\":\"read\",\"mbean\":\"org.apache.cassandra
		.metrics:type=Compaction,name=CompletedTasks\"},{\"type\":\"read\",\"mbean\":\"org.apache.cassandra.metrics:type=Compaction,name=PendingTasks\"}]"), &expectedBodyTwo)*/

	var actualBodyOne []Entry
	err = json.Unmarshal([]byte(jolokiaConfig[0].Body), &actualBodyOne)
	assert.Nil(t, err)
/*
	var actualBodyTwo []Entry
	err = json.Unmarshal([]byte(jolokiaConfig[1].Body), &actualBodyTwo)
	assert.Nil(t, err)
*/
	assert.Equal(t, 1, len(jolokiaConfig))
	assert.Equal(t, "http://localhost:4008/jolokia/?ignoreErrors=true&canonicalNaming=false", jolokiaConfig[0].Url)
	assert.Equal(t, "instance1", jolokiaConfig[0].Instance)
	assert.Equal(t, "application1", jolokiaConfig[0].Application)
	assert.Equal(t, mappingConfigOne, jolokiaConfig[0].Mapping)
	assert.Equal(t, expectedBodyOne, actualBodyOne)
/*
	assert.Equal(t, "http://localhost:4002/jolokia/?ignoreErrors=true&canonicalNaming=false", jolokiaConfig[1].Url)
	assert.Equal(t, "", jolokiaConfig[1].Instance)
	assert.Equal(t, "cassandra", jolokiaConfig[1].Application)
	assert.Equal(t, mappingConfigTwo, jolokiaConfig[1].Mapping)
	assert.Equal(t, expectedBodyTwo, actualBodyTwo)
*/
}
