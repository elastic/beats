// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"time"
)

// Sample Config:
/*
- type: salesforce
  enabled: true
  version: 56
  auth.oauth2:
    enabled: false
    client.id: clientid
    client.secret: clientsecret
    token_url: https://instance_id.develop.my.salesforce.com/services/oauth2/token
    user: username
    password: password
  auth.jwt:
    enabled: true
    client.id: clientid
    client.username: username
    client.key_path: ./server_client.key
    url: https://login.salesforce.com
  url: https://instance_id.develop.my.salesforce.com
  data_collection_method:
    event_log_file:
      interval: 1h
      enabled: true
      query:
        default: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' ORDER BY CreatedDate ASC NULLS FIRST"
        value: "SELECT Id,CreatedDate,LogDate,LogFile FROM EventLogFile WHERE EventType = 'Login' AND CreatedDate > [[ .cursor.logdate ]] ORDER BY CreatedDate ASC NULLS FIRST"
      cursor:
        field: "CreatedDate"
    object:
      interval: 5m
      enabled: true
      query:
        default: "SELECT FIELDS(STANDARD) FROM LoginEvent"
        value: "SELECT FIELDS(STANDARD) FROM LoginEvent WHERE EventDate > [[ .cursor.logdate ]]"
      cursor:
        field: "EventDate"
*/
type config struct {
	Auth                 *authConfig           `config:"auth"`
	URL                  string                `config:"url" validate:"required"`
	Version              int                   `config:"version" validate:"required"`
	InitialInterval      time.Duration         `config:"initial_interval"`
	DataCollectionMethod *DataCollectionMethod `config:"data_collection_method"`
}

type DataCollectionMethod struct {
	EventLogFile EventLogFileMethod `config:"event_log_file"`
	Object       ObjectMethod       `config:"object"`
}

type EventLogFileMethod struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval" validate:"required"`
	Query    *QueryConfig  `config:"query"`
	Cursor   *cursorConfig `config:"cursor"`
}

type ObjectMethod struct {
	Enabled  bool          `config:"enabled"`
	Interval time.Duration `config:"interval" validate:"required"`
	Query    *QueryConfig  `config:"query"`
	Cursor   *cursorConfig `config:"cursor"`
}

type cursorConfig struct {
	Field string `config:"field"`
}

func (c *config) Validate() error {
	switch {
	case !c.Auth.JWT.isEnabled() && !c.Auth.OAuth2.isEnabled():
		return errors.New("no auth provider enabled")
	case c.Auth.JWT.isEnabled() && c.Auth.OAuth2.isEnabled():
		return errors.New("only one auth provider must be enabled")
	case c.URL == "":
		return errors.New("no instance url is configured")
	case !c.DataCollectionMethod.Object.Enabled && !c.DataCollectionMethod.EventLogFile.Enabled:
		return errors.New(`at least one of "data_collection_method.event_log_file.enabled" or "data_collection_method.object.enabled" must be set to true`)
	case c.DataCollectionMethod.EventLogFile.Enabled && c.DataCollectionMethod.EventLogFile.Interval == 0:
		return fmt.Errorf("not a valid interval %d", c.DataCollectionMethod.EventLogFile.Interval)
	case c.DataCollectionMethod.Object.Enabled && c.DataCollectionMethod.Object.Interval == 0:
		return fmt.Errorf("not a valid interval %d", c.DataCollectionMethod.Object.Interval)

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
