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

package communityid

type config struct {
	Fields fieldsConfig `config:"fields"`
	Target string       `config:"target"`
	Seed   uint16       `config:"seed"`
}

type fieldsConfig struct {
	SourceIP          string `config:"source_ip"`
	SourcePort        string `config:"source_port"`
	DestinationIP     string `config:"destination_ip"`
	DestinationPort   string `config:"destination_port"`
	TransportProtocol string `config:"transport"`
	ICMPType          string `config:"icmp_type"`
	ICMPCode          string `config:"icmp_code"`
}

func defaultConfig() config {
	return config{
		Fields: fieldsConfig{
			SourceIP:          "source.ip",
			SourcePort:        "source.port",
			DestinationIP:     "destination.ip",
			DestinationPort:   "destination.port",
			TransportProtocol: "network.transport",
			ICMPType:          "icmp.type",
			ICMPCode:          "icmp.code",
		},
		Target: "network.community_id",
	}
}
