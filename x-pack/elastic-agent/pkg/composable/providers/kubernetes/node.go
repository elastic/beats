// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package kubernetes

import (
	"time"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8s "k8s.io/client-go/kubernetes"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes"
	"github.com/elastic/beats/v7/libbeat/common/kubernetes/metadata"
	"github.com/elastic/beats/v7/libbeat/common/safemapstr"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/agent/errors"
	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/composable"
)

type node struct {
	logger         *logp.Logger
	cleanupTimeout time.Duration
	comm           composable.DynamicProviderComm
	scope          string
	config         *Config
	metagen        metadata.MetaGen
	watcher        kubernetes.Watcher
}

type nodeData struct {
	node       *kubernetes.Node
	mapping    map[string]interface{}
	processors []map[string]interface{}
}

// NewNodeEventer creates an eventer that can discover and process node objects
func NewNodeEventer(
	comm composable.DynamicProviderComm,
	cfg *Config,
	logger *logp.Logger,
	client k8s.Interface,
	scope string) (Eventer, error) {
	watcher, err := kubernetes.NewNamedWatcher("agent-node", client, &kubernetes.Node{}, kubernetes.WatchOptions{
		SyncTimeout:  cfg.SyncPeriod,
		Node:         cfg.Node,
		IsUpdated:    isUpdated,
		HonorReSyncs: true,
	}, nil)
	if err != nil {
		return nil, errors.New(err, "couldn't create kubernetes watcher")
	}

	rawConfig, err := common.NewConfigFrom(cfg)
	if err != nil {
		return nil, errors.New(err, "failed to unpack configuration")
	}
	metaGen := metadata.NewNodeMetadataGenerator(rawConfig, watcher.Store(), client)
	n := &node{
		logger,
		cfg.CleanupTimeout,
		comm,
		scope,
		cfg,
		metaGen,
		watcher}
	watcher.AddEventHandler(n)

	return n, nil
}

func (n *node) emitRunning(node *kubernetes.Node) {
	data := generateNodeData(node, n.config, n.metagen)
	if data == nil {
		return
	}
	data.mapping["scope"] = n.scope

	// Emit the node
	n.comm.AddOrUpdate(string(node.GetUID()), NodePriority, data.mapping, data.processors)
}

func (n *node) emitStopped(node *kubernetes.Node) {
	n.comm.Remove(string(node.GetUID()))
}

// Start starts the eventer
func (n *node) Start() error {
	return n.watcher.Start()
}

// Stop stops the eventer
func (n *node) Stop() {
	n.watcher.Stop()
}

// OnAdd ensures processing of node objects that are newly created
func (n *node) OnAdd(obj interface{}) {
	n.logger.Debugf("Watcher Node add: %+v", obj)
	n.emitRunning(obj.(*kubernetes.Node))
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
		time.AfterFunc(n.cleanupTimeout, func() { n.emitStopped(node) })
	} else {
		n.logger.Debugf("Watcher Node update: %+v", obj)
		n.emitRunning(node)
	}
}

// OnDelete ensures processing of node objects that are deleted
func (n *node) OnDelete(obj interface{}) {
	n.logger.Debugf("Watcher Node delete: %+v", obj)
	node := obj.(*kubernetes.Node)
	time.AfterFunc(n.cleanupTimeout, func() { n.emitStopped(node) })
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

// getAddress returns the IP of the node Resource. If there is a
// NodeExternalIP then it is returned, if not then it will try to find
// an address of NodeExternalIP type and if not found it looks for a NodeHostName address type
func getAddress(node *kubernetes.Node) string {
	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP && address.Address != "" {
			return address.Address
		}
	}

	for _, address := range node.Status.Addresses {
		if address.Type == v1.NodeExternalIP && address.Address != "" {
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

func generateNodeData(node *kubernetes.Node, cfg *Config, kubeMetaGen metadata.MetaGen) *nodeData {
	host := getAddress(node)

	// If a node doesn't have an IP then dont monitor it
	if host == "" {
		return nil
	}

	// If the node is not in ready state then dont monitor it
	if !isNodeReady(node) {
		return nil
	}

	meta := kubeMetaGen.Generate(node)
	kubemetaMap, err := meta.GetValue("kubernetes")
	if err != nil {
		return &nodeData{}
	}

	// Pass annotations to all events so that it can be used in templating and by annotation builders.
	annotations := common.MapStr{}
	for k, v := range node.GetObjectMeta().GetAnnotations() {
		safemapstr.Put(annotations, k, v)
	}

	// k8sMapping includes only the metadata that fall under kubernetes.*
	// and these are available as dynamic vars through the provider
	k8sMapping := map[string]interface{}(kubemetaMap.(common.MapStr).Clone())

	// add annotations to be discoverable by templates
	k8sMapping["annotations"] = annotations

	processors := []map[string]interface{}{}
	// meta map includes metadata that go under kubernetes.*
	// but also other ECS fields like orchestrator.*
	for field, metaMap := range meta {
		processor := map[string]interface{}{
			"add_fields": map[string]interface{}{
				"fields": metaMap,
				"target": field,
			},
		}
		processors = append(processors, processor)
	}
	return &nodeData{
		node:       node,
		mapping:    k8sMapping,
		processors: processors,
	}
}
