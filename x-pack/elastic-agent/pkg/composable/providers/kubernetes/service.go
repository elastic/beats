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
	scope          string
	config         *Config
}

type serviceData struct {
	service    *kubernetes.Service
	mapping    map[string]interface{}
	processors []map[string]interface{}
}

// NewServiceWatcher creates a watcher that can discover and process service objects
func NewServiceWatcher(
	comm composable.DynamicProviderComm,
	cfg *Config,
	logger *logp.Logger,
	client k8s.Interface,
	scope string) (kubernetes.Watcher, error) {
	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Service{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}
	watcher.AddEventHandler(&service{logger, cfg.CleanupTimeout, comm, scope, cfg})

	return watcher, nil
}

func (s *service) emitRunning(service *kubernetes.Service) {
	data := generateServiceData(service, s.config)
	if data == nil {
		return
	}
	data.mapping["scope"] = s.scope

	// Emit the service
	s.comm.AddOrUpdate(string(service.GetUID()), ServicePriority, data.mapping, data.processors)
}

func (s *service) emitStopped(service *kubernetes.Service) {
	s.comm.Remove(string(service.GetUID()))
}

// OnAdd ensures processing of service objects that are newly created
func (s *service) OnAdd(obj interface{}) {
	s.logger.Debugf("Watcher Service add: %+v", obj)
	s.emitRunning(obj.(*kubernetes.Service))
}

// OnUpdate ensures processing of service objects that are updated
func (s *service) OnUpdate(obj interface{}) {
	service := obj.(*kubernetes.Service)
	// Once service is in terminated state, mark it for deletion
	if service.GetObjectMeta().GetDeletionTimestamp() != nil {
		s.logger.Debugf("Watcher Service update (terminating): %+v", obj)
		time.AfterFunc(s.cleanupTimeout, func() { s.emitStopped(service) })
	} else {
		s.logger.Debugf("Watcher Node update: %+v", obj)
		s.emitRunning(service)
	}
}

// OnDelete ensures processing of service objects that are deleted
func (s *service) OnDelete(obj interface{}) {
	s.logger.Debugf("Watcher Service delete: %+v", obj)
	service := obj.(*kubernetes.Service)
	time.AfterFunc(s.cleanupTimeout, func() { s.emitStopped(service) })
}

func generateServiceData(service *kubernetes.Service, cfg *Config) *serviceData {
	host := service.Spec.ClusterIP

	// If a service doesn't have an IP then dont monitor it
	if host == "" {
		return nil
	}

	//TODO: add metadata here too ie -> meta := s.metagen.Generate(service)

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range service.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	labels := common.MapStr{}
	for k, v := range service.GetObjectMeta().GetLabels() {
		// TODO: add dedoting option
		safemapstr.Put(labels, k, v)
	}

	mapping := map[string]interface{}{
		"service": map[string]interface{}{
			"uid":         string(service.GetUID()),
			"name":        service.GetName(),
			"labels":      labels,
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
	return &serviceData{
		service:    service,
		mapping:    mapping,
		processors: processors,
	}
}
