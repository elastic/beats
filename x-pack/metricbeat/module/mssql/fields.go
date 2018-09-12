// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package mssql

import (
	"github.com/elastic/beats/libbeat/asset"
)

func init() {
	if err := asset.SetFields("metricbeat", "mssql", Asset); err != nil {
		panic(err)
	}
}

// Asset returns asset data
func Asset() string {
	return "eJx8j0GuwiAYhPecYtJ9L8Di7d4pjItGRvOnUBBobG9vCkbbpjo7ZuD7Qoues4ZL6W4VkCVbajTl3CjAMF2ihCx+0PhTAMApMIrjkDt7OqvSlftw3oyWCrgKrUm6TC2GzvGjWJLnQI1b9GN4NQeeLWaNEv+ujlhfeTWb13vFWsOpc6H8Z53q6zk/fDS77Yd1yX8FVql6BgAA//8fgmN/"
}
