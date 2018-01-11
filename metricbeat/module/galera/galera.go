/*
Package galera is Metricbeat module for Galera Cluster.
*/

package galera

import (
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mysql"
)

func init() {
	if err := mb.Registry.AddModule("galera", mysql.NewModule); err != nil  {
		panic(err)
	}
}
