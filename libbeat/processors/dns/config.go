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

package dns

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/miekg/dns"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

// config defines the configuration options for the DNS processor.
type config struct {
	cacheConfig  `config:",inline"`
	Nameservers  []string      `config:"nameservers"`              // Required on Windows. /etc/resolv.conf is used if none are given.
	Timeout      time.Duration `config:"timeout"`                  // Per request timeout (with 2 nameservers the total timeout would be 2x).
	Type         queryType     `config:"type" validate:"required"` // One of A, AAAA, TXT or PTR (or reverse).
	Action       fieldAction   `config:"action"`                   // Append or replace (defaults to append) when target exists.
	TagOnFailure []string      `config:"tag_on_failure"`           // Tags to append when a failure occurs.
	Fields       mapstr.M      `config:"fields"`                   // Mapping of source fields to target fields.
	Transport    string        `config:"transport"`                // Can be tls or udp.
	reverseFlat  map[string]string
}

// fieldAction defines the behavior when the target field exists.
type fieldAction uint8

// List of fieldAction types.
const (
	actionAppend fieldAction = iota
	actionReplace
)

var fieldActionNames = map[fieldAction]string{
	actionAppend:  "append",
	actionReplace: "replace",
}

// String returns a field action name.
func (fa fieldAction) String() string {
	name, found := fieldActionNames[fa]
	if found {
		return name
	}
	return "unknown (" + strconv.Itoa(int(fa)) + ")"
}

// Unpack unpacks a string to a fieldAction.
func (fa *fieldAction) Unpack(v string) error {
	switch strings.ToLower(v) {
	case "", "append":
		*fa = actionAppend
	case "replace":
		*fa = actionReplace
	default:
		return fmt.Errorf("invalid dns field action value '%v'", v)
	}
	return nil
}

// queryType represents a DNS query type.
type queryType uint16

const (
	typePTR  = queryType(dns.TypePTR)
	typeA    = queryType(dns.TypeA)
	typeAAAA = queryType(dns.TypeAAAA)
	typeTXT  = queryType(dns.TypeTXT)
)

func (qt queryType) String() string {
	if name := dns.TypeToString[uint16(qt)]; name != "" {
		return name
	}
	return strconv.FormatUint(uint64(qt), 10)
}

// Unpack unpacks a string to a queryType.
func (qt *queryType) Unpack(v string) error {
	switch strings.ToLower(v) {
	case "a":
		*qt = typeA
	case "aaaa":
		*qt = typeAAAA
	case "reverse", "ptr":
		*qt = typePTR
	case "txt":
		*qt = typeTXT
	default:
		return fmt.Errorf("invalid dns lookup type '%s' specified in "+
			"config (valid values are: A, AAAA, PTR, reverse, TXT)", v)
	}
	return nil
}

// cacheConfig defines the success and failure caching parameters.
type cacheConfig struct {
	SuccessCache cacheSettings `config:"success_cache"`
	FailureCache cacheSettings `config:"failure_cache"`
}

// cacheSettings define the caching behavior for an individual cache.
type cacheSettings struct {
	// TTL value for items in cache. Not used for success because we use TTL
	// from the DNS record.
	TTL time.Duration `config:"ttl"`

	// Minimum TTL value for successful DNS responses.
	MinTTL time.Duration `config:"min_ttl" validate:"min=1ns"`

	// Initial capacity. How much space is allocated at initialization.
	InitialCapacity int `config:"capacity.initial" validate:"min=0"`

	// Max capacity of the cache. When capacity is reached a random item is
	// evicted from the cache.
	MaxCapacity int `config:"capacity.max" validate:"min=1"`
}

// Validate validates the data contained in the config.
func (c *config) Validate() error {
	// Flatten the mapping of source fields to target fields.
	c.reverseFlat = map[string]string{}
	for k, v := range c.Fields.Flatten() {
		target, ok := v.(string)
		if !ok {
			return fmt.Errorf("target field for dns lookup of %v "+
				"must be a string but got %T", k, v)
		}
		c.reverseFlat[k] = target
	}

	c.Transport = strings.ToLower(c.Transport)
	switch c.Transport {
	case "tls":
	case "udp":
	default:
		return fmt.Errorf("invalid transport method type '%v' specified in "+
			"config (valid value is: tls or udp)", c.Transport)
	}
	return nil
}

// Validate validates the data contained in the cacheConfig.
func (c *cacheConfig) Validate() error {
	if c.SuccessCache.MinTTL <= 0 {
		return fmt.Errorf("success_cache.min_ttl must be > 0")
	}
	if c.FailureCache.TTL <= 0 {
		return fmt.Errorf("failure_cache.ttl must be > 0")
	}

	if c.SuccessCache.MaxCapacity <= 0 {
		return fmt.Errorf("success_cache.capacity.max must be > 0")
	}
	if c.FailureCache.MaxCapacity <= 0 {
		return fmt.Errorf("failure_cache.capacity.max must be > 0")
	}

	if c.SuccessCache.MaxCapacity < c.SuccessCache.InitialCapacity {
		return fmt.Errorf("success_cache.capacity.max must be >= success_cache.capacity.initial")
	}
	if c.FailureCache.MaxCapacity < c.FailureCache.InitialCapacity {
		return fmt.Errorf("failure_cache.capacity.max must be >= failure_cache.capacity.initial")
	}

	return nil
}

func defaultConfig() config {
	return config{
		cacheConfig: cacheConfig{
			SuccessCache: cacheSettings{
				MinTTL:          time.Minute,
				InitialCapacity: 1000,
				MaxCapacity:     10000,
			},
			FailureCache: cacheSettings{
				MinTTL:          time.Minute,
				TTL:             time.Minute,
				InitialCapacity: 1000,
				MaxCapacity:     10000,
			},
		},
		Transport: "udp",
		Timeout:   500 * time.Millisecond,
	}
}
