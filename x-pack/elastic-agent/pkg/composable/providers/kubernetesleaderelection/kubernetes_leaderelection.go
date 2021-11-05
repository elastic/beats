// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetesleaderelection

import (
	"context"
	"os"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"

	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/application/info"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	corecomp "github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/composable"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/core/logger"
)

func init() {
	composable.Providers.AddContextProvider("kubernetes_leaderelection", ContextProviderBuilder)
}

type contextProvider struct {
	logger               *logger.Logger
	config               *Config
	comm                 corecomp.ContextProviderComm
	leaderElection       *leaderelection.LeaderElectionConfig
	cancelLeaderElection context.CancelFunc
}

// ContextProviderBuilder builds the provider.
func ContextProviderBuilder(logger *logger.Logger, c *config.Config) (corecomp.ContextProvider, error) {
	var cfg Config
	if c == nil {
		c = config.New()
	}
	err := c.Unpack(&cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	return &contextProvider{logger, &cfg, nil, nil, nil}, nil
}

// Run runs the leaderelection provider.
func (p *contextProvider) Run(comm corecomp.ContextProviderComm) error {
	client, err := kubernetes.GetKubernetesClient(p.config.KubeConfig, p.config.KubeClientOptions)
	if err != nil {
		// info only; return nil (do nothing)
		p.logger.Debugf("Kubernetes leaderelection provider skipped, unable to connect: %s", err)
		return nil
	}

	agentInfo, err := info.NewAgentInfo(false)
	if err != nil {
		return err
	}
	var id string
	podName, found := os.LookupEnv("POD_NAME")
	if found {
		id = "elastic-agent-leader-" + podName
	} else {
		id = "elastic-agent-leader-" + agentInfo.AgentID()
	}

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
func (p *contextProvider) startLeaderElector(ctx context.Context) {
	le, err := leaderelection.NewLeaderElector(*p.leaderElection)
	if err != nil {
		p.logger.Errorf("error while creating Leader Elector: %v", err)
	}
	p.logger.Debugf("Starting Leader Elector")
	go le.Run(ctx)
}

func (p *contextProvider) startLeading(metaUID string) {
	mapping := map[string]interface{}{
		"leader": true,
	}

	err := p.comm.Set(mapping)
	if err != nil {
		p.logger.Errorf("Failed updating leaderelection status to leader TRUE: %s", err)
	}
}

func (p *contextProvider) stopLeading(metaUID string) {
	mapping := map[string]interface{}{
		"leader": false,
	}

	err := p.comm.Set(mapping)
	if err != nil {
		p.logger.Errorf("Failed updating leaderelection status to leader FALSE: %s", err)
	}
}

// Stop signals the stop channel to force the leader election loop routine to stop.
func (p *contextProvider) Stop() {
	if p.cancelLeaderElection != nil {
		p.cancelLeaderElection()
	}
}
