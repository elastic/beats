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

// +build darwin freebsd linux openbsd windows

package memory

import (
	"github.com/pkg/errors"

	"github.com/elastic/beats/v7/libbeat/common"
	mem "github.com/elastic/beats/v7/libbeat/metric/system/memory"
)

// FetchLinuxMemStats gets page_stat and huge pages data for linux
func FetchLinuxMemStats(baseMap common.MapStr) error {

	vmstat, err := mem.GetVMStat()
	if err != nil {
		return errors.Wrap(err, "VMStat")
	}

	if vmstat != nil {
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
	}

	hugePagesStat, err := mem.GetHugeTLBPages()
	if err != nil {
		return errors.Wrap(err, "hugepages")
	}
	if hugePagesStat != nil {
		mem.AddHugeTLBPagesPercentage(hugePagesStat)
		thp := common.MapStr{
			"total": hugePagesStat.Total,
			"used": common.MapStr{
				"bytes": hugePagesStat.TotalAllocatedSize,
				"pct":   hugePagesStat.UsedPercent,
			},
			"free":         hugePagesStat.Free,
			"reserved":     hugePagesStat.Reserved,
			"surplus":      hugePagesStat.Surplus,
			"default_size": hugePagesStat.DefaultSize,
		}
		if vmstat != nil {
			thp["swap"] = common.MapStr{
				"out": common.MapStr{
					"pages":    vmstat.ThpSwpout,
					"fallback": vmstat.ThpSwpoutFallback,
				},
			}
		}
		baseMap["hugepages"] = thp
	}
	return nil
}
