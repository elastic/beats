// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows
// +build windows

package test

import (
	"github.com/StackExchange/wmi"
	"github.com/pkg/errors"
)

// Service struct used to map Win32_Service
type Service struct {
	Name    string
	Started bool
	State   string
}

func GetConfig(metricset string) map[string]interface{} {
	return map[string]interface{}{
		"module":     "iis",
		"metricsets": []string{metricset},
	}
}

// EnsureIISIsRunning func will check if IIS is installed and running
func EnsureIISIsRunning() error {
	var ser []Service
	err := wmi.Query("Select * from Win32_Service where Name = 'w3svc'", &ser)
	if err != nil {
		return err
	}
	if len(ser) == 0 {
		return errors.New("IIS is not not installed")
	}
	if ser[0].State != "Running" {
		return errors.Errorf("IIS is installed but status is %s", ser[0].State)
	}
	return nil
}
