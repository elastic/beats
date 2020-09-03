// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"path/filepath"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
)

const (
	monitoringName     = "FLEET_MONITORING"
	outputKey          = "output"
	logsProcessName    = "filebeat"
	metricsProcessName = "metricbeat"
	artifactPrefix     = "beats"
)

func (o *Operator) handleStartSidecar(s configrequest.Step) (result error) {
	// if monitoring is disabled and running stop it
	if !o.monitor.IsMonitoringEnabled() {
		if o.isMonitoring != 0 {
			o.logger.Info("operator.handleStartSidecar: monitoring is running and disabled, proceeding to stop")
			return o.handleStopSidecar(s)
		}

		o.logger.Info("operator.handleStartSidecar: monitoring is not running and disabled, no action taken")
		return nil
	}

	for _, step := range o.getMonitoringSteps(s) {
		p, cfg, err := getProgramFromStepWithTags(step, o.config.DownloadConfig, monitoringTags())
		if err != nil {
			return errors.New(err,
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, step.ProgramSpec.Cmd),
				"operator.handleStartSidecar failed to create program")
		}

		// best effort on starting monitoring, if no hosts provided stop and spare resources
		if step.ID == configrequest.StepRemove {
			if err := o.stop(p); err != nil {
				result = multierror.Append(err, err)
			} else {
				o.markStopMonitoring(step.ProgramSpec.Cmd)
			}
		} else {
			if err := o.start(p, cfg); err != nil {
				result = multierror.Append(err, err)
			} else {
				o.markStartMonitoring(step.ProgramSpec.Cmd)
			}
		}
	}

	return result
}

func (o *Operator) handleStopSidecar(s configrequest.Step) (result error) {
	for _, step := range o.generateMonitoringSteps(s.Version, nil) {
		p, _, err := getProgramFromStepWithTags(step, o.config.DownloadConfig, monitoringTags())
		if err != nil {
			return errors.New(err,
				errors.TypeApplication,
				errors.M(errors.MetaKeyAppName, step.ProgramSpec.Cmd),
				"operator.handleStopSidecar failed to create program")
		}

		o.logger.Debugf("stopping program %v", p)
		if err := o.stop(p); err != nil {
			result = multierror.Append(err, err)
		} else {
			o.markStopMonitoring(step.ProgramSpec.Cmd)
		}
	}

	return result
}

func monitoringTags() map[app.Tag]string {
	return map[app.Tag]string{
		app.TagSidecar: "true",
	}
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
		o.logger.Errorf("operator.getMonitoringSteps: monitoring configuration not found for sidecar of type %s", step.ProgramSpec.Cmd)
		return nil
	}

	outputMap, ok := outputIface.(map[string]interface{})
	if !ok {
		o.logger.Error("operator.getMonitoringSteps: monitoring config is not a map")
		return nil
	}

	output, found := outputMap["elasticsearch"]
	if !found {
		o.logger.Error("operator.getMonitoringSteps: monitoring is missing an elasticsearch output configuration configuration for sidecar of type: %s", step.ProgramSpec.Cmd)
		return nil
	}

	return o.generateMonitoringSteps(step.Version, output)
}

func (o *Operator) generateMonitoringSteps(version string, output interface{}) []configrequest.Step {
	var steps []configrequest.Step
	watchLogs := o.monitor.WatchLogs()
	watchMetrics := o.monitor.WatchMetrics()

	// generate only on change
	if watchLogs != o.isMonitoringLogs() {
		fbConfig, any := o.getMonitoringFilebeatConfig(output)
		stepID := configrequest.StepRun
		if !watchLogs || !any {
			stepID = configrequest.StepRemove
		}
		filebeatStep := configrequest.Step{
			ID:      stepID,
			Version: version,
			ProgramSpec: program.Spec{
				Name:     logsProcessName,
				Cmd:      logsProcessName,
				Artifact: fmt.Sprintf("%s/%s", artifactPrefix, logsProcessName),
			},
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: fbConfig,
			},
		}

		steps = append(steps, filebeatStep)
	}
	if watchMetrics != o.isMonitoringMetrics() {
		mbConfig, any := o.getMonitoringMetricbeatConfig(output)
		stepID := configrequest.StepRun
		if !watchMetrics || !any {
			stepID = configrequest.StepRemove
		}

		metricbeatStep := configrequest.Step{
			ID:      stepID,
			Version: version,
			ProgramSpec: program.Spec{
				Name:     metricsProcessName,
				Cmd:      metricsProcessName,
				Artifact: fmt.Sprintf("%s/%s", artifactPrefix, logsProcessName),
			},
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: mbConfig,
			},
		}

		steps = append(steps, metricbeatStep)
	}

	return steps
}

func (o *Operator) getMonitoringFilebeatConfig(output interface{}) (map[string]interface{}, bool) {
	inputs := []interface{}{
		map[string]interface{}{
			"type": "log",
			"json": map[string]interface{}{
				"keys_under_root": true,
				"overwrite_keys":  true,
				"message_key":     "message",
			},
			"paths": []string{
				filepath.Join(paths.Home(), "logs", "elastic-agent-json.log"),
			},
			"index": "logs-elastic.agent-default",
			"processors": []map[string]interface{}{
				{
					"add_fields": map[string]interface{}{
						"target": "data_stream",
						"fields": map[string]interface{}{
							"type":      "logs",
							"dataset":   "elastic.agent",
							"namespace": "default",
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "event",
						"fields": map[string]interface{}{
							"dataset": "elastic.agent",
						},
					},
				},
			},
		},
	}
	logPaths := o.getLogFilePaths()
	if len(logPaths) > 0 {
		for name, paths := range logPaths {
			inputs = append(inputs, map[string]interface{}{
				"type": "log",
				"json": map[string]interface{}{
					"keys_under_root": true,
					"overwrite_keys":  true,
					"message_key":     "message",
				},
				"paths": paths,
				"index": fmt.Sprintf("logs-elastic.agent.%s-default", name),
				"processors": []map[string]interface{}{
					{
						"add_fields": map[string]interface{}{
							"target": "data_stream",
							"fields": map[string]interface{}{
								"type":      "logs",
								"dataset":   fmt.Sprintf("elastic.agent.%s", name),
								"namespace": "default",
							},
						},
					},
					{
						"add_fields": map[string]interface{}{
							"target": "event",
							"fields": map[string]interface{}{
								"dataset": fmt.Sprintf("elastic.agent.%s", name),
							},
						},
					},
				},
			})
		}
	}
	result := map[string]interface{}{
		"filebeat": map[string]interface{}{
			"inputs": inputs,
		},
		"output": map[string]interface{}{
			"elasticsearch": output,
		},
	}

	o.logger.Debugf("monitoring configuration generated for filebeat: %v", result)

	return result, true
}

func (o *Operator) getMonitoringMetricbeatConfig(output interface{}) (map[string]interface{}, bool) {
	hosts := o.getMetricbeatEndpoints()
	if len(hosts) == 0 {
		return nil, false
	}
	var modules []interface{}
	for name, endpoints := range hosts {
		modules = append(modules, map[string]interface{}{
			"module":     "beat",
			"metricsets": []string{"stats", "state"},
			"period":     "10s",
			"hosts":      endpoints,
			"index":      fmt.Sprintf("metrics-elastic.agent.%s-default", name),
			"processors": []map[string]interface{}{
				{
					"add_fields": map[string]interface{}{
						"target": "data_stream",
						"fields": map[string]interface{}{
							"type":      "metrics",
							"dataset":   fmt.Sprintf("elastic.agent.%s", name),
							"namespace": "default",
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "event",
						"fields": map[string]interface{}{
							"dataset": fmt.Sprintf("elastic.agent.%s", name),
						},
					},
				},
			},
		})
	}
	result := map[string]interface{}{
		"metricbeat": map[string]interface{}{
			"modules": modules,
		},
		"output": map[string]interface{}{
			"elasticsearch": output,
		},
	}

	o.logger.Debugf("monitoring configuration generated for metricbeat: %v", result)

	return result, true
}

func (o *Operator) getLogFilePaths() map[string][]string {
	paths := map[string][]string{}

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		logPath := a.Monitor().LogPath(a.Name(), o.pipelineID)
		if logPath != "" {
			paths[a.Name()] = append(paths[a.Name()], logPath)
		}
	}

	return paths
}

func (o *Operator) getMetricbeatEndpoints() map[string][]string {
	endpoints := map[string][]string{}

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		metricEndpoint := a.Monitor().MetricsPathPrefixed(a.Name(), o.pipelineID)
		if metricEndpoint != "" {
			endpoints[a.Name()] = append(endpoints[a.Name()], metricEndpoint)
		}
	}

	return endpoints
}

func (o *Operator) markStopMonitoring(process string) {
	switch process {
	case logsProcessName:
		o.isMonitoring ^= isMonitoringLogsFlag
	case metricsProcessName:
		o.isMonitoring ^= isMonitoringMetricsFlag
	}
}

func (o *Operator) markStartMonitoring(process string) {
	switch process {
	case logsProcessName:
		o.isMonitoring |= isMonitoringLogsFlag
	case metricsProcessName:
		o.isMonitoring |= isMonitoringMetricsFlag
	}
}

func (o *Operator) isMonitoringLogs() bool {
	return (o.isMonitoring & isMonitoringLogsFlag) != 0
}

func (o *Operator) isMonitoringMetrics() bool {
	return (o.isMonitoring & isMonitoringMetricsFlag) != 0
}
