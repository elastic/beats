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

package withtestlog

import "testing"

func TestLogIsPrintedOnError(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Error("Log should fail")
	t.Errorf("Log should fail with printf style log: %v", 23)
}

func TestLogIsPrintedOnFatal(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Fatal("Log should fail")
}

func TestLogIsPrintedOnFatalf(t *testing.T) {
	t.Log("Log message should be printed")
	t.Logf("printf style log message: %v", 42)
	t.Fatalf("Log should fail with printf style log: %v", 42)
}

func TestLogsWithNewlines(t *testing.T) {
	t.Log("Log\nmessage\nshould\nbe\nprinted")
	t.Logf("printf\nstyle\nlog\nmessage:\n%v", 42)
	t.Fatalf("Log\nshould\nfail\nwith\nprintf\nstyle\nlog:\n%v", 42)
}
