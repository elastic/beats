// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

//go:build linux

package pkg

import (
	"os"
	"strconv"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/auditbeat/module/system/user"
	"github.com/elastic/elastic-agent-libs/logp"
)

func TestRPMParallel(t *testing.T) {
	currentUID := os.Getuid()
	if currentUID != 0 {
		t.Skipf("can only run as root")
	}
	logp.DevelopmentSetup()

	count := 20
	waiter := sync.WaitGroup{}
	waiter.Add(count)

	useUID := getUser(t)

	t.Logf("Starting...")
	for i := 0; i < count; i++ {
		inner := i
		go func() {
			defer waiter.Done()
			testMs := MetricSet{
				log: logp.L(),
				config: config{
					PackageSuidDrop: &useUID,
				},
			}

			pkgList, err := testMs.getPackages()
			require.NoError(t, err)

			t.Logf("got %d packages from %d", len(pkgList), inner)
		}()

	}
	waiter.Wait()
}

func TestWithSuid(t *testing.T) {
	currentUID := os.Getuid()
	if currentUID != 0 {
		t.Skipf("can only run as root")
	}
	useUID := getUser(t)
	testMs := MetricSet{
		log: logp.L(),
		config: config{
			PackageSuidDrop: &useUID,
		},
	}

	packages, err := testMs.getPackages()
	require.NoError(t, err)

	require.NotZero(t, packages)
	t.Logf("got %d packages", len(packages))
}

func getUser(t *testing.T) int64 {
	// pick a user to drop to
	userList, err := user.GetUsers(false)
	require.NoError(t, err)

	var useUID int64
	for _, user := range userList {
		if user.UID != "0" {
			newUID, err := strconv.ParseInt(user.UID, 10, 64)
			require.NoError(t, err)
			useUID = newUID
			break
		}
	}
	return useUID
}
