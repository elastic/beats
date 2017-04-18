package jmx

import "encoding/json"

type JMXMapping struct {
	MBean      string
	Attributes []Attribute
}

type Attribute struct {
	Attr  string
	Field string
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

func buildRequestBodyAndMapping(mappings []JMXMapping) ([]byte, map[string]string, error) {

	responseMapping := map[string]string{}
	blocks := []RequestBlock{}

	for _, mapping := range mappings {

		rb := RequestBlock{
			Type:  "read",
			MBean: mapping.MBean,
		}

		for _, attribute := range mapping.Attributes {
			rb.Attribute = append(rb.Attribute, attribute.Attr)
			responseMapping[mapping.MBean+"_"+attribute.Attr] = attribute.Field
		}
		blocks = append(blocks, rb)
	}

	content, err := json.Marshal(blocks)
	return content, responseMapping, err
}
