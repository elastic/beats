package jmx

import "encoding/json"

type JMXMapping struct {
	MBean      string
	Attributes []Attribute
	Target     Target
}

type Attribute struct {
	Attr  string
	Field string
}

type Target struct {
	Url      string
	User     string
	Password string
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
	Type      string       `json:"type"`
	MBean     string       `json:"mbean"`
	Attribute []string     `json:"attribute"`
	Target    *TargetBlock `json:"target,omitempty"`
}

type TargetBlock struct {
	Url      string `json:"url"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
}

func buildRequestBodyAndMapping(mappings []JMXMapping) ([]byte, map[string]string, error) {
	responseMapping := map[string]string{}
	var blocks []RequestBlock

	for _, mapping := range mappings {
		rb := RequestBlock{
			Type:  "read",
			MBean: mapping.MBean,
		}

		if len(mapping.Target.Url) != 0 {
			rb.Target = new(TargetBlock)
			rb.Target.Url = mapping.Target.Url
			rb.Target.User = mapping.Target.User
			rb.Target.Password = mapping.Target.Password
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
