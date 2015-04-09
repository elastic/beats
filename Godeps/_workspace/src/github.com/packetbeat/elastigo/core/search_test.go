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
	"github.com/bmizerany/assert"
	"testing"
)

func TestSearchRequest(t *testing.T) {
	qry := map[string]interface{}{
		"query": map[string]interface{}{
			"wildcard": map[string]string{"actor": "a*"},
		},
	}
	var args map[string]interface{}
	out, err := SearchRequest("github", "", args, qry)
	//log.Println(out)
	assert.T(t, &out != nil && err == nil, fmt.Sprintf("Should get docs"))
	assert.T(t, out.Hits.Len() == 10, fmt.Sprintf("Should have 10 docs but was %v", out.Hits.Len()))
	expectedHits := 621
	assert.T(t, CloseInt(out.Hits.Total, expectedHits), fmt.Sprintf("Should have %v hits but was %v", expectedHits, out.Hits.Total))
}

func TestSearchResultToJSON(t *testing.T) {
	qry := map[string]interface{}{
		"query": map[string]interface{}{
			"wildcard": map[string]string{"actor": "a*"},
		},
	}
	var args map[string]interface{}
	out, err := SearchRequest("github", "", args, qry)

	if err != nil {
		t.Error(err)
	}
	_, err = json.Marshal(out.Hits.Hits)
	if err != nil {
		t.Error(err)
	}
}
