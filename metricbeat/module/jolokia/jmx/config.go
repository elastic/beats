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
	Event string
}

// Target inputs the value you want to set for jolokia target block
type Target struct {
	URL      string
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
//       ],
//       "target":{
//          "url":"service:jmx:rmi:///jndi/rmi://targethost:9999/jmxrmi",
//          "user":"jolokia",
//          "password":"s!cr!t"
//       }
//    }
// ]
type RequestBlock struct {
	Type      string                 `json:"type"`
	MBean     string                 `json:"mbean"`
	Attribute []string               `json:"attribute"`
	Config    map[string]interface{} `json:"config"`
	Target    *TargetBlock           `json:"target,omitempty"`
}

// TargetBlock is used to build the target blocks of the following format into RequestBlock.
//
// "target":{
//    "url":"service:jmx:rmi:///jndi/rmi://targethost:9999/jmxrmi",
//    "user":"jolokia",
//    "password":"s!cr!t"
// }
type TargetBlock struct {
	URL      string `json:"url"`
	User     string `json:"user,omitempty"`
	Password string `json:"password,omitempty"`
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

	config := map[string]interface{}{
		"ignoreErrors":    true,
		"canonicalNaming": false,
	}
	for _, mapping := range mappings {
		rb := RequestBlock{
			Type:   "read",
			MBean:  mapping.MBean,
			Config: config,
		}

		if len(mapping.Target.URL) != 0 {
			rb.Target = new(TargetBlock)
			rb.Target.URL = mapping.Target.URL
			rb.Target.User = mapping.Target.User
			rb.Target.Password = mapping.Target.Password
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
