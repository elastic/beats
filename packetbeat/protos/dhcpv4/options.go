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

func optionsToMap(dhcp *dhcpv4.DHCPv4) (mapstr.M, error) {
	opts := mapstr.M{}

	if msgType := dhcp.MessageType(); msgType != dhcpv4.MessageTypeNone {
		opts.Put("message_type", strings.ToLower(msgType.String()))
	}

	if reqList := dhcp.ParameterRequestList(); reqList != nil {
		var optNames []string
		for _, optCode := range reqList {
			optNames = append(optNames, optCode.String())
		}
		opts.Put("parameter_request_list", optNames)
	}

	if reqIP := dhcp.RequestedIPAddress(); reqIP != nil {
		opts.Put("requested_ip_address", reqIP.String())
	}

	if srvIP := dhcp.ServerIdentifier(); srvIP != nil {
		opts.Put("server_identifier", srvIP.String())
	}

	if broadcastIP := dhcp.BroadcastAddress(); broadcastIP != nil {
		opts.Put("broadcast_address", broadcastIP.String())
	}

	if maxMsgSize, err := dhcp.MaxMessageSize(); err == nil {
		opts.Put("max_dhcp_message_size", maxMsgSize)
	}

	if classID := dhcp.ClassIdentifier(); classID != "" {
		opts.Put("class_identifier", classID)
	}

	if domainName := dhcp.DomainName(); domainName != "" {
		opts.Put("domain_name", domainName)
	}

	if dnsServers := dhcp.DNS(); dnsServers != nil {
		var dnsServerStr []string
		for _, srv := range dnsServers {
			dnsServerStr = append(dnsServerStr, srv.String())
		}
		opts.Put("dns_servers", dnsServerStr)
	}

	// see RFC3925
	if vivc := dhcp.VIVC(); vivc != nil {
		var subOptions []mapstr.M
		for _, subOpt := range vivc {
			subOptions = append(subOptions, mapstr.M{
				"id":   subOpt.EntID,
				"data": hex.EncodeToString(subOpt.Data),
			})
		}
		opts.Put("vendor_identifying_options", subOptions)
	}

	if mask := dhcp.SubnetMask(); mask != nil {
		opts.Put("subnet_mask", net.IP(mask).String())
	}

	if offset := dhcp.GetOneOption(dhcpv4.OptionTimeOffset); offset != nil {
		opts.Put("utc_time_offset_sec", int32(binary.BigEndian.Uint32(offset))) //nolint:gosec // RFC says it should be signed
	}

	if routerList := dhcp.Router(); routerList != nil {
		var routersStr []string
		for _, router := range routerList {
			routersStr = append(routersStr, router.String())
		}
		// NOTE: this is a breaking change, but RFC2132 says router should be a *list*
		opts.Put("router", routersStr)
	}

	if timeServer := dhcp.GetOneOption(dhcpv4.OptionTimeServer); timeServer != nil {
		var ips dhcpv4.IPs
		if err := ips.FromBytes(timeServer); err != nil {
			return nil, fmt.Errorf("error parsing IP options for time servers: %w", err)
		}
		var timeServers []string
		for _, s := range ips {
			timeServers = append(timeServers, s.String())
		}
		opts.Put("time_servers", timeServers)
	}

	if ntpServers := dhcp.NTPServers(); ntpServers != nil {
		var timeServers []string
		for _, srv := range ntpServers {
			timeServers = append(timeServers, srv.String())
		}
		opts.Put("ntp_servers", timeServers)
	}

	if hostname := dhcp.HostName(); hostname != "" {
		opts.Put("hostname", hostname)
	}

	if leaseTime := dhcp.IPAddressLeaseTime(0); leaseTime != 0 {
		opts.Put("ip_address_lease_time_sec", uint32(leaseTime.Seconds()))
	}

	if msg := dhcp.Message(); msg != "" {
		opts.Put("message", msg)
	}

	if time := dhcp.IPAddressRenewalTime(0); time != 0 {
		opts.Put("renewal_time_sec", uint32(time.Seconds()))
	}

	if time := dhcp.IPAddressRebindingTime(0); time != 0 {
		opts.Put("rebinding_time_sec", uint32(time.Seconds()))
	}

	if bootFile := dhcp.BootFileNameOption(); bootFile != "" {
		opts.Put("boot_file_name", bootFile)
	}

	if len(opts) > 0 {
		return opts, nil
	}
	return nil, nil
}
