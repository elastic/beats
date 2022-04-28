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

package dhcpv4

import (
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"net"
	"strings"

	"github.com/insomniacslk/dhcp/dhcpv4"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func optionsToMap(options []dhcpv4.Option) (mapstr.M, error) {
	opts := mapstr.M{}

	for _, opt := range options {
		if opt.Code() == dhcpv4.OptionEnd {
			break
		}

		switch v := opt.(type) {
		case *dhcpv4.OptMessageType:
			mt, found := dhcpv4.MessageTypeToString[v.MessageType]
			if !found {
				mt = fmt.Sprintf("unknown (%v)", v.MessageType)
			}
			opts.Put("message_type", strings.ToLower(mt))

		case *dhcpv4.OptParameterRequestList:
			var optNames []string
			for _, ro := range v.RequestedOpts {
				if name, ok := dhcpv4.OptionCodeToString[ro]; ok {
					optNames = append(optNames, name)
				} else {
					optNames = append(optNames, fmt.Sprintf("Unknown (%v)", ro))
				}
			}
			opts.Put("parameter_request_list", optNames)

		case *dhcpv4.OptRequestedIPAddress:
			opts.Put("requested_ip_address", v.RequestedAddr.String())

		case *dhcpv4.OptServerIdentifier:
			opts.Put("server_identifier", v.ServerID.String())

		case *dhcpv4.OptBroadcastAddress:
			opts.Put("broadcast_address", v.BroadcastAddress.String())

		case *dhcpv4.OptMaximumDHCPMessageSize:
			opts.Put("max_dhcp_message_size", v.Size)

		case *dhcpv4.OptClassIdentifier:
			opts.Put("class_identifier", v.Identifier)

		case *dhcpv4.OptDomainName:
			opts.Put("domain_name", v.DomainName)

		case *dhcpv4.OptDomainNameServer:
			var dnsServers []string
			for _, s := range v.NameServers {
				dnsServers = append(dnsServers, s.String())
			}
			opts.Put("dns_servers", dnsServers)

		case *dhcpv4.OptVIVC:
			var subOptions []mapstr.M
			for _, vendorOpt := range v.Identifiers {
				subOptions = append(subOptions, mapstr.M{
					"id":   vendorOpt.EntID,
					"data": hex.EncodeToString(vendorOpt.Data),
				})
			}
			opts.Put("vendor_identifying_options", subOptions)

		case *dhcpv4.OptionGeneric:
			// Generic options have just a []byte so we need to do extra parsing.
			switch opt.Code() {
			case dhcpv4.OptionSubnetMask:
				if len(v.Data) >= 4 {
					opts.Put("subnet_mask", net.IP(v.Data).String())
				}

			case dhcpv4.OptionTimeOffset:
				if len(v.Data) >= 4 {
					opts.Put("utc_time_offset_sec", int32(binary.BigEndian.Uint32(v.Data)))
				}

			case dhcpv4.OptionRouter:
				ipOpt, err := ParseIPAddressOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}
				opts.Put("router", ipOpt.IPAddress.String())

			case dhcpv4.OptionTimeServer:
				tsOpt, err := ParseIPAddressesOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}

				var timeServers []string
				for _, s := range tsOpt.IPAddresses {
					timeServers = append(timeServers, s.String())
				}
				opts.Put("time_servers", timeServers)

			case dhcpv4.OptionNTPServers:
				tsOpt, err := ParseIPAddressesOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}

				var timeServers []string
				for _, s := range tsOpt.IPAddresses {
					timeServers = append(timeServers, s.String())
				}
				opts.Put("ntp_servers", timeServers)

			case dhcpv4.OptionHostName:
				txt, err := ParseTextOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}
				opts.Put("hostname", txt.Text)

			case dhcpv4.OptionIPAddressLeaseTime:
				if len(v.Data) >= 4 {
					opts.Put("ip_address_lease_time_sec", binary.BigEndian.Uint32(v.Data))
				}

			case dhcpv4.OptionMessage:
				txt, err := ParseTextOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}
				opts.Put("message", txt.Text)

			case dhcpv4.OptionRenewTimeValue:
				if len(v.Data) >= 4 {
					opts.Put("renewal_time_sec", binary.BigEndian.Uint32(v.Data))
				}

			case dhcpv4.OptionRebindingTimeValue:
				if len(v.Data) >= 4 {
					opts.Put("rebinding_time_sec", binary.BigEndian.Uint32(v.Data))
				}

			case dhcpv4.OptionBootfileName:
				txt, err := ParseTextOption(opt.ToBytes())
				if err != nil {
					return nil, err
				}
				opts.Put("boot_file_name", txt.Text)

			}
		}
	}

	if len(opts) > 0 {
		return opts, nil
	}
	return nil, nil
}
