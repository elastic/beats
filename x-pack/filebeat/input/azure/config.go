// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

var defaultConfig = config{
	YieldEventsFromField: "records",
}

type config struct {
	// YieldEventsFromField (i)
	YieldEventsFromField string `config:"yield_events_from_field"`
}
