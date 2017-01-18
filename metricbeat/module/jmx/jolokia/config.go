package jolokia

import (
	"fmt"
	"strings"
)

type MetricSetConfigInput struct {
	Host        string `yaml:"host"`
	Mapping     []MetricSetup `yaml:"mapping"`
	Application string `yaml:"application"`
	Instance    string `yaml:"instance"`
}

type MetricSetup struct {
	MBean      string
	Attr       string
	Field      string
	Field_type string
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
func parseConfig(metricSetConfigInput []MetricSetConfigInput) ([]MetricSetConfig, error) {
	var metricSetConfig []MetricSetConfig
	if len(metricSetConfigInput) == 0 {
		return nil, fmt.Errorf("The jolokia module config is empty!")
	}

	for _, currConfig := range metricSetConfigInput {
		currBody, mapping := buildRequestBodyAndMapping(currConfig.Mapping)
		currUrl := "http://" + currConfig.Host + "/jolokia/?ignoreErrors=true&canonicalNaming=false"
		metricSetConfig = append(metricSetConfig, MetricSetConfig{currUrl, currBody,
			mapping, currConfig.Application, currConfig.Instance})
	}
	debugf("Jolokia request config will be included: %#v", metricSetConfig)

	return metricSetConfig, nil
}

func buildRequestBodyAndMapping(mapping []MetricSetup) (string, map[string]string) {
	var requestBodyStructure = make(SliceSet)
	var responseMapping = make(map[string]string)
	for _, metricSetup := range mapping {
		requestBodyStructure.Add(metricSetup.MBean, metricSetup.Attr)
		responseMapping[metricSetup.MBean + "_" + metricSetup.Attr] = metricSetup.Field
	}
	return marshalJSONRequest(requestBodyStructure), responseMapping
}

func marshalJSONRequest(this SliceSet) string {
	result := "["
	for mbean, attributes := range this {
		safeMBeanString := strings.Replace(mbean, "\"", "\\\"", -1)
		singleRequest := "{\"type\":\"read\",\"mbean\":\"" + safeMBeanString + "\",\"attribute\":["
		for _, attribute := range attributes {
			safeAttributeString := strings.Replace(attribute, "\"", "\\\"", -1)
			singleRequest = singleRequest + "\"" + safeAttributeString + "\","
		}
		result = result + strings.TrimRight(singleRequest, ",") + "]},"
	}
	result = strings.TrimRight(result, ",") + "]"

	debugf("Marshalled JSON: %s", result)
	return result
}
