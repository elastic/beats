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

//go:build !aix
// +build !aix

package kubernetes

import (
	"fmt"
	"time"

	"github.com/gofrs/uuid"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/autodiscover/builder"
	"github.com/elastic/beats/v7/libbeat/common/bus"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/logp"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/elastic-agent-libs/safemapstr"
)

type node struct {
	uuid    uuid.UUID
	config  *Config
	metagen metadata.MetaGen
	logger  *logp.Logger
	publish func([]bus.Event)
	watcher kubernetes.Watcher
}

// NewNodeEventer creates an eventer that can discover and process node objects
func NewNodeEventer(uuid uuid.UUID, cfg *config.C, client k8s.Interface, publish func(event []bus.Event)) (Eventer, error) {
	logger := logp.NewLogger("autodiscover.node")

	config := defaultConfig()
	err := cfg.Unpack(&config)
	if err != nil {
		return nil, err
	}

	// Ensure that node is set correctly whenever the scope is set to "node". Make sure that node is empty
	// when cluster scope is enforced.
	if config.Scope == "node" {
		nd := &kubernetes.DiscoverKubernetesNodeParams{
			ConfigHost:  config.Node,
			Client:      client,
			IsInCluster: kubernetes.IsInCluster(config.KubeConfig),
			HostUtils:   &kubernetes.DefaultDiscoveryUtils{},
		}
		config.Node, err = kubernetes.DiscoverKubernetesNode(logger, nd)
		if err != nil {
			return nil, fmt.Errorf("could not discover kubernetes node: %w", err)
		}
	} else {
		config.Node = ""
	}

	logger.Debugf("Initializing a new Kubernetes watcher using node: %v", config.Node)

	watcher, err := kubernetes.NewNamedWatcher("node", client, &kubernetes.Node{}, kubernetes.WatchOptions{
		SyncTimeout:  config.SyncPeriod,
		Node:         config.Node,
		IsUpdated:    isUpdated,
		HonorReSyncs: true,
	}, nil)

	if err != nil {
		return nil, fmt.Errorf("couldn't create watcher for %T due to error %+v", &kubernetes.Node{}, err)
	}

	p := &node{
		config:  config,
		uuid:    uuid,
		publish: publish,
		metagen: metadata.NewNodeMetadataGenerator(cfg, watcher.Store(), client),
		logger:  logger,
		watcher: watcher,
	}

	watcher.AddEventHandler(p)
	return p, nil
}

// OnAdd ensures processing of node objects that are newly created
func (n *node) OnAdd(obj interface{}) {
	n.logger.Debugf("Watcher Node add: %+v", obj)
	n.emit(obj.(*kubernetes.Node), "start")
}

// OnUpdate ensures processing of node objects that are updated
func (n *node) OnUpdate(obj interface{}) {
	node := obj.(*kubernetes.Node)
	if node.GetObjectMeta().GetDeletionTimestamp() != nil {
		n.logger.Debugf("Watcher Node update (terminating): %+v", obj)
		// Node is terminating, don't reload its configuration and ignore the event as long as node is Ready.
		if isNodeReady(node) {
			return
		}
		time.AfterFunc(n.config.CleanupTimeout, func() { n.emit(node, "stop") })
	} else {
		n.logger.Debugf("Watcher Node update: %+v", obj)
		n.emit(node, "stop")
		n.emit(node, "start")
	}
}

// OnDelete ensures processing of node objects that are deleted
func (n *node) OnDelete(obj interface{}) {
	n.logger.Debugf("Watcher Node delete: %+v", obj)
	time.AfterFunc(n.config.CleanupTimeout, func() { n.emit(obj.(*kubernetes.Node), "stop") })
}

// GenerateHints creates hints needed for hints builder
func (n *node) GenerateHints(event bus.Event) bus.Event {
	// Try to build a config with enabled builders. Send a provider agnostic payload.
	// Builders are Beat specific.
	e := bus.Event{}
	var annotations mapstr.M
	var kubeMeta mapstr.M
	rawMeta, ok := event["kubernetes"]
	if ok {
		kubeMeta = rawMeta.(mapstr.M)
		// The builder base config can configure any of the field values of kubernetes if need be.
		e["kubernetes"] = kubeMeta
		if rawAnn, ok := kubeMeta["annotations"]; ok {
			annotations = rawAnn.(mapstr.M)
		}
	}
	if host, ok := event["host"]; ok {
		e["host"] = host
	}
	if port, ok := event["port"]; ok {
		e["port"] = port
	}

	hints := builder.GenerateHints(annotations, "", n.config.Prefix)
	n.logger.Debugf("Generated hints %+v", hints)
	if len(hints) != 0 {
		e["hints"] = hints
	}

	n.logger.Debugf("Generated builder event %+v", e)

	return e
}

// Start starts the eventer
func (n *node) Start() error {
	return n.watcher.Start()
}

// Stop stops the eventer
func (n *node) Stop() {
	n.watcher.Stop()
}

func (n *node) emit(node *kubernetes.Node, flag string) {
	host := getAddress(node)

	// If a node doesn't have an IP then dont monitor it
	if host == "" && flag != "stop" {
		return
	}

	// If the node is not in ready state then dont monitor it unless its a stop event
	if !isNodeReady(node) && flag != "stop" {
		return
	}

	eventID := fmt.Sprint(node.GetObjectMeta().GetUID())
	meta := n.metagen.Generate(node)

	kubemetaMap, _ := meta.GetValue("kubernetes")
	kubemeta, _ := kubemetaMap.(mapstr.M)
	kubemeta = kubemeta.Clone()
	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := mapstr.M{}
	for k, v := range node.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}
	kubemeta["annotations"] = annotations
	event := bus.Event{
		"provider":   n.uuid,
		"id":         eventID,
		flag:         true,
		"host":       host,
		"kubernetes": kubemeta,
		"meta":       meta,
	}
	n.publish([]bus.Event{event})
}

func isUpdated(o, n interface{}) bool {
	old, _ := o.(*kubernetes.Node)
	new, _ := n.(*kubernetes.Node)

	// Consider as not update in case one of the two objects is not a Node
	if old == nil || new == nil {
		return true
	}

	// This is a resync. It is not an update
	if old.ResourceVersion == new.ResourceVersion {
		return false
	}

	// If the old object and new object are different
	oldCopy := old.DeepCopy()
	oldCopy.ResourceVersion = ""

	newCopy := new.DeepCopy()
	newCopy.ResourceVersion = ""

	// If the old object and new object are different in either meta or spec then there is a valid change
	if !equality.Semantic.DeepEqual(oldCopy.Spec, newCopy.Spec) || !equality.Semantic.DeepEqual(oldCopy.ObjectMeta, newCopy.ObjectMeta) {
		return true
	}

	// If there is a change in the node status then there is a valid change.
	if isNodeReady(old) != isNodeReady(new) {
		return true
	}
	return false
}

func getAddress(node *kubernetes.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP && address.Address != "" {
			return address.Address
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeInternalIP && address.Address != "" {
			return address.Address
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeHostName && address.Address != "" {
			return address.Address
		}
	}

	return ""
}

func isNodeReady(node *kubernetes.Node) bool {
	for _, c := range node.Status.Conditions {
		if c.Type == v1.NodeReady {
			return c.Status == v1.ConditionTrue
		}
	}
	return false
}
