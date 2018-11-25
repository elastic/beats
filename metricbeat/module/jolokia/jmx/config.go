// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package jmx

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

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

// MBeanName is an internal struct used to store
// the information by the parsed ```mbean``` (bean name) configuration
// field in ```jmx.mappings```.
type MBeanName struct {
	Domain     string
	Properties map[string]string
}

// Canonicalize Returns the canonical form of the name; that is, a string representation where the
// properties are sorted in lexical order.
// The canonical form of the name is a String consisting of the domain part,
// a colon (:), the canonical key property list, and a pattern indication.
//
// For more information refer to Java 8 [getCanonicalName()](https://docs.oracle.com/javase/8/docs/api/javax/management/ObjectName.html#getCanonicalName--)
// method.
//
// Set "escape" parameter to true if you want to use the canonicalized name for a Jolokia HTTP GET request, false otherwise.
func (m *MBeanName) Canonicalize(escape bool) string {

	var propertySlice []string

	r2 := regexp.MustCompile(`(["]|[.]|[!]|[\/])`)

	for key, value := range m.Properties {

		tmpVal := value
		if escape {
			tmpVal = r2.ReplaceAllString(value, "!$1")
		}

		propertySlice = append(propertySlice, key+"="+tmpVal)
	}

	sort.Strings(propertySlice)

	return m.Domain + ":" + strings.Join(propertySlice, ",")
}

// ParseMBeanName is a factory function which parses a Managed Bean name string
// identified by mBeanName and returns a new MBean object which
// contains all the information, i.e. domain and properties of the MBean.
//
// The Mbean string has to abide by the rules which are imposed by Java.
// For more info: https://docs.oracle.com/javase/8/docs/api/javax/management/ObjectName.html#getCanonicalName--
func ParseMBeanName(mBeanName string) (*MBeanName, error) {

	// Split mbean string in two parts: the bean domain and the properties
	parts := strings.SplitN(mBeanName, ":", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return nil, fmt.Errorf("domain and properties needed in mbean name: %s", mBeanName)
	}

	// Create a new MBean object
	mybean := &MBeanName{
		Domain: parts[0],
	}

	// First of all verify that all bean properties are
	// in the form key=value
	tmpProps := propertyRegexp.FindAllString(parts[1], -1)
	propertyList := strings.Join(tmpProps, ",")
	if len(propertyList) != len(parts[1]) {
		// Some property didn't match
		return nil, fmt.Errorf("mbean properties must be in the form key=value: %s", mBeanName)
	}

	var mbeanRegexp = regexp.MustCompile("([^,=:*?]+)=([^,=:\"]+|\".*\")")

	// Using this regexp we will split the properties in a 2 dimensional array
	// instead of just splitting by commas because values can be quoted
	// and contain commas, what complicates the parsing.
	// For example this MBean property string:
	//
	// name=HttpRequest1,type=RequestProcessor,worker="http-nio-8080"
	//
	// will become:
	//
	// [][]string{
	// 	[]string{"name=HttpRequest1", "name", "HttpRequest1"},
	// 	[]string{"type=RequestProcessor", "type", "RequestProcessor"},
	// 	[]string{"worker=\"http-nio-8080\"", "worker", "\"http-nio-8080\""}
	// }
	properties := mbeanRegexp.FindAllStringSubmatch(parts[1], -1)

	// If we could not parse MBean properties
	if properties == nil {
		return nil, fmt.Errorf("mbean properties must be in the form key=value: %s", mBeanName)
	}

	// Initialise properties map
	mybean.Properties = make(map[string]string)

	for _, prop := range properties {

		// If every row does not have 3 columns, then
		// parsing must have failed.
		if (prop == nil) || (len(prop) < 3) {
			// Some property didn't match
			return nil, fmt.Errorf("mbean properties must be in the form key=value: %s", mBeanName)
		}

		mybean.Properties[prop[1]] = prop[2]
	}

	return mybean, nil
}

// JolokiaHTTPRequest is a small struct which contains all request information
// needed to construct a reqest helper.HTTP object which will be sent to Jolokia.
// It is just an intermediary structure which can be easily tested as helper.HTTP
// fields are all private.
type JolokiaHTTPRequest struct {
	// HttpMethod can be either "GET" or "POST"
	HTTPMethod string
	// URI which will be used to query Jolokia
	URI string
	// Request body which is only filled if the http method is "POST"
	Body []byte
}

// JolokiaHTTPRequestBuilder is an interface which describes
// the behaviour of the builder which generates the HTTP request
// which is sent to Jolokia
type JolokiaHTTPRequestBuilder interface {
	// Fetches the information from Jolokia server regarding MBeans
	BuildRequestsAndMappings(configMappings []JMXMapping) ([]*JolokiaHTTPRequest, AttributeMapping, error)
}

// JolokiaHTTPGetBuilder constructs an HTTP GET request
// which will read MBean information from Jolokia
type JolokiaHTTPGetBuilder struct {
}

// BuildRequestsAndMappings generates HTTP GET request
// such as URI,Body.
func (pc *JolokiaHTTPGetBuilder) BuildRequestsAndMappings(configMappings []JMXMapping) ([]*JolokiaHTTPRequest, AttributeMapping, error) {

	// Create Jolokia URLs
	uris, responseMapping, err := pc.buildGetRequestURIs(configMappings)
	if err != nil {
		return nil, nil, err
	}

	// Create one or more HTTP GET requests
	var httpRequests []*JolokiaHTTPRequest
	for _, i := range uris {
		http := &JolokiaHTTPRequest{
			HTTPMethod: "GET",
			URI:        i,
		}

		httpRequests = append(httpRequests, http)
	}

	return httpRequests, responseMapping, err
}

// Builds a GET URI which will have the following format:
//
// /read/<mbean>/<attribute>/[path]?ignoreErrors=true&canonicalNaming=false
func (pc *JolokiaHTTPGetBuilder) buildJolokiaGETUri(mbean string, attr []Attribute) string {
	initialURI := "/read/%s?ignoreErrors=true&canonicalNaming=false"

	var attrList []string
	for _, attribute := range attr {
		attrList = append(attrList, attribute.Attr)
	}

	tmpURL := mbean + "/" + strings.Join(attrList, ",")

	tmpURL = fmt.Sprintf(initialURI, tmpURL)

	return tmpURL
}

func (pc *JolokiaHTTPGetBuilder) mBeanAttributeHasField(attr *Attribute) bool {

	if attr.Field != "" && (strings.Trim(attr.Field, " ") != "") {
		return true
	}

	return false
}

func (pc *JolokiaHTTPGetBuilder) buildGetRequestURIs(mappings []JMXMapping) ([]string, AttributeMapping, error) {

	responseMapping := make(AttributeMapping)
	var urls []string

	// At least Jolokia 1.5 responses with canonicalized MBean names when using
	// wildcards, even when canonicalNaming is set to false, this makes mappings to fail.
	// So use canonicalized names everywhere.
	// If Jolokia returns non-canonicalized MBean names, then we'll need to canonicalize
	// them or change our approach to mappings.

	for _, mapping := range mappings {
		mbean, err := ParseMBeanName(mapping.MBean)
		if err != nil {
			return urls, nil, err
		}

		if len(mapping.Target.URL) != 0 {
			err := errors.New("Proxy requests are only valid when using POST method")
			return urls, nil, err
		}

		// For every attribute we will build a response mapping
		for _, attribute := range mapping.Attributes {
			responseMapping[attributeMappingKey{mbean.Canonicalize(true), attribute.Attr}] = attribute
		}

		// Build a new URI for all attributes
		urls = append(urls, pc.buildJolokiaGETUri(mbean.Canonicalize(true), mapping.Attributes))

	}

	return urls, responseMapping, nil
}

// JolokiaHTTPPostBuilder constructs an HTTP GET request
// which will read MBean information from Jolokia
type JolokiaHTTPPostBuilder struct {
}

// BuildRequestsAndMappings generates HTTP POST request
// such as URI,Body.
func (pc *JolokiaHTTPPostBuilder) BuildRequestsAndMappings(configMappings []JMXMapping) ([]*JolokiaHTTPRequest, AttributeMapping, error) {

	body, mapping, err := pc.buildRequestBodyAndMapping(configMappings)
	if err != nil {
		return nil, nil, err
	}

	http := &JolokiaHTTPRequest{
		HTTPMethod: "POST",
		Body:       body,
	}

	// Create an array with only one HTTP POST request
	httpRequests := []*JolokiaHTTPRequest{http}

	return httpRequests, mapping, nil
}

// Parse strings with properties with the format key=value, being:
// - key a nonempty string of characters which may not contain any of the characters,
//   comma (,), equals (=), colon, asterisk, or question mark.
// - value a string that can be quoted or unquoted, if unquoted it cannot be empty and
//   cannot contain any of the characters comma, equals, colon, or quote.
var propertyRegexp = regexp.MustCompile("[^,=:*?]+=([^,=:\"]+|\".*\")")

func (pc *JolokiaHTTPPostBuilder) buildRequestBodyAndMapping(mappings []JMXMapping) ([]byte, AttributeMapping, error) {
	responseMapping := make(AttributeMapping)
	var blocks []RequestBlock

	// At least Jolokia 1.5 responses with canonicalized MBean names when using
	// wildcards, even when canonicalNaming is set to false, this makes mappings to fail.
	// So use canonicalized names everywhere.
	// If Jolokia returns non-canonicalized MBean names, then we'll need to canonicalize
	// them or change our approach to mappings.
	config := map[string]interface{}{
		"ignoreErrors":    true,
		"canonicalNaming": true,
	}
	for _, mapping := range mappings {
		mbeanObj, err := ParseMBeanName(mapping.MBean)
		if err != nil {
			return nil, nil, err
		}

		mbean := mbeanObj.Canonicalize(false)

		rb := RequestBlock{
			Type:   "read",
			MBean:  mbean,
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
			responseMapping[attributeMappingKey{mbean, attribute.Attr}] = attribute
		}
		blocks = append(blocks, rb)
	}

	content, err := json.Marshal(blocks)
	return content, responseMapping, err
}

// NewJolokiaHTTPRequestBuiler is a factory method which creates and returns an implementation
// class of NewJolokiaHTTPRequestBuiler interface. HTTP GET and POST are currently supported.
func NewJolokiaHTTPRequestBuiler(httpMethod string) JolokiaHTTPRequestBuilder {

	if httpMethod == "GET" {
		return &JolokiaHTTPGetBuilder{}
	}

	return &JolokiaHTTPPostBuilder{}

}
