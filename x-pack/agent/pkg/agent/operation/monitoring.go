// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/x-pack/agent/pkg/agent/configrequest"
	"github.com/elastic/beats/x-pack/agent/pkg/agent/errors"
	"github.com/elastic/beats/x-pack/agent/pkg/core/logger"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app"
	"github.com/elastic/beats/x-pack/agent/pkg/core/plugin/app/monitoring"
)

const (
	monitoringName          = "FLEET_MONITORING"
	settingsKey             = "settings"
	monitoringKey           = "monitoring"
	outputKey               = "output"
	monitoringEnabledSubkey = "enabled"
)

func (o *Operator) handleStartSidecar(s configrequest.Step) (result error) {
	cfg, err := getConfigFromStep(s)
	if err != nil {
		return errors.New(err,
			errors.TypeConfig,
			"operator.handleStartSidecar failed to retrieve config from step")
	}

	// if monitoring is disabled and running stop it
	if isEnabled := isMonitoringEnabled(o.logger, cfg); !isEnabled {
		if o.isMonitoring {
			o.logger.Info("operator.handleStartSidecar: monitoring is running and disabled, proceeding to stop")
			return o.handleStopSidecar(s)
		}

		o.logger.Info("operator.handleStartSidecar: monitoring is not running and disabled, no action taken")
		return nil
	}

	o.isMonitoring = true

	for _, step := range o.getMonitoringSteps(s) {
		p, cfg, err := getProgramFromStepWithTags(step, o.config.DownloadConfig, monitoringTags())
		if err != nil {
			return errors.New(err,
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, step.Process),
				"operator.handleStartSidecar failed to create program")
		}

		// best effort on starting monitoring, if no hosts provided stop and spare resources
		if step.ID == configrequest.StepRemove {
			if err := o.stop(p); err != nil {
				result = multierror.Append(err, err)
			}
		} else {
			if err := o.start(p, cfg); err != nil {
				result = multierror.Append(err, err)
			}
		}
	}

	return result
}

func (o *Operator) handleStopSidecar(s configrequest.Step) (result error) {
	for _, step := range o.getMonitoringSteps(s) {
		p, _, err := getProgramFromStepWithTags(step, o.config.DownloadConfig, monitoringTags())
		if err != nil {
			return errors.New(err,
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, step.Process),
				"operator.handleStopSidecar failed to create program")
		}

		if err := o.stop(p); err != nil {
			result = multierror.Append(err, err)
		}
	}

	// if result != nil then something might be still running, setting isMonitoring to false
	// will prevent tearing it down in a future
	if result == nil {
		o.isMonitoring = false
	}

	return result
}

func monitoringTags() map[app.Tag]string {
	return map[app.Tag]string{
		app.TagSidecar: "true",
	}
}

func isMonitoringEnabled(logger *logger.Logger, cfg map[string]interface{}) bool {
	settingsVal, found := cfg[settingsKey]
	if !found {
		logger.Error("operator.isMonitoringEnabled: settings not found in config")
		return false
	}

	settingsMap, ok := settingsVal.(map[string]interface{})
	if !ok {
		logger.Error("operator.isMonitoringEnabled: settings not a map")
		return false
	}

	monitoringVal, found := settingsMap[monitoringKey]
	if !found {
		logger.Error("operator.isMonitoringEnabled: settings.monitoring not found in config")
		return false
	}

	monitoringMap, ok := monitoringVal.(map[string]interface{})
	if !ok {
		logger.Error("operator.isMonitoringEnabled: settings.monitoring not a map")
		return false
	}

	enabledVal, found := monitoringMap[monitoringEnabledSubkey]
	if !found {
		logger.Infof("operator.isMonitoringEnabled: monitoring.enabled key not found: %v", monitoringMap)
		return false
	}

	enabled, ok := enabledVal.(bool)

	return enabled && ok
}

func (o *Operator) getMonitoringSteps(step configrequest.Step) []configrequest.Step {
	// get output
	config, err := getConfigFromStep(step)
	if err != nil {
		o.logger.Error("operator.getMonitoringSteps: getting config from step failed: %v", err)
		return nil
	}

	outputIface, found := config[outputKey]
	if !found {
		o.logger.Errorf("operator.getMonitoringSteps: monitoring configuration not found for sidecar of type %s", step.Process)
		return nil
	}

	outputMap, ok := outputIface.(map[string]interface{})
	if !ok {
		o.logger.Error("operator.getMonitoringSteps: monitoring config is not a map")
		return nil
	}

	output, found := outputMap["elasticsearch"]
	if !found {
		o.logger.Error("operator.getMonitoringSteps: monitoring is missing an elasticsearch output configuration configuration for sidecar of type: %s", step.Process)
		return nil
	}

	return o.generateMonitoringSteps(o.config.MonitoringConfig, step.Version, output)
}

func (o *Operator) generateMonitoringSteps(cfg *monitoring.Config, version string, output interface{}) []configrequest.Step {
	var steps []configrequest.Step

	if cfg.MonitorLogs {
		fbConfig, any := o.getMonitoringFilebeatConfig(output)
		stepID := configrequest.StepRun
		if !any {
			stepID = configrequest.StepRemove
		}
		filebeatStep := configrequest.Step{
			ID:      stepID,
			Version: version,
			Process: "filebeat",
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: fbConfig,
			},
		}

		steps = append(steps, filebeatStep)
	}

	if cfg.MonitorMetrics {
		mbConfig, any := o.getMonitoringMetricbeatConfig(output)
		stepID := configrequest.StepRun
		if !any {
			stepID = configrequest.StepRemove
		}

		metricbeatStep := configrequest.Step{
			ID:      stepID,
			Version: version,
			Process: "metricbeat",
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: mbConfig,
			},
		}

		steps = append(steps, metricbeatStep)
	}

	return steps
}

func (o *Operator) getMonitoringFilebeatConfig(output interface{}) (map[string]interface{}, bool) {
	paths := o.getLogFilePaths()
	if len(paths) == 0 {
		return nil, false
	}

	result := map[string]interface{}{
		"filebeat": map[string]interface{}{
			"inputs": []interface{}{
				map[string]interface{}{
					"type":  "log",
					"paths": paths,
				},
			},
		},
		"output": map[string]interface{}{
			"elasticsearch": output,
		},
	}

	return result, true
}

func (o *Operator) getMonitoringMetricbeatConfig(output interface{}) (map[string]interface{}, bool) {
	hosts := o.getMetricbeatEndpoints()
	if len(hosts) == 0 {
		return nil, false
	}

	result := map[string]interface{}{
		"metricbeat": map[string]interface{}{
			"modules": []interface{}{
				map[string]interface{}{
					"module":     "beat",
					"metricsets": []string{"stats", "state"},
					"period":     "10s",
					"hosts":      hosts,
				},
			},
		},
		"output": map[string]interface{}{
			"elasticsearch": output,
		},
	}

	return result, true
}

func (o *Operator) getLogFilePaths() []string {
	var paths []string

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		logPath := a.Monitor().LogPath()
		if logPath != "" {
			paths = append(paths, logPath)
		}
	}

	return paths
}

func (o *Operator) getMetricbeatEndpoints() []string {
	var endpoints []string

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		metricEndpoint := a.Monitor().MetricsPathPrefixed()
		if metricEndpoint != "" {
			endpoints = append(endpoints, metricEndpoint)
		}
	}

	return endpoints
}
