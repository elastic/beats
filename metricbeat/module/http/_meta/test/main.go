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

package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/jsonarr", serveJSONArr)
	http.HandleFunc("/jsonobj", serveJSONObj)
	http.HandleFunc("/", serveJSONObj)

	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

func serveJSONArr(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `[{"hello1":"world1"}, {"hello2": "world2"}]`)
}

func serveJSONObj(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, `{"hello":"world"}`)
}
