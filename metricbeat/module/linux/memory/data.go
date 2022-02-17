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
	"io/ioutil"
	"reflect"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/metric/system/resolve"
	"github.com/elastic/beats/v7/metricbeat/internal/metrics/memory"
	"github.com/elastic/go-sysinfo/types"
)

// vmstatTagToFieldIndex contains a mapping of json struct tags to struct field indices.
var vmstatTagToFieldIndex = make(map[string]int)

// A little helper so we only have to initialize the little reflection maps we use to populate VMstat data once
func init() {
	var vmstat types.VMStatInfo
	val := reflect.ValueOf(vmstat)
	typ := reflect.TypeOf(vmstat)

	for i := 0; i < val.NumField(); i++ {
		field := typ.Field(i)
		if tag := field.Tag.Get("json"); tag != "" {
			vmstatTagToFieldIndex[tag] = i
		}
	}
}

// FetchLinuxMemStats gets page_stat and huge pages data for linux
func FetchLinuxMemStats(baseMap common.MapStr, hostfs resolve.Resolver) error {
	vmstat, err := GetVMStat(hostfs)
	if err != nil {
		return errors.Wrap(err, "error fetching VMStats")
	}

	pageStats := common.MapStr{}
	if pages, ok := vmstat["pgscan_kswapd"]; ok {
		pageStats.Put("pgscan_kswapd.pages", pages)
	}
	if pages, ok := vmstat["pgscan_direct"]; ok {
		pageStats.Put("pgscan_direct.pages", pages)
	}
	if pages, ok := vmstat["pgfree"]; ok {
		pageStats.Put("pgfree.pages", pages)
	}
	if pages, ok := vmstat["pgsteal_kswapd"]; ok {
		pageStats.Put("pgsteal_kswapd.pages", pages)
	}
	if pages, ok := vmstat["pgsteal_direct"]; ok {
		pageStats.Put("pgsteal_direct.pages", pages)
	}

	// This is similar to the vmeff stat gathered by sar
	// these ratios calculate thhe efficiency of page reclaim
	pgscan, _ := vmstat["pgscan_direct"]
	pgsteal, pgsteal_ok := vmstat["pgsteal_direct"]
	if pgscan != 0 && pgsteal_ok {
		pageStats["direct_efficiency"] = common.MapStr{
			"pct": common.Round(float64(pgsteal)/float64(pgscan), common.DefaultDecimalPlacesCount),
		}
	}

	pgscankswap, _ := vmstat["pgscan_kswapd"]
	pgstealkswap, pgstealswap_ok := vmstat["pgsteal_kswapd"]
	if pgscankswap != 0 && pgstealswap_ok {
		pageStats["kswapd_efficiency"] = common.MapStr{
			"pct": common.Round(float64(pgstealkswap)/float64(pgscankswap), common.DefaultDecimalPlacesCount),
		}
	}
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

	baseMap["hugepages"] = thp
	baseMap["vmstat"] = vmstat

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
func GetVMStat(hostfs resolve.Resolver) (map[string]uint64, error) {
	vmstat_file := hostfs.ResolveHostFS("proc/vmstat")
	content, err := ioutil.ReadFile(vmstat_file)
	if err != nil {
		return nil, errors.Wrapf(err, "error reading vmstat from %s", vmstat_file)
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
			return nil, errors.Wrap(err, "failed to parse value")
		}
		vmstat[parts[0]] = num

	}

	return vmstat, nil
}
