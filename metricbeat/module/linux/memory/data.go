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
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/transform/typeconv"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/internal/metrics/memory"
	metrics "github.com/elastic/beats/v7/metricbeat/internal/metrics/memory"
	sysinfo "github.com/elastic/go-sysinfo"
	sysinfotypes "github.com/elastic/go-sysinfo/types"
)

// FetchLinuxMemStats gets page_stat and huge pages data for linux
func FetchLinuxMemStats(baseMap common.MapStr, hostfs resolve.Resolver) error {

	vmstat, err := GetVMStat()
	if err != nil {
		return errors.Wrap(err, "error fetching VMStats")
	}

	pageStats := common.MapStr{
		"pgscan_kswapd": common.MapStr{
			"pages": vmstat.PgscanKswapd,
		},
		"pgscan_direct": common.MapStr{
			"pages": vmstat.PgscanDirect,
		},
		"pgfree": common.MapStr{
			"pages": vmstat.Pgfree,
		},
		"pgsteal_kswapd": common.MapStr{
			"pages": vmstat.PgstealKswapd,
		},
		"pgsteal_direct": common.MapStr{
			"pages": vmstat.PgstealDirect,
		},
	}
	// This is similar to the vmeff stat gathered by sar
	// these ratios calculate thhe efficiency of page reclaim
	if vmstat.PgscanDirect != 0 {
		pageStats["direct_efficiency"] = common.MapStr{
			"pct": common.Round(float64(vmstat.PgstealDirect)/float64(vmstat.PgscanDirect), common.DefaultDecimalPlacesCount),
		}
	}

	if vmstat.PgscanKswapd != 0 {
		pageStats["kswapd_efficiency"] = common.MapStr{
			"pct": common.Round(float64(vmstat.PgstealKswapd)/float64(vmstat.PgscanKswapd), common.DefaultDecimalPlacesCount),
		}
	}
	baseMap["page_stats"] = pageStats

	thp, err := getHugePages(hostfs)
	if err != nil {
		return errors.Wrap(err, "error getting huge pages")
	}
	thp["swap"] = common.MapStr{
		"out": common.MapStr{
			"pages":    vmstat.ThpSwpout,
			"fallback": vmstat.ThpSwpoutFallback,
		},
	}

	// This is largely for convenience, and allows the swap.* metrics to more closely emulate how they're reported on system/memory
	// This way very similar metrics aren't split across different modules, even though Linux reports them in different places.
	eventRaw, err := metrics.Get(hostfs)
	if err != nil {
		return errors.Wrap(err, "error fetching memory metrics")
	}
	swap := common.MapStr{}
	err = typeconv.Convert(&swap, &eventRaw.Swap)
	swap.Put("in.pages", vmstat.Pswpin)
	swap.Put("out.pages", vmstat.Pswpout)
	swap.Put("readahead.pages", vmstat.SwapRa)
	swap.Put("readahead.cached", vmstat.SwapRaHit)

	baseMap["swap"] = swap

	baseMap["hugepages"] = thp

	return nil
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
func GetVMStat() (*sysinfotypes.VMStatInfo, error) {
	// TODO: We may want to pull this code out of go-sysinfo.
	// It's platform specific, and not used by anything else.
	h, err := sysinfo.Host()
	if err != nil {
		return nil, errors.Wrap(err, "failed to read self process information")
	}
	vmstatHandle, ok := h.(sysinfotypes.VMStat)
	if !ok {
		return nil, errors.New("VMStat not available for platform")
	}
	info, err := vmstatHandle.VMStat()
	if err != nil {
		return nil, errors.Wrap(err, "error getting VMStat info")
	}
	return info, nil

}
