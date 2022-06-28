// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import "fmt"

type config struct {
	AccountName string      `config:"account_name"`
	AccountKey  string      `config:"account_key"`
	Containers  []container `config:"containers"`
}

type container struct {
	Name           string `config:"name" validate:"required"`
	BatchSize      int32  `config:"batch_size"`
	Poll           bool   `config:"poll"`
	PollIntervalMs int32  `config:"poll_interval_ms"`
}

func (c config) Validate() error {
	for _, v := range c.Containers {
		if v.BatchSize > 10000 {
			return fmt.Errorf("batch size should be less than 10000")
		}
	}
	return nil
}

func defaultConfig() config {
	return config{
		AccountName: "beatsblobstorage",
		AccountKey:  "61A0frq/mFUSw6BGivRB8jhOiElUwGcMlI5lCbXruJokvYIWUcwvpp9ln6v7MPBzwsfvprCEt2qA+AStH+iVXw==",
		Containers: []container{
			{Name: "beatscontainer", BatchSize: 10, Poll: true, PollIntervalMs: 5000},
			{Name: "blobcontainer", BatchSize: 10, Poll: true, PollIntervalMs: 5000},
		},
	}
}
