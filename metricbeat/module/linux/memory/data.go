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

package memory

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/internal/metrics/memory"
)

// FetchLinuxMemStats gets page_stat and huge pages data for linux
func FetchLinuxMemStats(baseMap common.MapStr, hostfs resolve.Resolver) error {
	vmstat, err := GetVMStat(hostfs)
	if err != nil {
		return errors.Wrap(err, "error fetching VMStats")
	}

	pageStats := common.MapStr{}

	insertPagesChild("pgscan_kswapd", vmstat, pageStats)
	insertPagesChild("pgscan_direct", vmstat, pageStats)
	insertPagesChild("pgfree", vmstat, pageStats)
	insertPagesChild("pgsteal_kswapd", vmstat, pageStats)
	insertPagesChild("pgsteal_direct", vmstat, pageStats)

	computeEfficiency("pgscan_direct", "pgsteal_direct", "direct_efficiency", vmstat, pageStats)
	computeEfficiency("pgscan_kswapd", "pgsteal_kswapd", "kswapd_efficiency", vmstat, pageStats)

	baseMap["page_stats"] = pageStats

	thp, err := getHugePages(hostfs)
	if err != nil {
		return errors.Wrap(err, "error getting huge pages")
	}
	baseMap["hugepages"] = thp

	if thbswpout, ok := vmstat["thp_swpout"]; ok {
		baseMap.Put("hugepages.swap.out.pages", thbswpout)
	}
	if thbswpfall, ok := vmstat["thp_swpout_fallback"]; ok {
		baseMap.Put("hugepages.swap.out.fallback", thbswpfall)
	}

	baseMap["vmstat"] = vmstat

	return nil
}

func computeEfficiency(scanName string, stealName string, fieldName string, raw map[string]uint64, inMap common.MapStr) {
	scanVal, _ := raw[scanName]
	stealVal, stealOk := raw[stealName]
	if scanVal != 0 && stealOk {
		inMap[fieldName] = common.MapStr{
			"pct": common.Round(float64(stealVal)/float64(scanVal), common.DefaultDecimalPlacesCount),
		}
	}

}

// insertPagesChild inserts a "child" MapStr into given events. This is mostly so we don't break mapping for fields that have been around.
// most of the fields in vmstat are fairly esoteric and (somewhat) self-documenting, so use of this shouldn't expand beyond what's needed for backwards compat.
func insertPagesChild(field string, raw map[string]uint64, evt common.MapStr) {
	stat, ok := raw[field]
	if ok {
		evt.Put(fmt.Sprintf("%s.pages", field), stat)
	}
}

func getHugePages(hostfs resolve.Resolver) (common.MapStr, error) {
	// see https://www.kernel.org/doc/Documentation/vm/hugetlbpage.txt
	table, err := memory.ParseMeminfo(hostfs)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing meminfo")
	}
	thp := common.MapStr{}

	total, okTotal := table["HugePages_Total"]
	free, okFree := table["HugePages_Free"]
	reserved, okReserved := table["HugePages_Rsvd"]
	totalSize, okTotalSize := table["Hugetlb"]
	defaultSize, okDefaultSize := table["Hugepagesize"]

	// Calculate percentages
	if okTotal && okFree && okReserved {
		thp.Put("total", total)
		thp.Put("free", free)
		thp.Put("reserved", reserved)

		// TODO: this repliactes the behavior of metricbeat in the past,
		// but it might be possilbe to do something like (HugePages_Total*Hugepagesize)-Hugetlb / (HugePages_Total*Hugepagesize)
		var perc float64
		if total > 0 {
			perc = float64(total-free+reserved) / float64(total)
		}
		thp.Put("used.pct", common.Round(perc, common.DefaultDecimalPlacesCount))

		if !okTotalSize && okDefaultSize {
			thp.Put("used.bytes", (total-free+reserved)*defaultSize)
		}
	}
	if okTotalSize {
		thp.Put("used.bytes", totalSize)
	}
	if okDefaultSize {
		thp.Put("default_size", defaultSize)
	}
	if surplus, ok := table["HugePages_Surp"]; ok {
		thp.Put("surplus", surplus)
	}

	return thp, nil
}

// GetVMStat gets linux vmstat metrics
func GetVMStat(hostfs resolve.Resolver) (map[string]uint64, error) {
	vmstatFile := hostfs.ResolveHostFS("proc/vmstat")
	content, err := os.ReadFile(vmstatFile)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading vmstat from %s", vmstatFile)
	}

	// I'm not a fan of throwing stuff directly to maps, but this is a huge amount of kernel/config specific metrics, and we're the only consumer of this for now.
	vmstat := map[string]uint64{}
	for _, metric := range strings.Split(string(content), "\n") {
		parts := strings.SplitN(metric, " ", 2)
		if len(parts) != 2 {
			continue
		}

		num, err := strconv.ParseUint(string(parts[1]), 10, 64)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to parse value %s", parts[1])
		}
		vmstat[parts[0]] = num

	}

	return vmstat, nil
}
