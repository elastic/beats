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

package ilm

//func TestNewIlmPolicyCfg(t *testing.T) {
//	beatInfo := beat.Info{
//		Beat:        "testbeatilm",
//		IndexPrefix: "testbeat",
//		Version:     "1.2.3",
//	}
//	pName := "deleteAfter30days"
//
//	for _, data := range []struct{
//		idx string
//		cfg *ilmPolicyCfg
//	}{
//		{"", nil},
//		{"testbeat", &ilmPolicyCfg{idxName: "testbeat", policyName: pName}},
//		{"testbeat-%{[beat.version]}", &ilmPolicyCfg{idxName: "testbeat-1.2.3", policyName: pName}},
//		{"testbeat-SNAPSHOT-%{[beat.version]}", &ilmPolicyCfg{idxName: "testbeat-snapshot-1.2.3", policyName: pName}},
//		{"testbeat-%{[beat.version]}-%{+yyyy.MM.dd}", nil},
//	}{
//		cfg := newIlmPolicyCfg(data.idx, pName, beatInfo)
//		assert.Equal(t, data.cfg, cfg)
//	}
//}
