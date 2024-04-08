// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package kubernetes

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/coordination/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/elastic/elastic-agent-libs/logp"
)

const namespace = "default"
const leaseName = "metricbeat-lease-test"

// createLease creates a new lease resource
func createLease() *v1.Lease {
	lease := &v1.Lease{
		ObjectMeta: metav1.ObjectMeta{
			Name:      leaseName,
			Namespace: namespace,
		},
	}
	return lease
}

// applyLease applies the lease
func applyLease(client kubernetes.Interface, lease *v1.Lease, firstTime bool) error {
	var err error
	if firstTime {
		_, err = client.CoordinationV1().Leases(namespace).Create(context.Background(), lease, metav1.CreateOptions{})
		return err
	}
	_, err = client.CoordinationV1().Leases(namespace).Update(context.Background(), lease, metav1.UpdateOptions{})
	return err
}

// TestLeaseConfigurableFields tests if the leader election is using the fields given in the configuration
func TestLeaseConfigurableFields(t *testing.T) {
	client := k8sfake.NewSimpleClientset()

	uuid, err := uuid.NewV4()
	require.NoError(t, err)

	startLeadingFunc := func(uuid string, eventID string) {}
	stopLeadingFunc := func(uuid string, eventID string) {}
	logger := logp.NewLogger("kubernetes-test")

	// the number of leader election managers corresponds to the number of nodes in a cluster with metricbeat
	var leaseDuration time.Duration
	var retryPeriod time.Duration
	var renewDeadline time.Duration

	cfg := Config{
		Node:          "node-1",
		LeaderLease:   leaseName,
		LeaseDuration: 30 * time.Second,
		RenewDeadline: 25 * time.Second,
		RetryPeriod:   10 * time.Second,
	}

	le, err := NewLeaderElectionManager(uuid, &cfg, client, startLeadingFunc, stopLeadingFunc, logger)
	require.NoError(t, err)

	leaseDuration = le.(*leaderElectionManager).leaderElection.LeaseDuration
	retryPeriod = le.(*leaderElectionManager).leaderElection.RetryPeriod
	renewDeadline = le.(*leaderElectionManager).leaderElection.RenewDeadline

	require.Equalf(t, cfg.LeaseDuration, leaseDuration, "lease duration should be the same as the one provided in the configuration.")
	require.Equalf(t, cfg.RetryPeriod, retryPeriod, "retry period should be the same as the one provided in the configuration.")
	require.Equalf(t, cfg.RenewDeadline, renewDeadline, "renew deadline should be the same as the one provided in the configuration.")
}

// TestNewLeaderElectionManager will test the leader elector.
// This tests aims to check two things:
// 1. The event id used to stop the leader is the same as the event id that was used to start it.
// 2. The leader elector runs again after it stops. The only way for it to stop, is to stop the event manager as well - this
// could be caused by the provider stopping, for example.
func TestNewLeaderElectionManager(t *testing.T) {
	client := k8sfake.NewSimpleClientset()

	lease := createLease()
	// create the lease that leader election will be using
	err := applyLease(client, lease, true)
	require.NoError(t, err)

	uuid, err := uuid.NewV4()
	require.NoError(t, err)

	waitForNewLeader := make(chan string)
	waitForLosingLeader := make(chan string)

	startLeadingFunc := func(uuid string, eventID string) {
		waitForNewLeader <- eventID
	}
	stopLeadingFunc := func(uuid string, eventID string) {
		waitForLosingLeader <- eventID
	}
	logger := logp.NewLogger("kubernetes-test")

	cfg := Config{
		LeaderLease:   leaseName,
		RenewDeadline: 30 * time.Millisecond,
		RetryPeriod:   10 * time.Millisecond,
		LeaseDuration: 1 * time.Second,
	}

	// the number of leader election managers corresponds to the number of nodes in a cluster with metricbeat
	numberNodes := 2
	les := make([]*EventManager, numberNodes)
	nodeNames := make([]string, numberNodes)
	var leaseDuration time.Duration
	var retryPeriod time.Duration
	for i := 0; i < numberNodes; i++ {
		nodeName := "node-" + fmt.Sprint(i)
		nodeNames[i] = nodeName
		cfg.Node = nodeName

		le, err := NewLeaderElectionManager(uuid, &cfg, client, startLeadingFunc, stopLeadingFunc, logger)
		require.NoError(t, err)

		leaseDuration = le.(*leaderElectionManager).leaderElection.LeaseDuration
		retryPeriod = le.(*leaderElectionManager).leaderElection.RetryPeriod

		les[i] = &le
	}

	for _, le := range les {
		(*le).Start()
	}

	// It is possible that startLeading is triggered more than one time before stopLeading is called.
	// Example of a situation like this:
	// 1. node-1 is elected as leader, and startLeading already executed.
	// 2. node-1 loses the leader lock, and stopLeading is starting to get executed.
	// 3. node-2 calls startLeading before the execution of two ends.
	// This situation was observed in this unit test. So to check we are receiving correct event ids and without
	// knowing the right order, we have to save the ones we received from startLeading in a map.
	expectedLoosingEventIds := make(map[string]bool)

	finished := make(chan int)
	endedRequests := make(chan int)

	checkLoosingLeaders := func(eventId string) {
		_, exists := expectedLoosingEventIds[eventId]
		if exists {
			t.Fatalf("The new leader produced the same event id as the previous one.")
		}
		expectedLoosingEventIds[eventId] = true

		// wait for loosing leader
		loosingEventId := <-waitForLosingLeader
		_, exists = expectedLoosingEventIds[loosingEventId]
		if !exists {
			t.Fatalf("The loosing leader used an unexpected event id %s.", eventId)
		}
	}

	go func() {
		// wait for first leader
		newEventId := <-waitForNewLeader
		expectedLoosingEventIds[newEventId] = true

		// every time there is a new leader, we should check the event id emitted from the stopLeading
	waitForRenewals:
		for {
			select {
			case eventId := <-waitForNewLeader:
				checkLoosingLeaders(eventId)
			case <-endedRequests:
				// once we receive something in this channel, we know the lease is no longer being modified,
				// so we can finish this goroutine
				finished <- 1
				break waitForRenewals
			}
		}
	}()

	renewals := 5
	// cause lease renewals
	for i := 0; i < renewals; i++ {
		// Force the lease to be applied again, so a new leader is elected.
		newHolder := "does-not-matter-" + fmt.Sprint(i)
		lease.Spec.HolderIdentity = &newHolder
		err = applyLease(client, lease, false)
		require.NoError(t, err)

		// wait some time to ensure lease renewal
		<-time.After((retryPeriod + leaseDuration) * 2)
	}
	endedRequests <- 1

	<-finished

	// Wait for some to ensure we are not having lease fail renewal, and there is no new leader.
	<-time.After((retryPeriod + leaseDuration) * 2)

	// waitForNewLeader channel should be empty, because we removed it just before ending the for cycle.
	require.Equalf(t, 0, len(waitForNewLeader), "waitForNewLeader channel should be empty.")

	// waitForLosingLeader channel should be empty, because the last leader did not lose the lease lock yet.
	require.Equalf(t, 0, len(waitForLosingLeader), "waitForLosingLeader channel should be empty.")

	for _, le := range les {
		(*le).Stop()
	}

	// When the context gets cancelled, stopLeading is always called.
	// Let's check that the leaders electors are correctly stopping.
	for i := 0; i < numberNodes; i++ {
		<-waitForLosingLeader
	}
}
