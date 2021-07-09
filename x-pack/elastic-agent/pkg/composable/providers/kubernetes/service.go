// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"time"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
)

type service struct {
	logger         *logp.Logger
	cleanupTimeout time.Duration
	comm           composable.DynamicProviderComm
}

// NewServiceWatcher creates a watcher that can discover and process service objects
func NewServiceWatcher(comm composable.DynamicProviderComm, cfg *Config, logger *logp.Logger, client k8s.Interface) (kubernetes.Watcher, error) {
	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Service{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}
	watcher.AddEventHandler(&service{logger, cfg.CleanupTimeout, comm})

	return watcher, nil
}

func (s *service) emitRunning(service *kubernetes.Service) {
	host := service.Spec.ClusterIP

	// If a service doesn't have an IP then dont monitor it
	if host == "" {
		return
	}

	//TODO: add metadata here too ie -> meta := s.metagen.Generate(service)

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range service.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	mapping := map[string]interface{}{
		"node": map[string]interface{}{
			"uid":         string(service.GetUID()),
			"name":        service.GetName(),
			"labels":      service.GetLabels(),
			"annotations": annotations,
			"ip":          host,
		},
	}

	processors := []map[string]interface{}{
		{
			"add_fields": map[string]interface{}{
				"fields": mapping,
				"target": "kubernetes",
			},
		},
	}

	// Emit the node
	s.comm.AddOrUpdate(string(service.GetUID()), ServicePriority, mapping, processors)
}

func (n *service) emitStopped(service *kubernetes.Service) {
	n.comm.Remove(string(service.GetUID()))
}

// OnAdd ensures processing of service objects that are newly created
func (n *service) OnAdd(obj interface{}) {
	n.logger.Debugf("Watcher Service add: %+v", obj)
	n.emitRunning(obj.(*kubernetes.Service))
}

// OnUpdate ensures processing of service objects that are updated
func (n *service) OnUpdate(obj interface{}) {
	service := obj.(*kubernetes.Service)
	// Once service is in terminated state, mark it for deletion
	if service.GetObjectMeta().GetDeletionTimestamp() != nil {
		n.logger.Debugf("Watcher Service update (terminating): %+v", obj)
		time.AfterFunc(n.cleanupTimeout, func() { n.emitStopped(service) })
	} else {
		n.logger.Debugf("Watcher Node update: %+v", obj)
		n.emitStopped(service)
		n.emitRunning(service)
	}
}

// OnDelete ensures processing of service objects that are deleted
func (s *service) OnDelete(obj interface{}) {
	s.logger.Debugf("Watcher Service delete: %+v", obj)
	service := obj.(*kubernetes.Service)
	time.AfterFunc(s.cleanupTimeout, func() { s.emitStopped(service) })
}
