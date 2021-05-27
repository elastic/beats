// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package configuration

// IsStandalone decides based on missing of fleet.enabled: true or fleet.{access_token,kibana} will place Elastic Agent into standalone mode.
func IsStandalone(cfg *FleetAgentConfig) bool {
	return cfg == nil || !cfg.Enabled
}

// IsFleetServerBootstrap decides if Elastic Agent is started in bootstrap mode.
func IsFleetServerBootstrap(cfg *FleetAgentConfig) bool {
	return cfg != nil && cfg.Server != nil && cfg.Server.Bootstrap
}
