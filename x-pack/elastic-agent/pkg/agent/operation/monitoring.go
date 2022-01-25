// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package operation

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/paths"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/configrequest"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/program"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/app"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/monitoring/beats"
)

const (
	monitoringName     = "FLEET_MONITORING"
	outputKey          = "output"
	logsProcessName    = "filebeat"
	metricsProcessName = "metricbeat"
	artifactPrefix     = "beats"
	agentName          = "elastic-agent"
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
	for _, step := range o.generateMonitoringSteps(s.Version, "", nil) {
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

	if len(outputMap) == 0 {
		o.logger.Errorf("operator.getMonitoringSteps: monitoring is missing an output configuration for sidecar of type: %s", step.ProgramSpec.Cmd)
		return nil
	}

	// Guards against parser issues upstream, this should not be possible but
	// since we are folding all the child options as a map we should make sure we have
	//a unique output.
	if len(outputMap) > 1 {
		o.logger.Errorf("operator.getMonitoringSteps: monitoring has too many outputs configuration for sidecar of type: %s", step.ProgramSpec.Cmd)
		return nil
	}

	// Aggregate output configuration independently of the received output key.
	output := make(map[string]interface{})

	for _, v := range outputMap {
		child, ok := v.(map[string]interface{})
		if !ok {
			o.logger.Error("operator.getMonitoringSteps: monitoring config is not a map")
			return nil
		}
		for c, j := range child {
			output[c] = j
		}
	}

	t, ok := output["type"]
	if !ok {
		o.logger.Errorf("operator.getMonitoringSteps: unknown monitoring output for sidecar of type: %s", step.ProgramSpec.Cmd)
		return nil
	}

	outputType, ok := t.(string)
	if !ok {
		o.logger.Errorf("operator.getMonitoringSteps: unexpected monitoring output type: %+v for sidecar of type: %s", t, step.ProgramSpec.Cmd)
		return nil
	}

	return o.generateMonitoringSteps(step.Version, outputType, output)
}

func (o *Operator) generateMonitoringSteps(version, outputType string, output interface{}) []configrequest.Step {
	var steps []configrequest.Step
	watchLogs := o.monitor.WatchLogs()
	watchMetrics := o.monitor.WatchMetrics()
	monitoringNamespace := o.monitor.MonitoringNamespace()

	// generate only when monitoring is running (for config refresh) or
	// state changes (turning on/off)
	if watchLogs != o.isMonitoringLogs() || watchLogs {
		fbConfig, any := o.getMonitoringFilebeatConfig(outputType, output, monitoringNamespace)
		stepID := configrequest.StepRun
		if !watchLogs || !any {
			stepID = configrequest.StepRemove
		}
		filebeatStep := configrequest.Step{
			ID:          stepID,
			Version:     version,
			ProgramSpec: loadSpecFromSupported(logsProcessName),
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: fbConfig,
			},
		}

		steps = append(steps, filebeatStep)
	}
	if watchMetrics != o.isMonitoringMetrics() || watchMetrics {
		mbConfig, any := o.getMonitoringMetricbeatConfig(outputType, output, monitoringNamespace)
		stepID := configrequest.StepRun
		if !watchMetrics || !any {
			stepID = configrequest.StepRemove
		}

		metricbeatStep := configrequest.Step{
			ID:          stepID,
			Version:     version,
			ProgramSpec: loadSpecFromSupported(metricsProcessName),
			Meta: map[string]interface{}{
				configrequest.MetaConfigKey: mbConfig,
			},
		}

		steps = append(steps, metricbeatStep)
	}

	return steps
}

func loadSpecFromSupported(processName string) program.Spec {
	if loadedSpec, found := program.SupportedMap[strings.ToLower(processName)]; found {
		return loadedSpec
	}

	return program.Spec{
		Name:     processName,
		Cmd:      processName,
		Artifact: fmt.Sprintf("%s/%s", artifactPrefix, processName),
	}
}

func (o *Operator) getMonitoringFilebeatConfig(outputType string, output interface{}, monitoringNamespace string) (map[string]interface{}, bool) {
	inputs := []interface{}{
		map[string]interface{}{
			"type": "filestream",
			"close": map[string]interface{}{
				"on_state_change": map[string]interface{}{
					"inactive": "5m",
				},
			},
			"parsers": []map[string]interface{}{
				{
					"ndjson": map[string]interface{}{
						"overwrite_keys": true,
						"message_key":    "message",
					},
				},
			},
			"paths": []string{
				filepath.Join(paths.Home(), "logs", "elastic-agent-*.ndjson"),
				filepath.Join(paths.Home(), "logs", "elastic-agent-watcher-*.ndjson"),
			},
			"index": fmt.Sprintf("logs-elastic_agent-%s", monitoringNamespace),
			"processors": []map[string]interface{}{
				{
					"add_fields": map[string]interface{}{
						"target": "data_stream",
						"fields": map[string]interface{}{
							"type":      "logs",
							"dataset":   "elastic_agent",
							"namespace": monitoringNamespace,
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "event",
						"fields": map[string]interface{}{
							"dataset": "elastic_agent",
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "elastic_agent",
						"fields": map[string]interface{}{
							"id":       o.agentInfo.AgentID(),
							"version":  o.agentInfo.Version(),
							"snapshot": o.agentInfo.Snapshot(),
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "agent",
						"fields": map[string]interface{}{
							"id": o.agentInfo.AgentID(),
						},
					},
				},
				{
					"drop_fields": map[string]interface{}{
						"fields": []string{
							"ecs.version", //coming from logger, already added by libbeat
						},
						"ignore_missing": true,
					},
				},
			},
		},
	}
	logPaths := o.getLogFilePaths()
	if len(logPaths) > 0 {
		for name, paths := range logPaths {
			inputs = append(inputs, map[string]interface{}{
				"type": "filestream",
				"close": map[string]interface{}{
					"on_state_change": map[string]interface{}{
						"inactive": "5m",
					},
				},
				"parsers": []map[string]interface{}{
					{
						"ndjson": map[string]interface{}{
							"overwrite_keys": true,
							"message_key":    "message",
						},
					},
				},
				"paths": paths,
				"index": fmt.Sprintf("logs-elastic_agent.%s-%s", name, monitoringNamespace),
				"processors": []map[string]interface{}{
					{
						"add_fields": map[string]interface{}{
							"target": "data_stream",
							"fields": map[string]interface{}{
								"type":      "logs",
								"dataset":   fmt.Sprintf("elastic_agent.%s", name),
								"namespace": monitoringNamespace,
							},
						},
					},
					{
						"add_fields": map[string]interface{}{
							"target": "event",
							"fields": map[string]interface{}{
								"dataset": fmt.Sprintf("elastic_agent.%s", name),
							},
						},
					},
					{
						"add_fields": map[string]interface{}{
							"target": "elastic_agent",
							"fields": map[string]interface{}{
								"id":       o.agentInfo.AgentID(),
								"version":  o.agentInfo.Version(),
								"snapshot": o.agentInfo.Snapshot(),
							},
						},
					},
					{
						"add_fields": map[string]interface{}{
							"target": "agent",
							"fields": map[string]interface{}{
								"id": o.agentInfo.AgentID(),
							},
						},
					},
					{
						"drop_fields": map[string]interface{}{
							"fields": []string{
								"ecs.version", //coming from logger, already added by libbeat
							},
							"ignore_missing": true,
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
			outputType: output,
		},
	}

	return result, true
}

func (o *Operator) getMonitoringMetricbeatConfig(outputType string, output interface{}, monitoringNamespace string) (map[string]interface{}, bool) {
	hosts := o.getMetricbeatEndpoints()
	if len(hosts) == 0 {
		return nil, false
	}
	var modules []interface{}
	fixedAgentName := strings.ReplaceAll(agentName, "-", "_")

	for name, endpoints := range hosts {
		modules = append(modules, map[string]interface{}{
			"module":     "beat",
			"metricsets": []string{"stats", "state"},
			"period":     "10s",
			"hosts":      endpoints,
			"index":      fmt.Sprintf("metrics-elastic_agent.%s-%s", name, monitoringNamespace),
			"processors": []map[string]interface{}{
				{
					"add_fields": map[string]interface{}{
						"target": "data_stream",
						"fields": map[string]interface{}{
							"type":      "metrics",
							"dataset":   fmt.Sprintf("elastic_agent.%s", name),
							"namespace": monitoringNamespace,
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "event",
						"fields": map[string]interface{}{
							"dataset": fmt.Sprintf("elastic_agent.%s", name),
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "elastic_agent",
						"fields": map[string]interface{}{
							"id":       o.agentInfo.AgentID(),
							"version":  o.agentInfo.Version(),
							"snapshot": o.agentInfo.Snapshot(),
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "agent",
						"fields": map[string]interface{}{
							"id": o.agentInfo.AgentID(),
						},
					},
				},
			},
		}, map[string]interface{}{
			"module":     "http",
			"metricsets": []string{"json"},
			"namespace":  "agent",
			"period":     "10s",
			"path":       "/stats",
			"hosts":      endpoints,
			"index":      fmt.Sprintf("metrics-elastic_agent.%s-%s", fixedAgentName, monitoringNamespace),
			"processors": []map[string]interface{}{
				{
					"add_fields": map[string]interface{}{
						"target": "data_stream",
						"fields": map[string]interface{}{
							"type":      "metrics",
							"dataset":   fmt.Sprintf("elastic_agent.%s", fixedAgentName),
							"namespace": monitoringNamespace,
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "event",
						"fields": map[string]interface{}{
							"dataset": fmt.Sprintf("elastic_agent.%s", fixedAgentName),
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "elastic_agent",
						"fields": map[string]interface{}{
							"id":       o.agentInfo.AgentID(),
							"version":  o.agentInfo.Version(),
							"snapshot": o.agentInfo.Snapshot(),
							"process":  name,
						},
					},
				},
				{
					"add_fields": map[string]interface{}{
						"target": "agent",
						"fields": map[string]interface{}{
							"id": o.agentInfo.AgentID(),
						},
					},
				},
				{
					"copy_fields": map[string]interface{}{
						"fields":         normalizeHTTPCopyRules(name),
						"ignore_missing": true,
					},
				},
				{
					"drop_fields": map[string]interface{}{
						"fields": []string{
							"http",
						},
						"ignore_missing": true,
					},
				},
			},
		})
	}

	modules = append(modules, map[string]interface{}{
		"module":     "http",
		"metricsets": []string{"json"},
		"namespace":  "agent",
		"period":     "10s",
		"path":       "/stats",
		"hosts":      []string{beats.AgentPrefixedMonitoringEndpoint(o.config.DownloadConfig.OS(), o.config.MonitoringConfig.HTTP)},
		"index":      fmt.Sprintf("metrics-elastic_agent.%s-%s", fixedAgentName, monitoringNamespace),
		"processors": []map[string]interface{}{
			{
				"add_fields": map[string]interface{}{
					"target": "data_stream",
					"fields": map[string]interface{}{
						"type":      "metrics",
						"dataset":   fmt.Sprintf("elastic_agent.%s", fixedAgentName),
						"namespace": monitoringNamespace,
					},
				},
			},
			{
				"add_fields": map[string]interface{}{
					"target": "event",
					"fields": map[string]interface{}{
						"dataset": fmt.Sprintf("elastic_agent.%s", fixedAgentName),
					},
				},
			},
			{
				"add_fields": map[string]interface{}{
					"target": "elastic_agent",
					"fields": map[string]interface{}{
						"id":       o.agentInfo.AgentID(),
						"version":  o.agentInfo.Version(),
						"snapshot": o.agentInfo.Snapshot(),
						"process":  "elastic-agent",
					},
				},
			},
			{
				"add_fields": map[string]interface{}{
					"target": "agent",
					"fields": map[string]interface{}{
						"id": o.agentInfo.AgentID(),
					},
				},
			},
			{
				"copy_fields": map[string]interface{}{
					"fields":         normalizeHTTPCopyRules(fixedAgentName),
					"ignore_missing": true,
				},
			},
			{
				"drop_fields": map[string]interface{}{
					"fields": []string{
						"http",
					},
					"ignore_missing": true,
				},
			},
		},
	})

	result := map[string]interface{}{
		"metricbeat": map[string]interface{}{
			"modules": modules,
		},
		"output": map[string]interface{}{
			outputType: output,
		},
	}

	return result, true
}

func (o *Operator) getLogFilePaths() map[string][]string {
	paths := map[string][]string{}

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		logPath := a.Monitor().LogPath(a.Spec(), o.pipelineID)
		if logPath != "" {
			paths[strings.ReplaceAll(a.Name(), "-", "_")] = []string{
				logPath,
				fmt.Sprintf("%s*", logPath),
			}
		}
	}

	return paths
}

func (o *Operator) getMetricbeatEndpoints() map[string][]string {
	endpoints := map[string][]string{}

	o.appsLock.Lock()
	defer o.appsLock.Unlock()

	for _, a := range o.apps {
		metricEndpoint := a.Monitor().MetricsPathPrefixed(a.Spec(), o.pipelineID)
		if metricEndpoint != "" {
			safeName := strings.ReplaceAll(a.Name(), "-", "_")
			// prevent duplicates
			var found bool
			for _, ep := range endpoints[safeName] {
				if ep == metricEndpoint {
					found = true
					break
				}
			}

			if !found {
				endpoints[safeName] = append(endpoints[safeName], metricEndpoint)
			}
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

func normalizeHTTPCopyRules(name string) []map[string]interface{} {
	fromToMap := []map[string]interface{}{
		// I should be able to see the CPU Usage on the running machine. Am using too much CPU?
		{
			"from": "http.agent.beat.cpu",
			"to":   "system.process.cpu",
		},
		// I should be able to see the Memory usage of Elastic Agent. Is the Elastic Agent using too much memory?
		{
			"from": "http.agent.beat.memstats.memory_sys",
			"to":   "system.process.memory.size",
		},
		// I should be able to see the system memory. Am I running out of memory?
		// TODO: with APM agent: total and free

		// I should be able to see Disk usage on the running machine. Am I running out of disk space?
		// TODO: with APM agent

		// I should be able to see fd usage. Am I keep too many files open?
		{
			"from": "http.agent.beat.handles",
			"to":   "system.process.fd",
		},
		// Cgroup reporting
		{
			"from": "http.agent.beat.cgroup",
			"to":   "system.process.cgroup",
		},
	}

	spec, found := program.SupportedMap[name]
	if !found {
		return fromToMap
	}

	for _, exportedMetric := range spec.ExprtedMetrics {
		fromToMap = append(fromToMap, map[string]interface{}{
			"from": fmt.Sprintf("http.agent.%s", exportedMetric),
			"to":   exportedMetric,
		})
	}

	return fromToMap
}
