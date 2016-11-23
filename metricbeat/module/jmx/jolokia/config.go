package jolokia

import (
	"encoding/json"
	"fmt"
	"strings"
)

type MetricSetConfigInput struct {
	Host        string `yaml:"host"`
	Mapping     string `yaml:"mapping"`
	Application string `yaml:"application"`
	Instance    string `yaml:"instance"`
}

type MetricSetConfig struct {
	Url         string
	Body        string
	Mapping     map[string]string
	Application string
	Instance    string
}

type SliceSet map[string][]string

func (s SliceSet) Add(key, value string) {
	_, ok := s[key]
	if !ok {
		s[key] = make([]string, 0, 30)
	}
	s[key] = append(s[key], value)
}

//parse MetricSetConfigRaw to MetricSetConfig
func parseConfig(metricSetConfigRaw []MetricSetConfigInput) ([]MetricSetConfig, error) {
	var metricSetConfig []MetricSetConfig
	if len(metricSetConfigRaw) == 0 {
		return nil, fmt.Errorf("The jolokia module config is empty!")
	}

	for _, currConfig := range metricSetConfigRaw {
		var responseMapping = make(map[string]string)
		err := json.Unmarshal([]byte(currConfig.Mapping), &responseMapping)
		if err != nil {
			return nil, fmt.Errorf("Cannot unmarshal json mapping: %s", err)
		}
		currBody, err := buildRequestBody(responseMapping)
		if err != nil {
			return nil, fmt.Errorf("Cannot build request body from json mapping: %s", err)
		}
		currUrl := "http://" + currConfig.Host + "/jolokia/?ignoreErrors=true&canonicalNaming=false"
		metricSetConfig = append(metricSetConfig, MetricSetConfig{currUrl, currBody,
			responseMapping, currConfig.Application, currConfig.Instance})
	}
	debugf("Jolokia request config will be included: %#v", metricSetConfig)

	return metricSetConfig, nil
}

func buildRequestBody(mapping map[string]string) (string, error) {
	var requestBodyStructure = make(SliceSet)
	for k, _ := range mapping {
		mbeanAttributePair := strings.Split(k, ":::")
		if len(mbeanAttributePair) != 2 {
			return "", fmt.Errorf("Cannot parse mbean/attribute pair: %s", k)
		}
		requestBodyStructure.Add(mbeanAttributePair[0], mbeanAttributePair[1])
	}
	return marshalJSONRequest(requestBodyStructure), nil
}

func marshalJSONRequest(this SliceSet) string {
	result := "["
	for mbean, attributes := range this {
		singleRequest := "{\"type\":\"read\",\"mbean\":\"" + mbean + "\",\"attribute\":["
		for _, attribute := range attributes {
			singleRequest = singleRequest + "\"" + attribute + "\","
		}
		result = result + strings.TrimRight(singleRequest, ",") + "]},"
	}
	result = strings.TrimRight(result, ",") + "]"

	debugf("Marshalled JSON: %s", result)
	return result
}
