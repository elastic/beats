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

package sniffer

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"

	"github.com/google/gopacket/pcap"

	"github.com/elastic/beats/v7/packetbeat/route"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/monitoring"
)

var deviceAnySupported = runtime.GOOS == "linux"

// ListDeviceNames returns the list of adapters available for sniffing on this
// computer. If the withDescription parameter is set to true, a human-readable
// version of the adapter name is added. If the withIP parameter is set to
// true, IP address of the adapter is added.
func ListDeviceNames(withDescription, withIP bool) ([]string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return nil, err
	}
	return formatDeviceNames(devices, withDescription, withIP), nil
}

func formatDeviceNames(devices []pcap.Interface, withDescription, withIP bool) []string {
	if len(devices) == 0 {
		return nil
	}
	names := make([]string, 0, len(devices))
	var buf strings.Builder
	for _, dev := range devices {
		buf.Reset()
		buf.WriteString(dev.Name)

		if withDescription {
			desc := "No description available"
			if len(dev.Description) > 0 {
				desc = dev.Description
			}
			fmt.Fprintf(&buf, " (%s)", desc)
		}

		if withIP {
			buf.WriteString(" (")
			if len(dev.Addresses) == 0 {
				buf.WriteString("Not assigned ip address")
			} else {
				for i, address := range dev.Addresses {
					if i != 0 {
						buf.WriteByte(' ')
					}
					fmt.Fprint(&buf, address.IP)
				}
			}
			buf.WriteByte(')')
		}
		names = append(names, buf.String())
	}
	return names
}

func resolveDeviceName(name string) (string, error) {
	if name == "" {
		if runtime.GOOS == "linux" {
			return "any", nil
		}
		name = "default_route"
	}
	if strings.HasPrefix(name, "default_route") {
		var (
			iface string
			err   error
		)
		registerDefaultRouteMetricOnce()
		switch name {
		case "default_route":
			for _, inet := range []int{syscall.AF_INET, syscall.AF_INET6} {
				iface, _, err = route.Default(inet)
				if err == nil {
					break
				}
				if err != route.ErrNotFound { //nolint:errorlint // route.ErrNotFound is never wrapped.
					return "", err
				}
			}
		case "default_route_ipv4":
			iface, _, err = route.Default(syscall.AF_INET)
		case "default_route_ipv6":
			iface, _, err = route.Default(syscall.AF_INET6)
		default:
			return "", fmt.Errorf("invalid default route: %v", name)
		}
		if err != nil {
			return "", fmt.Errorf("failed to get default route device: %w", err)
		}
		defaultRouteMetric.Set(iface)

		devices, err := ListDeviceNames(false, false)
		if err != nil {
			return "", fmt.Errorf("failed to get device list: %w", err)
		}
		// The order of devices returned by pcap differs from the order
		// obtained by route, so search by iface name.
		for _, dev := range devices {
			if sameDevice(iface, dev) {
				return dev, nil
			}
		}
	}

	index, err := strconv.Atoi(name)
	if err != nil {
		return name, nil //nolint:nilerr // This is a non-numeric interface identifier.
	}

	// The device is an index into the interface list.
	devices, err := ListDeviceNames(false, false)
	if err != nil {
		return "", fmt.Errorf("failed to get device list: %w", err)
	}

	name, err = deviceNameFromIndex(index, devices)
	if err != nil {
		return "", fmt.Errorf("invalid device index %d: %w", index, err)
	}

	logp.L().Named("sniffer").Info("Resolved device index %d to device: %s", index, name)
	return name, nil
}

var (
	registerRoute      sync.Once
	defaultRouteMetric *monitoring.String
)

func registerDefaultRouteMetricOnce() {
	registerRoute.Do(func() {
		defaultRouteMetric = monitoring.NewString(nil, "packetbeat.default_route")
	})
}

func sameDevice(route, pcap string) bool {
	if runtime.GOOS == "windows" {
		// The device returned by route does not have the same device tree
		// as the device obtained from npcap, so rely on the GUID to match.
		idx := strings.Index(pcap, "_") // Replace this with strings.Cut.
		if idx > -1 {
			pcap = pcap[idx+1:]
		}
	}
	return route == pcap
}

func deviceNameFromIndex(index int, devices []string) (string, error) {
	if index >= len(devices) {
		return "", fmt.Errorf("looking for device index %d, but there are only %d devices",
			index, len(devices))
	}
	return devices[index], nil
}
