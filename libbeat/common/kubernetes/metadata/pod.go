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

package metadata

import (
	"context"
	"fmt"

	k8s "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/kubernetes"
	"github.com/elastic/beats/libbeat/common/safemapstr"
)

type pod struct {
	store      cache.Store
	cancelFunc context.CancelFunc
	node       MetaGen
	namespace  MetaGen
	resource   *resource
}

func NewPodMetadataGenerator(cfg *Config, acfg *AddResourceMetadataConfig, client k8s.Interface, options kubernetes.WatchOptions) (MetaGen, error) {
	ctx, cancelFunc := context.WithCancel(context.Background())
	po := &pod{
		resource:   NewResourceMetadataGenerator(cfg),
		cancelFunc: cancelFunc,
	}

	if acfg.Namespace != nil && acfg.Namespace.Enabled() {
		// add namespace generator here
	}

	if acfg.Node != nil && acfg.Node.Enabled() {
		// add node generator here
	}

	inf, _, err := kubernetes.NewInformer(client, &kubernetes.Pod{}, options)
	if err != nil {
		return nil, err
	}
	po.store = inf.GetStore()

	go inf.Run(ctx.Done())

	if !cache.WaitForCacheSync(ctx.Done(), inf.HasSynced) {
		return nil, fmt.Errorf("kubernetes informer unable to sync cache")
	}

	return po, nil
}

func (p *pod) Generate(obj kubernetes.Resource, opts ...FieldOptions) common.MapStr {
	po, ok := obj.(*kubernetes.Pod)
	if !ok {
		return nil
	}

	out := p.resource.Generate(obj, opts...)

	safemapstr.Put(out, "pod.uid", string(po.GetObjectMeta().GetUID()))
	safemapstr.Put(out, "pod.name", po.GetObjectMeta().GetName())

	if p.node != nil {
		p.node.GenerateFromName(po.Spec.NodeName)
	} else {
		safemapstr.Put(out, "node.name", po.Spec.NodeName)
	}

	if p.namespace != nil {
		p.namespace.GenerateFromName(po.GetNamespace())
	}

	return out
}

func (p *pod) GenerateFromName(name string, opts ...FieldOptions) common.MapStr {
	if obj, ok, _ := p.store.GetByKey(name); ok {
		po, ok := obj.(*kubernetes.Pod)
		if !ok {
			return nil
		}

		return p.Generate(po, opts...)
	} else {
		return nil
	}
}

func (p *pod) Stop() {
	p.cancelFunc()
}
