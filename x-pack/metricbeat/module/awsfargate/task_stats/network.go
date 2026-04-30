// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package task_stats

import (
	dcontainer "github.com/docker/docker/api/types/container"
)

type networkStats struct {
	NameInterface string
	Total         dcontainer.NetworkStats
}

func getNetworkStats(taskStats dcontainer.StatsResponse) []networkStats {
	var networks []networkStats
	for nameInterface, rawNetStats := range taskStats.Networks {
		networks = append(networks, networkStats{
			NameInterface: nameInterface,
			Total:         rawNetStats,
		})
	}
	return networks
}
