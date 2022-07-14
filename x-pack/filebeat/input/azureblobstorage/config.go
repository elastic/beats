// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build !aix
// +build !aix

package azureblobstorage

import (
	"fmt"
	"time"
)

type config struct {
	AccountName string      `config:"account_name"`
	AccountKey  string      `config:"account_key"`
	Containers  []container `config:"containers"`
}

type container struct {
	Name         string        `config:"name" validate:"required"`
	MaxWorkers   int           `config:"max_workers"`
	Poll         bool          `config:"poll"`
	PollInterval time.Duration `config:"poll_interval"`
}

func (c config) Validate() error {
	for _, v := range c.Containers {
		if v.MaxWorkers > 10000 {
			return fmt.Errorf("batch size should be less than 10000")
		}
	}
	return nil
}

func defaultConfig() config {
	return config{
		AccountName: "beatsblobstorage1",
		AccountKey:  "7pfLm1betGiRyyABEM/RFrLYlafLZHbLtGhB52LkWVeBxE7la9mIvk6YYAbQKYE/f0GdhiaOZeV8+AStsAdr/Q==",
		Containers: []container{
			{Name: "beatscontainer", MaxWorkers: 1, Poll: true, PollInterval: time.Duration(time.Second * 5)},
			{Name: "blobcontainer", MaxWorkers: 3, Poll: true, PollInterval: time.Duration(time.Second * 5)},
		},
	}
}
