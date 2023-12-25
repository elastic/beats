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
	Auth                  *authConfig            `config:"auth"`
	URL                   string                 `config:"url" validate:"required"`
	Version               int                    `config:"version" validate:"required"`
	InitialInterval       time.Duration          `config:"initial_interval"`
	EventMonitoringMethod *EventMonitoringMethod `config:"event_monitoring_method"`
}

type EventMonitoringMethod struct {
	EventLogFile EventMonitoringConfig `config:"event_log_file"`
	Object       EventMonitoringConfig `config:"object"`
}

type EventMonitoringConfig struct {
	Enabled  *bool         `config:"enabled"`
	Interval time.Duration `config:"interval"`
	Query    *QueryConfig  `config:"query"`
	Cursor   *cursorConfig `config:"cursor"`
}

func (e *EventMonitoringConfig) isEnabled() bool {
	return e != nil && (e.Enabled != nil && *e.Enabled)
}

type cursorConfig struct {
	Field string `config:"field"`
}

// Validate validates the configuration.
func (c *config) Validate() error {
	switch {
	case !c.Auth.OAuth2.JWTBearerFlow.isEnabled() && !c.Auth.OAuth2.UserPasswordFlow.isEnabled():
		return errors.New("no auth provider enabled")
	case c.Auth.OAuth2.JWTBearerFlow.isEnabled() && c.Auth.OAuth2.UserPasswordFlow.isEnabled():
		return errors.New("only one auth provider must be enabled")
	case c.URL == "":
		return errors.New("no instance url is configured")
	case !c.EventMonitoringMethod.Object.isEnabled() && !c.EventMonitoringMethod.EventLogFile.isEnabled():
		return errors.New(`at least one of "event_monitoring_method.event_log_file.enabled" or "event_monitoring_method.object.enabled" must be set to true`)
	case c.EventMonitoringMethod.EventLogFile.isEnabled() && c.EventMonitoringMethod.EventLogFile.Interval == 0:
		return fmt.Errorf("not a valid interval %d", c.EventMonitoringMethod.EventLogFile.Interval)
	case c.EventMonitoringMethod.Object.isEnabled() && c.EventMonitoringMethod.Object.Interval == 0:
		return fmt.Errorf("not a valid interval %d", c.EventMonitoringMethod.Object.Interval)

	case c.Version < 46:
		// * EventLogFile object is available in API version 32.0 or later
		// * SetupAuditTrail object is available in API version 15.0 or later
		// * Real-Time Event monitoring objects that were introduced as part of
		// the beta release in API version 46.0
		//
		// To keep things simple, only one version is entertained i.e., the
		// minimum version supported by all objects for which we have support
		// for.
		//
		// minimum_vesion_supported_by_all_objects([32.0, 15.0, 46.0]) = 46.0
		//
		// (Objects like EventLogFile, SetupAuditTrail and Real-time monitoring
		// objects are available in v46.0 and above)

		// References:
		// https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_eventlogfile.htm
		// https://developer.salesforce.com/docs/atlas.en-us.object_reference.meta/object_reference/sforce_api_objects_setupaudittrail.htm
		// https://developer.salesforce.com/docs/atlas.en-us.platform_events.meta/platform_events/platform_events_objects_monitoring.htm
		return errors.New("not a valid version i.e., 46.0 or above")
	}

	return nil
}

type QueryConfig struct {
	Default *valueTpl `config:"default"`
	Value   *valueTpl `config:"value"`
}
