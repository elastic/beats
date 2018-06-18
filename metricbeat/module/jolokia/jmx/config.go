package jmx

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

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
	Type      string                 `json:"type"`
	MBean     string                 `json:"mbean"`
	Attribute []string               `json:"attribute"`
	Config    map[string]interface{} `json:"config"`
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

// Parse strings with properties with the format key=value, being:
// - key a nonempty string of characters which may not contain any of the characters,
//   comma (,), equals (=), colon, asterisk, or question mark.
// - value a string that can be quoted or unquoted, if unquoted it cannot be empty and
//   cannot contain any of the characters comma, equals, colon, or quote.
var propertyRegexp = regexp.MustCompile("[^,=:*?]+=([^,=:\"]+|\".*\")")

func canonicalizeMBeanName(name string) (string, error) {
	// From https://docs.oracle.com/javase/8/docs/api/javax/management/ObjectName.html#getCanonicalName--
	//
	//   Returns the canonical form of the name; that is, a string representation where the
	//   properties are sorted in lexical order.
	//   The canonical form of the name is a String consisting of the domain part,
	//   a colon (:), the canonical key property list, and a pattern indication.
	//
	parts := strings.SplitN(name, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return name, fmt.Errorf("domain and properties needed in mbean name: %s", name)
	}
	domain := parts[0]

	// Using this regexp instead of just splitting by commas because values can be quoted
	// and contain commas, what complicates the parsing.
	properties := propertyRegexp.FindAllString(parts[1], -1)
	propertyList := strings.Join(properties, ",")
	if len(propertyList) != len(parts[1]) {
		// Some property didn't match
		return name, fmt.Errorf("mbean properties must be in the form key=value: %s", name)
	}

	sort.Strings(properties)
	return domain + ":" + strings.Join(properties, ","), nil
}

func buildRequestBodyAndMapping(mappings []JMXMapping) ([]byte, AttributeMapping, error) {
	responseMapping := make(AttributeMapping)
	var blocks []RequestBlock

	// At least Jolokia 1.5 responses with canonicalized MBean names when using
	// wildcards, even when canonicalNaming is set to false, this makes mappings to fail.
	// So use canonicalzed names everywhere.
	// If Jolokia returns non-canonicalized MBean names, then we'll need to canonicalize
	// them or change our approach to mappings.
	config := map[string]interface{}{
		"ignoreErrors":    true,
		"canonicalNaming": true,
	}
	for _, mapping := range mappings {
		mbean, err := canonicalizeMBeanName(mapping.MBean)
		if err != nil {
			return nil, nil, err
		}
		rb := RequestBlock{
			Type:   "read",
			MBean:  mbean,
			Config: config,
		}

		for _, attribute := range mapping.Attributes {
			rb.Attribute = append(rb.Attribute, attribute.Attr)
			responseMapping[attributeMappingKey{mbean, attribute.Attr}] = attribute
		}
		blocks = append(blocks, rb)
	}

	content, err := json.Marshal(blocks)
	return content, responseMapping, err
}
