// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"time"
)

type config struct {
	Interval        time.Duration `config:"interval" validate:"required"`
	Auth            *authConfig   `config:"auth"`
	URL             string        `config:"url" validate:"required"`
	Version         int           `config:"version" validate:"required"`
	Query           *QueryConfig  `config:"query"`
	InitialInterval time.Duration `config:"initial_interval"`
	From            string        `config:"from"`
	Cursor          *cursorConfig `config:"cursor"`
}

type cursorConfig struct {
	Field string `config:"field"`
}

func (c *config) Validate() error {
	switch {
	case c.URL == "":
		return errors.New("no instance url was configured or detected")
	case c.Interval == 0:
		return fmt.Errorf("please provide a valid interval %d", c.Interval)
	case c.Version < 46:
		// * EventLogFile object is available in API version 32.0 or later
		// * SetupAuditTrail object is available in API version 15.0 or later
		// * Real-Time Event monitoring objects that were introduced as part of
		// the beta release in API version 46.0
		//
		// To keep things simple, only one version is entertained i.e., the
		// minimum version supported by all objects for which we have support for.
		//
		// min_vesion_support_all[32.0, 15.0, 46.0] = 46.0
		//
		// (objects like EventLogFile, SetupAuditTrail and Real-time monitoring
		// objects are available in v46.0)

		// References:
		// https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_eventlogfile.htm
		// https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_setupaudittrail.htm
		// https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/platform_events_objects_monitoring.htm
		return fmt.Errorf("please provide a valid version i.e., 46.0 or above")
	}

	return nil
}

type QueryConfig struct {
	Default *valueTpl `config:"default"`
	Value   *valueTpl `config:"value"`
}
