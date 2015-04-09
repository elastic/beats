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
	"net/http"
)

// Get allows caller to get a typed JSON document from the index based on its id.
// GET - retrieves the doc
// HEAD - checks for existence of the doc
// http://www.elasticsearch.org/guide/reference/api/get.html
// TODO: make this implement an interface
func get(index string, _type string, id string, args map[string]interface{}, source interface{}) (api.BaseResponse, error) {
	var url string
	retval := api.BaseResponse{Source: source}
	if len(_type) > 0 {
		url = fmt.Sprintf("/%s/%s/%s", index, _type, id)
	} else {
		url = fmt.Sprintf("/%s/%s", index, id)
	}
	body, err := api.DoCommand("GET", url, args, nil)
	if err != nil {
		return retval, err
	}
	if err == nil {
		// marshall into json
		jsonErr := json.Unmarshal(body, &retval)
		if jsonErr != nil {
			return retval, jsonErr
		}
	}
	return retval, err
}

// The get API allows to get a typed JSON document from the index based on its id.
// GET - retrieves the doc
// HEAD - checks for existence of the doc
// http://www.elasticsearch.org/guide/reference/api/get.html
// TODO: make this implement an interface
func Get(index string, _type string, id string, args map[string]interface{}) (api.BaseResponse, error) {
	return get(index, _type, id, args, nil)
}

// Same as Get but with custom source type.
func GetCustom(index string, _type string, id string, args map[string]interface{}, source interface{}) (api.BaseResponse, error) {
	return get(index, _type, id, args, source)
}

// GetSource retrieves the document by id and converts it to provided interface
func GetSource(index string, _type string, id string, args map[string]interface{}, source interface{}) error {
	url := fmt.Sprintf("/%s/%s/%s/_source", index, _type, id)
	body, err := api.DoCommand("GET", url, args, nil)
	if err == nil {
		err = json.Unmarshal(body, &source)
	}
	return err
}

// Exists allows caller to check for the existance of a document using HEAD
func Exists(index string, _type string, id string, args map[string]interface{}) (bool, error) {

	var url string

	query, err := api.QueryString(args)
	if err != nil {
		return false, err
	}

	if len(_type) > 0 {
		url = fmt.Sprintf("/%s/%s/%s?fields=_id", index, _type, id)
	} else {
		url = fmt.Sprintf("/%s/%s?fields=_id", index, id)
	}

	req, err := api.ElasticSearchRequest("HEAD", url, query)

	if err != nil {
		return false, err
	}

	httpStatusCode, _, err := req.Do(nil)

	if err != nil {
		return false, err
	}
	if httpStatusCode == http.StatusOK {
		return true, err
	}
	return false, err
}

// ExistsIndex allows caller to check for the existance of an index or a type using HEAD
func ExistsIndex(index string, _type string, args map[string]interface{}) (bool, error) {
	var url string

	query, err := api.QueryString(args)
	if err != nil {
		return false, err
	}

	if len(_type) > 0 {
		url = fmt.Sprintf("/%s/%s", index, _type)
	} else {
		url = fmt.Sprintf("/%s", index)
	}
	req, err := api.ElasticSearchRequest("HEAD", url, query)
	httpStatusCode, _, err := req.Do(nil)

	if err != nil {
		return false, err
	}
	if httpStatusCode == http.StatusOK {
		return true, err
	}
	return false, err
}
