// Copyright 2017 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package auparse

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"

	"github.com/pkg/errors"
)

func parseSockaddr(s string) (map[string]string, error) {
	addressFamily, err := hexToDec(s[2:4] + s[0:2]) // host-order
	if err != nil {
		return nil, err
	}

	out := map[string]string{}
	switch addressFamily {
	case 1: // AF_UNIX
		socket, err := hexToASCII(s[4:])
		if err != nil {
			return nil, err
		}

		out["family"] = "unix"
		out["path"] = socket
	case 2: // AF_INET
		port, err := hexToDec(s[4:8])
		if err != nil {
			return nil, err
		}

		ip, err := hexToIP(s[8:16])
		if err != nil {
			return nil, err
		}

		out["family"] = "ipv4"
		out["addr"] = ip
		out["port"] = strconv.Itoa(int(port))
	case 10: // AF_INET6
		port, err := hexToDec(s[4:8])
		if err != nil {
			return nil, err
		}

		flow, err := hexToDec(s[8:16])
		if err != nil {
			return nil, err
		}

		ip, err := hexToIP(s[16:48])
		if err != nil {
			return nil, err
		}

		out["family"] = "ipv6"
		out["addr"] = ip
		out["port"] = strconv.Itoa(int(port))
		if flow > 0 {
			out["flow"] = strconv.Itoa(int(flow))
		}
	case 16: // AF_NETLINK
		out["family"] = "netlink"
		out["saddr"] = s
	default:
		out["family"] = strconv.Itoa(int(addressFamily))
		out["saddr"] = s
	}

	return out, nil
}

func hexToASCII(h string) (string, error) {
	output, err := hex.DecodeString(h)

	nullTerm := bytes.Index(output, []byte{0})
	if nullTerm != -1 {
		output = output[:nullTerm]
	}

	return string(output), err
}

func hexToDec(h string) (int32, error) {
	num, err := strconv.ParseInt(h, 16, 32)
	return int32(num), err
}

func hexToIP(h string) (string, error) {
	if len(h) == 8 {
		a1, _ := hexToDec(h[0:2])
		a2, _ := hexToDec(h[2:4])
		a3, _ := hexToDec(h[4:6])
		a4, _ := hexToDec(h[6:8])
		return fmt.Sprintf("%d.%d.%d.%d", a1, a2, a3, a4), nil
	} else if len(h) == 32 {
		b, err := hex.DecodeString(h)
		if err != nil {
			return "", err
		}
		return net.IP(b).String(), nil
	}

	return "", errors.New("invalid size")
}
