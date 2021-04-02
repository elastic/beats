// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetesleaderelection

import (
	"context"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func init() {
	composable.Providers.AddDynamicProvider("kubernetes_leaderelection", DynamicProviderBuilder)
}

type dynamicProvider struct {
	logger               *logger.Logger
	config               *Config
	comm                 composable.DynamicProviderComm
	leaderElection       *leaderelection.LeaderElectionConfig
	cancelLeaderElection context.CancelFunc
}

// DynamicProviderBuilder builds the dynamic provider.
func DynamicProviderBuilder(logger *logger.Logger, c *config.Config) (composable.DynamicProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &dynamicProvider{logger, &cfg, nil, nil, nil}, nil
}

// Run runs the environment context provider.
func (p *dynamicProvider) Run(comm composable.DynamicProviderComm) error {
	client, err := kubernetes.GetKubernetesClient(p.config.KubeConfig)
	if err != nil {
		// info only; return nil (do nothing)
		p.logger.Debugf("Kubernetes leaderelection provider skipped, unable to connect: %s", err)
		return nil
	}
	if p.config.LeaderLease == "" {
		p.logger.Debugf("Kubernetes leaderelection provider skipped, unable to define leader lease")
		return nil
	}

	agentInfo, err := info.NewAgentInfo()
	if err != nil {
		return err
	}
	id := "elastic-agent-leader-" + agentInfo.AgentID()

	ns, err := kubernetes.InClusterNamespace()
	if err != nil {
		ns = "default"
	}
	lease := metav1.ObjectMeta{
		Name:      p.config.LeaderLease,
		Namespace: ns,
	}
	metaUID := lease.GetObjectMeta().GetUID()
	p.leaderElection = &leaderelection.LeaderElectionConfig{
		Lock: &resourcelock.LeaseLock{
			LeaseMeta: lease,
			Client:    client.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: id,
			},
		},
		ReleaseOnCancel: true,
		LeaseDuration:   15 * time.Second,
		RenewDeadline:   10 * time.Second,
		RetryPeriod:     2 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				p.logger.Debugf("leader election lock GAINED, id %v", id)
				p.startLeading(string(metaUID))
			},
			OnStoppedLeading: func() {
				p.logger.Debugf("leader election lock LOST, id %v", id)
				p.stopLeading(string(metaUID))
			},
		},
	}
	ctx, cancel := context.WithCancel(context.TODO())
	p.cancelLeaderElection = cancel
	p.comm = comm
	p.startLeaderElector(ctx)

	return nil
}

// startLeaderElector starts a Leader Elector in the background with the provided config
func (p *dynamicProvider) startLeaderElector(ctx context.Context) {
	le, err := leaderelection.NewLeaderElector(*p.leaderElection)
	if err != nil {
		p.logger.Errorf("error while creating Leader Elector: %v", err)
	}
	p.logger.Debugf("Starting Leader Elector")
	go le.Run(ctx)
}

func (p *dynamicProvider) startLeading(metaUID string) {
	mapping := map[string]interface{}{
		"leader": true,
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes_leaderelection",
			},
		},
	}

	p.comm.AddOrUpdate(metaUID, 0, mapping, processors)
}

func (p *dynamicProvider) stopLeading(metaUID string) {
	mapping := map[string]interface{}{
		"leader": false,
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes_leaderelection",
			},
		},
	}

	p.comm.AddOrUpdate(metaUID, 0, mapping, processors)
}

// Stop signals the stop channel to force the leader election loop routine to stop.
func (p *dynamicProvider) Stop() {
	if p.cancelLeaderElection != nil {
		p.cancelLeaderElection()
	}
}
