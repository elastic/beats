// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package jumplists

import (
	"fmt"
	"os"
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/logger"
)

func TestFindCustomDestinationFiles(t *testing.T) {
	log := logger.New(os.Stdout, true)
	files, err := FindJumplistFiles(JumpListTypeCustom, log)
	fmt.Println("Files:")
	for _, file := range files {
		fmt.Printf("    %s\n", file)
	}
	if err != nil {
		t.Errorf("FindCustomFiles() returned error: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("No custom files found")
	}
	for _, file := range files {
		customJumpList, err := NewCustomJumpList(file, log)
		if err != nil {
			t.Errorf("NewAutomaticJumpList() returned error: %v", err)
		}
		fmt.Printf("Path: %s\n", customJumpList.Path())
		fmt.Printf("AppId: %s\n", customJumpList.AppId())
		fmt.Printf("Type: %s\n", customJumpList.Type())
	}
}

func TestFindAutomaticFiles(t *testing.T) {
	log := logger.New(os.Stdout, true)
	files, err := FindJumplistFiles(JumpListTypeAutomatic, log)
	fmt.Println("Files:")
	for _, file := range files {
		fmt.Printf("    %s\n", file)
	}
	if err != nil {
		t.Errorf("FindAutomaticDestinationFiles() returned error: %v", err)
	}
	if len(files) == 0 {
		t.Errorf("No automatic destination files found")
	}
	for _, file := range files {
		automaticJumpList, err := NewAutomaticJumpList(file, log)
		if err != nil {
			t.Errorf("NewAutomaticJumpList() returned error: %v", err)
		}
		fmt.Printf("Path: %s\n", automaticJumpList.Path())
		fmt.Printf("AppId: %s\n", automaticJumpList.AppId())
		fmt.Printf("Type: %s\n", automaticJumpList.Type())
	}
}