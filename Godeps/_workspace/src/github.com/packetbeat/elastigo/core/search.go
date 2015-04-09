// Copyright 2013 Matthew Baird
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package core

import (
	"encoding/json"
	"fmt"
	"github.com/packetbeat/elastigo/api"
	"strconv"
	"strings"
)

var (
	DebugRequests = false
)

// SearchRequest performs a very basic search on an index via the request URI API.
//
// params:
//   @index:  the elasticsearch index
//   @_type:  optional ("" if not used) search specific type in this index
//   @args:   a map of URL parameters. Allows all the URI-request parameters allowed by ElasticSearch.
//   @query:  this can be one of 3 types:
//              1)  string value that is valid elasticsearch
//              2)  io.Reader that can be set in body (also valid elasticsearch string syntax..)
//              3)  other type marshalable to json (also valid elasticsearch json)
//
//   out, err := SearchRequest(true, "github", map[string]interface{} {"from" : 10}, qryType)
//
// http://www.elasticsearch.org/guide/reference/api/search/uri-request.html
func SearchRequest(index string, _type string, args map[string]interface{}, query interface{}) (SearchResult, error) {
	var uriVal string
	var retval SearchResult
	if len(_type) > 0 && _type != "*" {
		uriVal = fmt.Sprintf("/%s/%s/_search", index, _type)
	} else {
		uriVal = fmt.Sprintf("/%s/_search", index)
	}
	body, err := api.DoCommand("POST", uriVal, args, query)
	if err != nil {
		return retval, err
	}
	if err == nil {
		// marshall into json
		jsonErr := json.Unmarshal([]byte(body), &retval)
		if jsonErr != nil {
			return retval, jsonErr
		}
	}
	return retval, err
}

// SearchUri performs the simplest possible query in url string
// params:
//   @index:  the elasticsearch index
//   @_type:  optional ("" if not used) search specific type in this index
//   @args: a map of URL parameters. Most important one is q
//
//   out, err := SearchUri("github","", map[string]interface{} { "q" : `user:kimchy`})
//
// produces a request like this:    host:9200/github/_search?q=user:kimchy"
//
// http://www.elasticsearch.org/guide/reference/api/search/uri-request.html
func SearchUri(index, _type string, args map[string]interface{}) (SearchResult, error) {
	var uriVal string
	var retval SearchResult
	if len(_type) > 0 && _type != "*" {
		uriVal = fmt.Sprintf("/%s/%s/_search", index, _type)
	} else {
		uriVal = fmt.Sprintf("/%s/_search", index)
	}
	//log.Println(uriVal)
	body, err := api.DoCommand("GET", uriVal, args, nil)
	if err != nil {
		return retval, err
	}
	if err == nil {
		// marshall into json
		jsonErr := json.Unmarshal([]byte(body), &retval)
		if jsonErr != nil {
			return retval, jsonErr
		}
	}
	return retval, err
}

func Scroll(args map[string]interface{}, scroll_id string) (SearchResult, error) {
	var url string
	var retval SearchResult

	if _, ok := args["scroll"]; !ok {
		return retval, fmt.Errorf("Cannot call scroll without 'scroll' in arguments")
	}

	url = "/_search/scroll"

	body, err := api.DoCommand("POST", url, args, scroll_id)
	if err != nil {
		return retval, err
	}
	if err == nil {
		// marshall into json
		jsonErr := json.Unmarshal([]byte(body), &retval)
		if jsonErr != nil {
			return retval, jsonErr
		}
	}
	return retval, err
}

type SearchResult struct {
	Took         int             `json:"took"`
	TimedOut     bool            `json:"timed_out"`
	ShardStatus  api.Status      `json:"_shards"`
	Hits         Hits            `json:"hits"`
	Facets       json.RawMessage `json:"facets,omitempty"` // structure varies on query
	ScrollId     string          `json:"_scroll_id,omitempty"`
	Aggregations json.RawMessage `json:"aggregations,omitempty"` // structure varies on query
}

func (s *SearchResult) String() string {
	return fmt.Sprintf("<Results took=%v Timeout=%v hitct=%v />", s.Took, s.TimedOut, s.Hits.Total)
}

type Hits struct {
	Total int `json:"total"`
	//	MaxScore float32 `json:"max_score"`
	Hits []Hit `json:"hits"`
}

func (h *Hits) Len() int {
	return len(h.Hits)
}

type Hit struct {
	Index       string           `json:"_index"`
	Type        string           `json:"_type,omitempty"`
	Id          string           `json:"_id"`
	Score       Float32Nullable  `json:"_score,omitempty"` // Filters (no query) dont have score, so is null
	Source      *json.RawMessage `json:"_source"`          // marshalling left to consumer
	Fields      *json.RawMessage `json:"fields"`           // when a field arg is passed to ES, instead of _source it returns fields
	Explanation *Explanation     `json:"_explanation,omitempty"`
}

type Explanation struct {
	Value       float32        `json:"value"`
	Description string         `json:"description"`
	Details     []*Explanation `json:"details,omitempty"`
}

func (e *Explanation) String(indent string) string {
	if len(e.Details) == 0 {
		return fmt.Sprintf("%s>>>  %v = %s", indent, e.Value, strings.Replace(e.Description, "\n", "", -1))
	} else {
		detailStrs := make([]string, 0)
		for _, detail := range e.Details {
			detailStrs = append(detailStrs, fmt.Sprintf("%s", detail.String(indent+"| ")))
		}
		return fmt.Sprintf("%s%v = %s(\n%s\n%s)", indent, e.Value, strings.Replace(e.Description, "\n", "", -1), strings.Join(detailStrs, "\n"), indent)
	}
}

// Elasticsearch returns some invalid (according to go) json, with floats having...
//
// json: cannot unmarshal null into Go value of type float32 (see last field.)
//
// "hits":{"total":6808,"max_score":null,
//    "hits":[{"_index":"10user","_type":"user","_id":"751820","_score":null,
type Float32Nullable float32

func (i *Float32Nullable) UnmarshalJSON(data []byte) error {
	if len(data) == 0 || string(data) == "null" {
		return nil
	}

	if in, err := strconv.ParseFloat(string(data), 32); err != nil {
		return err
	} else {
		*i = Float32Nullable(in)
	}
	return nil
}
