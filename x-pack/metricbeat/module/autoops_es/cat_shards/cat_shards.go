// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package cat_shards

import (
	"github.com/elastic/beats/v7/x-pack/metricbeat/module/autoops_es/metricset"
)

const (
	CatShardsMetricSet = "cat_shards"
	CatShardsPath      = "/_cat/shards?s=i&h=n,i,id,s,p,st,d,sto,sc,sqto,sqti,iito,iiti,iif,mt,mtt,gmto,gmti,ur,ud&bytes=b&time=ms&format=json"
	ResolveIndexPath   = "/_resolve/index/*?expand_wildcards=all&filter_path=indices"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	metricset.AddNestedAutoOpsMetricSet(CatShardsMetricSet, CatShardsPath, eventsMapping)
}
