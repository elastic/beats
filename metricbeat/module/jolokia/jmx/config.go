package jmx

import "encoding/json"

type JMXMapping struct {
	MBean      string
	Attributes []Attribute
}

type Attribute struct {
	Attr  string
	Field string
	Event string
}

// RequestBlock is used to build the request blocks of the following format:
//
// [
//    {
//       "type":"read",
//       "mbean":"java.lang:type=Runtime",
//       "attribute":[
//          "Uptime"
//       ]
//    },
//    {
//       "type":"read",
//       "mbean":"java.lang:type=GarbageCollector,name=ConcurrentMarkSweep",
//       "attribute":[
//          "CollectionTime",
//          "CollectionCount"
//       ]
//    }
// ]
type RequestBlock struct {
	Type      string   `json:"type"`
	MBean     string   `json:"mbean"`
	Attribute []string `json:"attribute"`
}

type attributeMappingKey struct {
	mbean, attr string
}

// AttributeMapping contains the mapping information between attributes in Jolokia
// responses and fields in metricbeat events
type AttributeMapping map[attributeMappingKey]Attribute

// Get the mapping options for the attribute of an mbean
func (m AttributeMapping) Get(mbean, attr string) (Attribute, bool) {
	a, found := m[attributeMappingKey{mbean, attr}]
	return a, found
}

func buildRequestBodyAndMapping(mappings []JMXMapping) ([]byte, AttributeMapping, error) {
	responseMapping := make(AttributeMapping)
	var blocks []RequestBlock

	for _, mapping := range mappings {
		rb := RequestBlock{
			Type:  "read",
			MBean: mapping.MBean,
		}

		for _, attribute := range mapping.Attributes {
			rb.Attribute = append(rb.Attribute, attribute.Attr)
			responseMapping[attributeMappingKey{mapping.MBean, attribute.Attr}] = attribute
		}
		blocks = append(blocks, rb)
	}

	content, err := json.Marshal(blocks)
	return content, responseMapping, err
}
