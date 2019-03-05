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

	"github.com/tsg/gopacket/pcap"

	"github.com/elastic/beats/libbeat/logp"
)

var deviceAnySupported = runtime.GOOS == "linux"

// ListDevicesNames returns the list of adapters available for sniffing on
// this computer. If the withDescription parameter is set to true, a human
// readable version of the adapter name is added. If the withIP parameter
// is set to true, IP address of the adapter is added.
func ListDeviceNames(withDescription bool, withIP bool) ([]string, error) {
	devices, err := pcap.FindAllDevs()
	if err != nil {
		return []string{}, err
	}

	ret := []string{}
	for _, dev := range devices {
		r := dev.Name

		if withDescription {
			desc := "No description available"
			if len(dev.Description) > 0 {
				desc = dev.Description
			}
			r += fmt.Sprintf(" (%s)", desc)
		}

		if withIP {
			ips := "Not assigned ip address"
			if len(dev.Addresses) > 0 {
				ips = ""

				for i, address := range []pcap.InterfaceAddress(dev.Addresses) {
					// Add a space between the IP address.
					if i > 0 {
						ips += " "
					}

					ips += fmt.Sprintf("%s", address.IP.String())
				}
			}
			r += fmt.Sprintf(" (%s)", ips)

		}
		ret = append(ret, r)
	}
	return ret, nil
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
