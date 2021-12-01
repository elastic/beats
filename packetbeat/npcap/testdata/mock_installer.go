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
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetPrefix("installer message: ")
	switch len(os.Args) {
	case 3, 4:
		// OK
	default:
		log.Fatalf("unexpected number of argument: want 3 or 4 but got:%q", os.Args)
	}
	if os.Args[1] != "/S" {
		log.Fatalf(`unexpected first argument: want:"/S" got:%q`, os.Args[1])
	}
	if os.Args[2] != "/winpcap_mode=yes" && os.Args[2] != "/winpcap_mode=no" {
		log.Fatalf(`unexpected second argument: want:"/winpcap_mode={yes,no}" got:%q`, os.Args[2])
	}
	if len(os.Args) > 3 && !strings.HasPrefix(os.Args[len(os.Args)-1], "/D=") {
		log.Fatalf(`unexpected final argument: want:"/D=<path>" got:%#q`, os.Args[3])
	}
}
