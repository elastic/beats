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

package secretstores

import (
	"errors"
	"fmt"
	"net/netip"
	"strings"

	"golang.org/x/net/idna"

	conf "github.com/elastic/elastic-agent-libs/config"

	"github.com/elastic/beats/v7/heartbeat/monitors/plugin"
)

// allowedHostsKey is the secret_stores entry setting restricting the hosts
// the secrets of a connection can be sent to.
const allowedHostsKey = "allowed_hosts"

// ErrNotAllowed is wrapped by the errors of monitors that can't use a
// connection because of its allowed_hosts.
var ErrNotAllowed = errors.New("not allowed by " + allowedHostsKey)

// checkAllowedHosts returns an error unless every destination of the resolved
// monitor config cfg is an allowed host of each entry. Hosts are not named in
// errors: resolved hosts can be secrets.
func checkAllowedHosts(entries []entry, monitorType string, destinations plugin.DestinationsFunc, cfg *conf.C) error {
	if len(entries) == 0 {
		return nil
	}
	if destinations == nil {
		return fmt.Errorf("%s: connection %q: %w: monitor type %q is not supported", ConfigKey, entries[0].name, ErrNotAllowed, monitorType)
	}
	hosts, err := destinations(cfg)
	if err != nil {
		return fmt.Errorf("%s: connection %q: %w: %s", ConfigKey, entries[0].name, ErrNotAllowed, err.Error())
	}
	for _, e := range entries {
		for _, h := range hosts {
			if n, ok := normalizeHost(h); !ok || !e.allowedHosts[n] {
				return fmt.Errorf("%s: connection %q: %w: the monitor connects to a host that is not allowed", ConfigKey, e.name, ErrNotAllowed)
			}
		}
	}
	return nil
}

// parseAllowedHosts returns the normalized hosts of an allowed_hosts value.
func parseAllowedHosts(raw any) (map[string]bool, error) {
	if raw == nil {
		return nil, nil
	}
	items, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("%s: expected a list of host names or IPs, got %T", allowedHostsKey, raw)
	}
	hosts := make(map[string]bool, len(items))
	for _, item := range items {
		h, ok := item.(string)
		if !ok {
			return nil, fmt.Errorf("%s: expected a host name or IP, got %T", allowedHostsKey, item)
		}
		if !validHost(h) {
			return nil, fmt.Errorf("%s: %q must be a host name or IP, without scheme, port or path", allowedHostsKey, h)
		}
		n, ok := normalizeHost(h)
		if !ok {
			return nil, fmt.Errorf("%s: %q is not a valid internationalized host name", allowedHostsKey, h)
		}
		hosts[n] = true
	}
	return hosts, nil
}

// validHost reports whether h is a host name or an IP. Wildcards are not supported.
func validHost(h string) bool {
	if h == "" || strings.TrimSpace(h) != h {
		return false
	}
	if _, err := netip.ParseAddr(h); err == nil {
		return true
	}

	return !strings.ContainsAny(h, ":/@?#*[]\\ \t\r\n")
}

// normalizeHost returns the form of a host used to compare it with the
// allowed hosts: without the trailing dot of a fully qualified name, IPs in
// their canonical form, names in lowercase ASCII. Like net/http, which dials
// the result, non-ASCII names are converted with IDNA: Unicode case folding
// doesn't always agree with it ("ẞ" lowercases to "ß", IDNA maps it to
// "ss"). ok is false if the name can't be converted; such a host must not
// match.
func normalizeHost(h string) (normalized string, ok bool) {
	h = strings.TrimSuffix(h, ".")
	if ip, err := netip.ParseAddr(h); err == nil {
		return ip.Unmap().String(), true
	}
	if !isASCII(h) {
		var err error
		if h, err = idna.Lookup.ToASCII(h); err != nil || h == "" {
			return "", false
		}
	}
	return strings.ToLower(h), true
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] >= 0x80 {
			return false
		}
	}
	return true
}
