package jolokia

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfigParser(t *testing.T) {
	metricSetupOK := []MetricSetup {
		{"java.lang:type=Runtime",
			[]Attribute {{"Uptime", "uptime"}}},
		{"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep",
			[]Attribute {{"CollectionTime","gc.cms_collection_time"},{"CollectionCount","gc.cms_collection_count"}},
		},
	}

	metricSetupWithQuotes := []MetricSetup {
		{"java.lang:type=SomeWeirdType\"1\"",
			[]Attribute {{"A \"strange\" value","special_value"}}},
	}
	jolokiaConfigInput := []MetricSetConfigInput{
		{"localhost:4008", metricSetupOK, "application1", "instance1"},
		{"localhost:4002", metricSetupWithQuotes, "", ""},
	}

	jolokiaConfig, err := parseConfig(jolokiaConfigInput)
	assert.Nil(t, err)

	mappingConfigOK := map[string]string{
		"java.lang:type=Runtime_Uptime": "uptime",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionTime": "gc.cms_collection_time",
		"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep_CollectionCount": "gc.cms_collection_count",
	}
	mappingConfigSpecial := map[string]string{
		"java.lang:type=SomeWeirdType\"1\"_A \"strange\" value": "special_value",
	}
	var expectedBodyOK []Entry
	json.Unmarshal([]byte("[{\"type\":\"read\",\"mbean\":\"java.lang:type=Runtime\",\"attribute\":[\"Uptime\"]},"+
		"{\"type\":\"read\",\"mbean\":\"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep\","+
		"\"attribute\":[\"CollectionTime\",\"CollectionCount\"]}]"), &expectedBodyOK)

	var expectedBodySpecial []Entry
	json.Unmarshal([]byte("[{\"type\":\"read\",\"mbean\":\"java.lang:type=SomeWeirdType\\\"1\\\"\"," +
		"\"attribute\":[\"A \\\"strange\\\" value\"]}]"),
		&expectedBodySpecial)

	var actualBodyOK []Entry
	err = json.Unmarshal([]byte(jolokiaConfig[0].Body), &actualBodyOK)
	assert.Nil(t, err)

	var actualBodySpecial []Entry
	err = json.Unmarshal([]byte(jolokiaConfig[1].Body), &actualBodySpecial)
	assert.Nil(t, err)

	assert.Equal(t, 2, len(jolokiaConfig))
	assert.Equal(t, "http://localhost:4008/jolokia/?ignoreErrors=true&canonicalNaming=false", jolokiaConfig[0].Url)
	assert.Equal(t, "instance1", jolokiaConfig[0].Instance)
	assert.Equal(t, "application1", jolokiaConfig[0].Application)
	assert.Equal(t, mappingConfigOK, jolokiaConfig[0].Mapping)
	assert.Equal(t, expectedBodyOK, actualBodyOK)

	assert.Equal(t, "http://localhost:4002/jolokia/?ignoreErrors=true&canonicalNaming=false", jolokiaConfig[1].Url)
	assert.Equal(t, "", jolokiaConfig[1].Instance)
	assert.Equal(t, "", jolokiaConfig[1].Application)
	assert.Equal(t, mappingConfigSpecial, jolokiaConfig[1].Mapping)
	assert.Equal(t, expectedBodySpecial, actualBodySpecial)

}
