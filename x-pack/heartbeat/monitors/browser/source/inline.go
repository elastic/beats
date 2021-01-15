// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

type InlineSource struct {
	Script string `config:"script"`
	BaseSource
}

func (l *InlineSource) Fetch() (err error) {
	return nil
}

func (l *InlineSource) Workdir() string {
	return ""
}

func (l *InlineSource) Close() error {
	return nil
}
