// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build windows

package state

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/tables"
	"github.com/elastic/beats/v7/x-pack/osquerybeat/ext/osquery-extension/pkg/amcache/testdata"
)

func TestGlobalStateConfig(t *testing.T) {
	testHivePath := testdata.GetTestHivePathOrFatal(t)

	globalInstance := GetGlobalState()
	if globalInstance == nil {
		t.Fatal("Expected globalInstance to be initialized")
	}

	if globalInstance.Config.HivePath != defaultHivePath {
		t.Errorf("Expected default hive path %s, got %s", defaultHivePath, globalInstance.Config.HivePath)
	}

	testInstance := &GlobalState{Config: &Config{HivePath: testHivePath}}
	if testInstance.Config.HivePath != testHivePath {
		t.Errorf("Expected hive path %s, got %s", testHivePath, testInstance.Config.HivePath)
	}

	if testInstance == globalInstance {
		t.Error("Expected testInstance and globalInstance to be different instances")
	}

	if globalInstance != GetGlobalState() {
		t.Error("Expected GetGlobalState to return the same globalInstance")
	}
}

func TestCachingBehavior(t *testing.T) {
	testHivePath := testdata.GetTestHivePathOrFatal(t)

	// Don't use the global instance for this test
	instance := &GlobalState{Config: &Config{HivePath: testHivePath, ExpirationDuration: defaultExpirationDuration}}

	// The Global state holds raw pointer to the cached entries as well as accessor functions
	// Accessing the raw pointers should not trigger an update, but is not available outside this package
	// Accessing via the accessor functions should trigger an update if expired
	// We will use both to validate the caching and expiration behavior
	accessors := []func(...string) []tables.Entry{
		instance.GetApplicationEntries,
		instance.GetApplicationFileEntries,
		instance.GetApplicationShortcutEntries,
		instance.GetDriverBinaryEntries,
		instance.GetDevicePnpEntries,
	}

	// Validate that lastUpdated is zero initially
	if !instance.LastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be zero initially, got %v", instance.LastUpdated)
	}

	// Calling any of the accessor functions should cause the cache to update
	_ = accessors[0]()
	if instance.LastUpdated.IsZero() {
		t.Errorf("Expected lastUpdated to be set after accessor call, got %v", instance.LastUpdated)
	}
	previousUpdate := instance.LastUpdated

	// Calling the accessor functions again should not cause an update since it has not expired
	// Additionally they should all return results
	for _, accessFunc := range accessors {
		if len(accessFunc()) == 0 {
			t.Errorf("Expected accessor function to return results, got 0")
		}
	}

	if !instance.LastUpdated.Equal(previousUpdate) {
		t.Errorf("Expected lastUpdated to remain the same since it has not expired, got %v", instance.LastUpdated)
	}

	// Simulate expiration by setting LastUpdated back in time
	expiredTime := instance.LastUpdated.Add(-instance.Config.ExpirationDuration * 2)
	instance.LastUpdated = expiredTime

	// Validate that lastUpdated is indeed in the past
	if !instance.LastUpdated.Before(previousUpdate) {
		t.Errorf("Expected lastUpdated to be before previousUpdate after manual expiration, got %v", instance.LastUpdated)
	}

	// Calling any of the accessor functions should cause the cache to update since it has expired
	_ = accessors[0]()
	if !instance.LastUpdated.After(expiredTime) {
		t.Errorf("Expected lastUpdated to be updated after accessor call since it has expired, got %v", instance.LastUpdated)
	}
}
