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
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/libbeat/autodiscover/builder"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/bus"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
	"github.com/elastic/beats/libbeat/logp"
)

type service struct {
	uuid    uuid.UUID
	config  *Config
	metagen kubernetes.MetaGenerator
	logger  *logp.Logger
	publish func(bus.Event)
	watcher kubernetes.Watcher
}

// NewServiceEventer creates an eventer that can discover and process service objects
func NewServiceEventer(uuid uuid.UUID, cfg *common.Config, client k8s.Interface, publish func(event bus.Event)) (Eventer, error) {
	metagen, err := kubernetes.NewMetaGenerator(cfg)
	if err != nil {
		return nil, err
	}

	logger := logp.NewLogger("autodiscover.service")

	config := defaultConfig()
	err = cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	watcher, err := kubernetes.NewWatcher(client, &kubernetes.Service{}, kubernetes.WatchOptions{
		SyncTimeout: config.SyncPeriod,
	})

	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Service{}, err)
	}

	p := &service{
		config:  config,
		uuid:    uuid,
		publish: publish,
		metagen: metagen,
		logger:  logger,
		watcher: watcher,
	}

	watcher.AddEventHandler(p)
	return p, nil
}

// OnAdd ensures processing of service objects that are newly created
func (s *service) OnAdd(obj interface{}) {
	s.logger.Debugf("Watcher Node add: %+v", obj)
	s.emit(obj.(*kubernetes.Service), "start")
}

// OnUpdate ensures processing of service objects that are updated
func (s *service) OnUpdate(obj interface{}) {
	svc := obj.(*kubernetes.Service)
	// Once service is in terminated state, mark it for deletion
	if svc.GetObjectMeta().GetDeletionTimestamp() != nil {
		time.AfterFunc(s.config.CleanupTimeout, func() { s.emit(svc, "stop") })
	} else {
		s.logger.Debugf("Watcher Node update: %+v", obj)
		s.emit(svc, "stop")
		s.emit(svc, "start")
	}
}

// OnDelete ensures processing of service objects that are deleted
func (s *service) OnDelete(obj interface{}) {
	s.logger.Debugf("Watcher Node delete: %+v", obj)
	time.AfterFunc(s.config.CleanupTimeout, func() { s.emit(obj.(*kubernetes.Service), "stop") })
}

// GenerateHints creates hints needed for hints builder
func (s *service) GenerateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var annotations common.MapStr
	var kubeMeta common.MapStr
	rawMeta, ok := event["kubernetes"]
	if ok {
		kubeMeta = rawMeta.(common.MapStr)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["kubernetes"] = kubeMeta
		if rawAnn, ok := kubeMeta["annotations"]; ok {
			annotations = rawAnn.(common.MapStr)
		}
	}
	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}

	hints := builder.GenerateHints(annotations, "", s.config.Prefix)
	s.logger.Debugf("Generated hints %+v", hints)
	if len(hints) != 0 {
		e["hints"] = hints
	}

	s.logger.Debugf("Generated builder event %+v", e)

	return e
}

// Start starts the eventer
func (s *service) Start() error {
	return s.watcher.Start()
}

// Stop stops the eventer
func (s *service) Stop() {
	s.watcher.Stop()
}

func (s *service) emit(svc *kubernetes.Service, flag string) {
	host := svc.Spec.ClusterIP

	// If a service doesn't have an IP then dont monitor it
	if host == "" && flag != "stop" {
		return
	}

	eventID := fmt.Sprint(svc.GetObjectMeta().GetUID())
	meta := s.metagen.ResourceMetadata(svc)

	// TODO: Refactor metagen to make sure that this is seamless
	meta.Put("service.name", svc.Name)
	meta.Put("service.uid", string(svc.GetObjectMeta().GetUID()))

	kubemeta := meta.Clone()
	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range svc.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	kubemeta["annotations"] = annotations

	for _, port := range svc.Spec.Ports {
		event := bus.Event{
			"provider":   s.uuid,
			"id":         eventID,
			flag:         true,
			"host":       host,
			"port":       int(port.Port),
			"kubernetes": kubemeta,
			"meta": common.MapStr{
				"kubernetes": meta,
			},
		}
		s.publish(event)
	}

}
