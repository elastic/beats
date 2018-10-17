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

package elb

import (
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/service/elbv2"
)

func TestBasicELB(t *testing.T) {
	stop := watch(
		10*time.Second,
		func(arn string, lb *elbv2.LoadBalancer) {
			fmt.Printf("GOT A NEW LB %s | %v\n", arn, lb)
		},
		func(arn string) {
			fmt.Printf("STOPPED A LB %s\n", arn)
		},
	)
	for {
		time.Sleep(time.Second)
	}
	stop()
}
