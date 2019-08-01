// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_cef

type config struct {
	Tag    string `config:"tag"`
	Field  string `config:"field"`
	Target string `config:"target_field"`
	ECS    bool   `config:"ecs"`
}

func defaultConfig() config {
	return config{
		Field:  "message",
		Target: "cef",
		ECS:    true,
	}
}
