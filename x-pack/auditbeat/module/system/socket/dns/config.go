// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package dns

type config struct {
	// Enabled toggles the DNS monitoring feature.
	Enabled bool `config:"socket.dns.enabled"`
	// Type is the dns monitoring implementation used.
	Type string `config:"socket.dns.type"`
}

func defaultConfig() config {
	return config{
		Enabled: true,
		Type:    "af_packet",
	}
}
