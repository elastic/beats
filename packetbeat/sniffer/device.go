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

	"github.com/google/gopacket/pcap"

	"github.com/elastic/beats/v8/libbeat/logp"
)

var deviceAnySupported = runtime.GOOS == "linux"

// ListDevicesNames returns the list of adapters available for sniffing on
// this computer. If the withDescription parameter is set to true, a human
// readable version of the adapter name is added. If the withIP parameter
// is set to true, IP address of the adapter is added.
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
				for i, address := range []pcap.InterfaceAddress(dev.Addresses) {
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
		return "any", nil
	}

	if index, err := strconv.Atoi(name); err == nil { // Device is numeric id
		devices, err := ListDeviceNames(false, false)
		if err != nil {
			return "", fmt.Errorf("Error getting devices list: %v", err)
		}

		name, err = deviceNameFromIndex(index, devices)
		if err != nil {
			return "", fmt.Errorf("Couldn't understand device index %d: %v", index, err)
		}

		logp.Info("Resolved device index %d to device: %s", index, name)
	}

	return name, nil
}

func deviceNameFromIndex(index int, devices []string) (string, error) {
	if index >= len(devices) {
		return "", fmt.Errorf("Looking for device index %d, but there are only %d devices",
			index, len(devices))
	}

	return devices[index], nil
}
