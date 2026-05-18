// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package salesforce

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/elastic/elastic-agent-libs/transport/httpcommon"
)

type config struct {
	Resource              *resourceConfig        `config:"resource"`
	Auth                  *authConfig            `config:"auth"`
	EventMonitoringMethod *eventMonitoringMethod `config:"event_monitoring_method"`
	URL                   string                 `config:"url" validate:"required"`
	Version               int                    `config:"version" validate:"required"`
}

type resourceConfig struct {
	Retry     retryConfig                      `config:"retry"`
	Transport httpcommon.HTTPTransportSettings `config:",inline"`
}

type retryConfig struct {
	MaxAttempts *int           `config:"max_attempts"`
	WaitMin     *time.Duration `config:"wait_min"`
	WaitMax     *time.Duration `config:"wait_max"`
}

func (c retryConfig) Validate() error {
	switch {
	case c.MaxAttempts != nil && *c.MaxAttempts <= 0:
		return errors.New("max_attempts must be greater than zero")
	case c.WaitMin != nil && *c.WaitMin <= 0:
		return errors.New("wait_min must be greater than zero")
	case c.WaitMax != nil && *c.WaitMax <= 0:
		return errors.New("wait_max must be greater than zero")
	}
	return nil
}

func (c retryConfig) getMaxAttempts() int {
	if c.MaxAttempts == nil {
		return 0
	}
	return *c.MaxAttempts
}

func (c retryConfig) getWaitMin() time.Duration {
	if c.WaitMin == nil {
		return 0
	}
	return *c.WaitMin
}

func (c retryConfig) getWaitMax() time.Duration {
	if c.WaitMax == nil {
		return 0
	}
	return *c.WaitMax
}

type eventMonitoringMethod struct {
	EventLogFile EventMonitoringConfig `config:"event_log_file"`
	Object       EventMonitoringConfig `config:"object"`
}

type EventMonitoringConfig struct {
	Enabled  *bool         `config:"enabled"`
	Batch    *batchConfig  `config:"batch"`
	Query    *QueryConfig  `config:"query"`
	Cursor   *cursorConfig `config:"cursor"`
	Interval time.Duration `config:"interval"`
}

func (e *EventMonitoringConfig) isEnabled() bool {
	return e != nil && (e.Enabled != nil && *e.Enabled)
}

// batchConfig controls bounded, windowed Object collection. Batching is
// opt-in and currently only applies to the Object method.
//
//   - Enabled          - master switch. When false the unbatched single-query
//     path in RunObject is used. When true, query.value must constrain the
//     SOQL with .cursor.object.batch_start_time and .cursor.object.batch_end_time.
//   - InitialInterval  - how far back the first-ever batched window extends
//     on a clean install (runEnd - InitialInterval). Ignored once any of
//     progress_time / first_event_time / last_event_time is persisted.
//   - MaxWindowsPerRun - upper bound on the number of (Start, End] windows a
//     single scheduler tick will process. Protects the Salesforce API from
//     runaway catch-up on a very old cursor; when nil, getMaxWindowsPerRun
//     returns 1 so a new batching config still makes forward progress.
//   - Window           - width of each (Start, End] slice.
type batchConfig struct {
	Enabled          *bool         `config:"enabled"`
	InitialInterval  time.Duration `config:"initial_interval"`
	MaxWindowsPerRun *int          `config:"max_windows_per_run"`
	Window           time.Duration `config:"window"`
}

// isEnabled reports whether bounded-batch Object collection is turned on. A
// nil receiver is treated as disabled so callers can write
// objectCfg.Batch.isEnabled() without a nil check.
func (b *batchConfig) isEnabled() bool {
	return b != nil && (b.Enabled != nil && *b.Enabled)
}

// getMaxWindowsPerRun returns the configured per-run window cap, defaulting
// to 1 when unset. Returning 1 rather than 0 guarantees forward progress on
// every tick even if the user omits the setting.
func (b *batchConfig) getMaxWindowsPerRun() int {
	if b == nil || b.MaxWindowsPerRun == nil {
		return 1
	}
	return *b.MaxWindowsPerRun
}

type cursorConfig struct {
	Field string `config:"field"`
}

// validateEnabledMethodConfig enforces the required query / cursor settings
// for an enabled collection method. path is the dotted config path
// (e.g. "event_monitoring_method.object") used to produce actionable error
// messages. A nil or disabled method is a no-op so callers can validate both
// methods unconditionally.
func validateEnabledMethodConfig(path string, method *EventMonitoringConfig) error {
	if method == nil || !method.isEnabled() {
		return nil
	}
	if method.Query == nil {
		return fmt.Errorf(`"%s.query" must be configured when "%s.enabled" is true`, path, path)
	}
	if method.Query.Default == nil {
		return fmt.Errorf(`"%s.query.default" must be configured when "%s.enabled" is true`, path, path)
	}
	if method.Query.Value == nil {
		return fmt.Errorf(`"%s.query.value" must be configured when "%s.enabled" is true`, path, path)
	}
	if method.Cursor == nil {
		return fmt.Errorf(`"%s.cursor" must be configured when "%s.enabled" is true`, path, path)
	}
	if method.Cursor.Field == "" {
		return fmt.Errorf(`"%s.cursor.field" must be configured when "%s.enabled" is true`, path, path)
	}
	return nil
}

// oauth2Config returns the OAuth2 sub-config, or nil when auth is not set.
// Used by Validate so the nil checks stay centralized.
func (c *config) oauth2Config() *OAuth2 {
	if c == nil || c.Auth == nil {
		return nil
	}
	return c.Auth.OAuth2
}

// objectMonitoringConfig returns the Object method config block, or nil
// when event_monitoring_method is not set.
func (c *config) objectMonitoringConfig() *EventMonitoringConfig {
	if c == nil || c.EventMonitoringMethod == nil {
		return nil
	}
	return &c.EventMonitoringMethod.Object
}

// eventLogFileMonitoringConfig returns the EventLogFile method config block,
// or nil when event_monitoring_method is not set.
func (c *config) eventLogFileMonitoringConfig() *EventMonitoringConfig {
	if c == nil || c.EventMonitoringMethod == nil {
		return nil
	}
	return &c.EventMonitoringMethod.EventLogFile
}

// Validate validates the configuration.
func (c *config) Validate() error {
	oauth2 := c.oauth2Config()
	objectMethod := c.objectMonitoringConfig()
	eventLogFileMethod := c.eventLogFileMonitoringConfig()

	switch {
	case oauth2 == nil || (!oauth2.JWTBearerFlow.isEnabled() && !oauth2.UserPasswordFlow.isEnabled()):
		return errors.New("no auth provider enabled")
	case oauth2.JWTBearerFlow.isEnabled() && oauth2.UserPasswordFlow.isEnabled():
		return errors.New("only one auth provider must be enabled")
	case c.URL == "":
		return errors.New("no instance url is configured")
	case !objectMethod.isEnabled() && !eventLogFileMethod.isEnabled():
		return errors.New(`at least one of "event_monitoring_method.event_log_file.enabled" or "event_monitoring_method.object.enabled" must be set to true`)
	}

	if eventLogFileMethod != nil && eventLogFileMethod.isEnabled() {
		if eventLogFileMethod.Interval == 0 {
			return fmt.Errorf("not a valid interval %d", eventLogFileMethod.Interval)
		}
		if err := validateEnabledMethodConfig("event_monitoring_method.event_log_file", eventLogFileMethod); err != nil {
			return err
		}
	}

	if objectMethod != nil && objectMethod.isEnabled() {
		if objectMethod.Interval == 0 {
			return fmt.Errorf("not a valid interval %d", objectMethod.Interval)
		}
		usesBatchStart, usesBatchEnd := objectMethod.Query.valueUsesObjectBatchWindow()
		if objectMethod.Batch.isEnabled() && objectMethod.Batch.InitialInterval <= 0 {
			return errors.New(`"event_monitoring_method.object.batch.initial_interval" must be greater than zero`)
		}
		if objectMethod.Batch.isEnabled() && objectMethod.Batch.Window <= 0 {
			return errors.New(`"event_monitoring_method.object.batch.window" must be greater than zero`)
		}
		if objectMethod.Batch.isEnabled() && objectMethod.Batch.getMaxWindowsPerRun() <= 0 {
			return errors.New(`"event_monitoring_method.object.batch.max_windows_per_run" must be greater than zero`)
		}
		if err := validateEnabledMethodConfig("event_monitoring_method.object", objectMethod); err != nil {
			return err
		}
		if objectMethod.Batch.isEnabled() && (!usesBatchStart || !usesBatchEnd) {
			return errors.New(`"event_monitoring_method.object.query.value" must reference both ".cursor.object.batch_start_time" and ".cursor.object.batch_end_time" when "event_monitoring_method.object.batch.enabled" is true`)
		}
		if !objectMethod.Batch.isEnabled() && (usesBatchStart || usesBatchEnd) {
			return errors.New(`"event_monitoring_method.object.query.value" must not reference ".cursor.object.batch_start_time" or ".cursor.object.batch_end_time" when "event_monitoring_method.object.batch.enabled" is false`)
		}
	}

	if c.Version < 46 {
		// - EventLogFile object is available in API version 32.0 or later
		// - SetupAuditTrail object is available in API version 15.0 or later
		// - Real-Time Event monitoring objects that were introduced as part of
		// the beta release in API version 46.0
		//
		// To keep things simple, only one version is entertained i.e., the
		// minimum version supported by all objects for which we have support
		// for.
		//
		// minimum_version_supported_by_all_objects([32.0, 15.0, 46.0]) = 46.0
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

const (
	objectBatchStartPlaceholder = ".cursor.object.batch_start_time"
	objectBatchEndPlaceholder   = ".cursor.object.batch_end_time"
)

// valueUsesObjectBatchWindow reports whether the object value query references
// the bounded-batch placeholders. Validation uses this to reject configs that
// enable batching without actually constraining the SOQL window, and the
// reverse mismatch where batching is disabled but the query still expects
// batch_start_time / batch_end_time to exist.
func (q *QueryConfig) valueUsesObjectBatchWindow() (usesStart, usesEnd bool) {
	if q == nil || q.Value == nil {
		return false, false
	}
	src := q.Value.Source()
	return strings.Contains(src, objectBatchStartPlaceholder), strings.Contains(src, objectBatchEndPlaceholder)
}
