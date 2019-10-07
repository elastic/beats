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

package elasticsearch

import (
	"fmt"
	"testing"
)

func TestConnectCallbacksManagement(t *testing.T) {
	f0 := func(client *Client) error { fmt.Println("i am function #0"); return nil }
	f1 := func(client *Client) error { fmt.Println("i am function #1"); return nil }
	f2 := func(client *Client) error { fmt.Println("i am function #2"); return nil }

	_, err := RegisterConnectCallback(f0)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id1, err := RegisterConnectCallback(f1)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id2, err := RegisterConnectCallback(f2)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}

	t.Logf("removing second callback")
	DeregisterConnectCallback(id1)
	if _, ok := connectCallbackRegistry.callbacks[id2]; !ok {
		t.Fatalf("third callback cannot be retrieved")
	}
}

func TestGlobalConnectCallbacksManagement(t *testing.T) {
	f0 := func(client *Client) error { fmt.Println("i am function #0"); return nil }
	f1 := func(client *Client) error { fmt.Println("i am function #1"); return nil }
	f2 := func(client *Client) error { fmt.Println("i am function #2"); return nil }

	_, err := RegisterGlobalCallback(f0)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id1, err := RegisterGlobalCallback(f1)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}
	id2, err := RegisterGlobalCallback(f2)
	if err != nil {
		t.Fatalf("error while registering callback: %v", err)
	}

	t.Logf("removing second callback")
	DeregisterGlobalCallback(id1)
	if _, ok := globalCallbackRegistry.callbacks[id2]; !ok {
		t.Fatalf("third callback cannot be retrieved")
	}
}
