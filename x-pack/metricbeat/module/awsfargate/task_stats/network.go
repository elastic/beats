// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import "github.com/docker/docker/api/types"

type networkStats struct {
	NameInterface string
	Total         types.NetworkStats
}

func getNetworkStats(taskStats types.StatsJSON) []networkStats {
	var networks []networkStats
	for nameInterface, rawNetStats := range taskStats.Networks {
		networks = append(networks, networkStats{
			NameInterface: nameInterface,
			Total:         rawNetStats,
		})
	}
	return networks
}
