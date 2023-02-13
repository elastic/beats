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

	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/transform/typeconv"
	util "github.com/elastic/elastic-agent-system-metrics/metric"
	"github.com/elastic/elastic-agent-system-metrics/metric/memory"
	metrics "github.com/elastic/elastic-agent-system-metrics/metric/memory"
	"github.com/elastic/elastic-agent-system-metrics/metric/system/resolve"
)

// FetchLinuxMemStats gets page_stat and huge pages data for linux
func FetchLinuxMemStats(baseMap mapstr.M, hostfs resolve.Resolver) error {
	vmstat, err := GetVMStat(hostfs)
	if err != nil {
		return errors.Wrap(err, "error fetching VMStats")
	}

	pageStats := mapstr.M{}

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

	// huge pages swap out
	if thbswpout, ok := vmstat["thp_swpout"]; ok {
		baseMap.Put("hugepages.swap.out.pages", thbswpout)
	}
	if thbswpfall, ok := vmstat["thp_swpout_fallback"]; ok {
		baseMap.Put("hugepages.swap.out.fallback", thbswpfall)
	}

	// This is largely for convenience, and allows the swap.* metrics to more closely emulate how they're reported on system/memory
	// This way very similar metrics aren't split across different modules, even though Linux reports them in different places.
	eventRaw, err := metrics.Get(hostfs)
	if err != nil {
		return errors.Wrap(err, "error fetching memory metrics")
	}
	swap := mapstr.M{}
	err = typeconv.Convert(&swap, &eventRaw.Swap)
	if err != nil {
		return errors.Wrap(err, "error converting raw event")
	}

	baseMap["swap"] = swap

	// linux-exclusive swap data
	map2evt("pswpin", "swap.in.pages", vmstat, baseMap)
	map2evt("pswpout", "swap.out.pages", vmstat, baseMap)
	map2evt("swap_ra", "swap.readahead.pages", vmstat, baseMap)
	map2evt("swap_ra_hit", "swap.readahead.cached", vmstat, baseMap)

	baseMap["vmstat"] = vmstat

	return nil
}

func map2evt(inName string, outName string, rawEvt map[string]uint64, outEvt mapstr.M) {
	if selected, ok := rawEvt[inName]; ok {
		outEvt.Put(outName, selected)
	}
}

// insertPagesChild inserts a "child" MapStr into given events. This is mostly so we don't break mapping for fields that have been around.
// most of the fields in vmstat are fairly esoteric and (somewhat) self-documenting, so use of this shouldn't expand beyond what's needed for backwards compat.
func insertPagesChild(field string, raw map[string]uint64, evt mapstr.M) {
	stat, ok := raw[field]
	if ok {
		evt.Put(fmt.Sprintf("%s.pages", field), stat)
	}
}

func computeEfficiency(scanName string, stealName string, fieldName string, raw map[string]uint64, inMap mapstr.M) {
	scanVal, _ := raw[scanName]
	stealVal, stealOk := raw[stealName]
	if scanVal != 0 && stealOk {
		inMap[fieldName] = mapstr.M{
			"pct": util.Round(float64(stealVal) / float64(scanVal)),
		}
	}

}

func getHugePages(hostfs resolve.Resolver) (mapstr.M, error) {
	// see https://www.kernel.org/doc/Documentation/vm/hugetlbpage.txt
	table, err := memory.ParseMeminfo(hostfs)
	if err != nil {
		return nil, errors.Wrap(err, "error parsing meminfo")
	}
	thp := mapstr.M{}

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
		thp.Put("used.pct", util.Round(perc))

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
