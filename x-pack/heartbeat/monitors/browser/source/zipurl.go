// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package source

type ZipURLSource struct {
	Url     string            `config:"url"`
	Headers map[string]string `config:"headers"`
	PollingSource
}

func (z *ZipURLSource) Fetch() error {
	panic("implement me")
}

func (z *ZipURLSource) Workdir() string {
	panic("implement me")
}
