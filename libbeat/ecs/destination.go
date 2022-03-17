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

package ecs

// Destination fields capture details about the receiver of a network
// exchange/packet. These fields are populated from a network event, packet, or
// other event containing details of a network transaction.
// Destination fields are usually populated in conjunction with source fields.
// The source and destination fields are considered the baseline and should
// always be filled if an event contains source and destination details from a
// network transaction. If the event also contains identification of the client
// and server roles, then the client and server fields should also be
// populated.
type Destination struct {
	// Some event destination addresses are defined ambiguously. The event will
	// sometimes list an IP, a domain or a unix socket.  You should always
	// store the raw address in the `.address` field.
	// Then it should be duplicated to `.ip` or `.domain`, depending on which
	// one it is.
	Address string `ecs:"address"`

	// IP address of the destination (IPv4 or IPv6).
	IP string `ecs:"ip"`

	// Port of the destination.
	Port int64 `ecs:"port"`

	// MAC address of the destination.
	// The notation format from RFC 7042 is suggested: Each octet (that is,
	// 8-bit byte) is represented by two [uppercase] hexadecimal digits giving
	// the value of the octet as an unsigned integer. Successive octets are
	// separated by a hyphen.
	MAC string `ecs:"mac"`

	// Destination domain.
	Domain string `ecs:"domain"`

	// The highest registered destination domain, stripped of the subdomain.
	// For example, the registered domain for "foo.example.com" is
	// "example.com".
	// This value can be determined precisely with a list like the public
	// suffix list (http://publicsuffix.org). Trying to approximate this by
	// simply taking the last two labels will not work well for TLDs such as
	// "co.uk".
	RegisteredDomain string `ecs:"registered_domain"`

	// The effective top level domain (eTLD), also known as the domain suffix,
	// is the last part of the domain name. For example, the top level domain
	// for example.com is "com".
	// This value can be determined precisely with a list like the public
	// suffix list (http://publicsuffix.org). Trying to approximate this by
	// simply taking the last label will not work well for effective TLDs such
	// as "co.uk".
	TopLevelDomain string `ecs:"top_level_domain"`

	// The subdomain portion of a fully qualified domain name includes all of
	// the names except the host name under the registered_domain.  In a
	// partially qualified domain, or if the the qualification level of the
	// full name cannot be determined, subdomain contains all of the names
	// below the registered domain.
	// For example the subdomain portion of "www.east.mydomain.co.uk" is
	// "east". If the domain has multiple levels of subdomain, such as
	// "sub2.sub1.example.com", the subdomain field should contain "sub2.sub1",
	// with no trailing period.
	Subdomain string `ecs:"subdomain"`

	// Bytes sent from the destination to the source.
	Bytes int64 `ecs:"bytes"`

	// Packets sent from the destination to the source.
	Packets int64 `ecs:"packets"`

	// Translated ip of destination based NAT sessions (e.g. internet to
	// private DMZ)
	// Typically used with load balancers, firewalls, or routers.
	NatIP string `ecs:"nat.ip"`

	// Port the source session is translated to by NAT Device.
	// Typically used with load balancers, firewalls, or routers.
	NatPort int64 `ecs:"nat.port"`
}
