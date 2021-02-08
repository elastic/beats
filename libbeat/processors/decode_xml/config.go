// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package decode_xml

type decodeXMLConfig struct {
	Field         string  `config:"field"`
	OverwriteKeys bool    `config:"overwrite_keys"`
	Target        *string `config:"target"`
	DocumentID    string  `config:"document_id"`
	ToLower       bool    `config:"to_lower"`
	IgnoreMissing bool    `config:"ignore_missing"`
	IgnoreFailure bool    `config:"ignore_failure"`
}

func defaultConfig() decodeXMLConfig {
	return decodeXMLConfig{
		Field:         "message",
		IgnoreMissing: false,
		Target:        new(string),
		OverwriteKeys: true,
		IgnoreFailure: false,
		ToLower:       true,
	}
}
