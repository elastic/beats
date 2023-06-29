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

package pageinfo

import (
	"bufio"
	"fmt"
	"io"
	"regexp"
	"strconv"
)

// zones represents raw pagetypeinfo data
type zones struct {
	DMA          map[string]map[int]int64
	DMA32        map[string]map[int]int64
	Normal       map[string]map[int]int64
	OrderSummary map[int]int64 `json:"order_summary"`
}

// buddyInfo emulates /proc/buddyinfo by summing migrate types across orders
type buddyInfo struct {
	DMA    map[int]int64
	DMA32  map[int]int64
	Normal map[int]int64
}

// pageInfo represents all the data we get from /proc/pagetypeinfo
type pageInfo struct {
	BuddyInfo buddyInfo
	Zones     map[int64]zones
}

var pageinfoLine = regexp.MustCompile(`Node\s*(\d), zone\s*([a-zA-z0-9]*), type\s*([a-zA-z0-9]*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)\s*(\d*)`)

// readPageFile reads a PageTypeInfo file and returns the parsed data
// This returns a massive representation of all the meaningful data in /proc/pagetypeinfo
func readPageFile(reader *bufio.Reader) (pageInfo, error) {
	nodes := make(map[int64]zones)

	buddy := buddyInfo{
		DMA:    make(map[int]int64),
		DMA32:  make(map[int]int64),
		Normal: make(map[int]int64),
	}

	for {
		raw, err := reader.ReadString('\n')

		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}

		var nodeLevel int64
		var zoneType string
		var migrateType string
		zoneOrders := make(map[int]int64)
		matches := pageinfoLine.FindAllSubmatch([]byte(raw), -1)
		//These procfs files aren't strictly defined and content can change. Assume no matches is fine.
		if matches == nil {
			continue
		}

		// Here's what a match looks like coming out of `FindAllSubmatch`
		// ['Node    0, zone      DMA, type    Unmovable      1      0      1      0      2      1      1      0      1      0      0' '0' 'DMA' 'Unmovable' '1' '0' '1' '0' '2' '1' '1' '0' '1' '0' '0']
		// '0' 'DMA' 'Unmovable' '1''0''1''0''2''1''1''0''1''0''0'
		match := matches[0]
		nodeLevel, err = strconv.ParseInt(string(match[1]), 10, 64)
		if err != nil {
			return pageInfo{}, fmt.Errorf("error parsing node number: %s: %w", string(match[1]), err)
		}
		if nodes[nodeLevel].DMA == nil {
			nodes[nodeLevel] = zones{
				DMA:          make(map[string]map[int]int64),
				DMA32:        make(map[string]map[int]int64),
				Normal:       make(map[string]map[int]int64),
				OrderSummary: make(map[int]int64),
			}
		}

		zoneType = string(match[2])
		migrateType = string(match[3])
		//Iterate over the order counts
		for order, count := range match[4:] {
			zoneOrders[order], err = strconv.ParseInt(string(count), 10, 64)
			if err != nil {
				return pageInfo{}, fmt.Errorf("error parsing zone: %s: %w", string(count), err)
			}
			nodes[nodeLevel].OrderSummary[order] += zoneOrders[order]
			if zoneType == "DMA" {
				buddy.DMA[order] += zoneOrders[order]
			} else if zoneType == "DMA32" {
				buddy.DMA32[order] += zoneOrders[order]
			} else if zoneType == "Normal" {
				buddy.Normal[order] += zoneOrders[order]
			}
		}

		if zoneType == "DMA" {
			nodes[nodeLevel].DMA[migrateType] = zoneOrders
		} else if zoneType == "DMA32" {
			nodes[nodeLevel].DMA32[migrateType] = zoneOrders
		} else if zoneType == "Normal" {
			nodes[nodeLevel].Normal[migrateType] = zoneOrders
		}
	} // end line loop

	return pageInfo{Zones: nodes, BuddyInfo: buddy}, nil
}
