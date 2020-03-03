// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/common"
	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/nomad"
)

const (
	AllocationNameIndexerName = "allocation_name"
	AllocationUUIDIndexerName = "allocation_uid"
)

// Indexer take known pods and generate all the metadata we need to enrich
// events in a efficient way. By preindexing the metadata in the way it will be
// checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given pod, then returns the
	// list of indexes to create, with the metadata to put on them
	GetMetadata(alloc *nomad.Resource) []MetadataIndex

	// GetIndexes return the list of indexes the given pod belongs to. This function
	// must return the same indexes than GetMetadata
	GetIndexes(alloc *nomad.Resource) []string
}

// MetadataIndex holds a pair of index -> metadata info
type MetadataIndex struct {
	Index string
	Data  common.MapStr
}

type Indexers struct {
	sync.RWMutex
	indexers []Indexer
}

// IndexerConstructor builds a new indexer from its settings
type IndexerConstructor func(config common.Config, metaGen nomad.MetaGenerator) (Indexer, error)

// NewIndexers builds indexers object
func NewIndexers(configs PluginConfig, metaGen nomad.MetaGenerator) *Indexers {
	indexers := []Indexer{}
	for _, pluginConfigs := range configs {
		for name, pluginConfig := range pluginConfigs {
			indexFunc := Indexing.GetIndexer(name)
			if indexFunc == nil {
				logp.Warn("Unable to find indexing plugin %s", name)
				continue
			}

			indexer, err := indexFunc(pluginConfig, metaGen)
			if err != nil {
				logp.Warn("Unable to initialize indexing plugin %s due to error %v", name, err)
				continue
			}

			indexers = append(indexers, indexer)
		}
	}

	return &Indexers{
		indexers: indexers,
	}
}

// GetIndexes returns the composed index list from all registered indexers
func (i *Indexers) GetIndexes(alloc *nomad.Resource) []string {
	var indexes []string
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, i := range indexer.GetIndexes(alloc) {
			indexes = append(indexes, i)
		}
	}

	return indexes
}

// GetMetadata returns the composed metadata list from all registered indexers
func (i *Indexers) GetMetadata(alloc *nomad.Resource) []MetadataIndex {
	var metadata []MetadataIndex
	i.RLock()
	defer i.RUnlock()
	for _, indexer := range i.indexers {
		for _, m := range indexer.GetMetadata(alloc) {
			metadata = append(metadata, m)
		}
	}

	return metadata
}

// Empty returns true if indexers list is empty
func (i *Indexers) Empty() bool {
	i.RLock()
	defer i.RUnlock()
	if len(i.indexers) == 0 {
		return true
	}

	return false
}

// AllocationNameIndexer implements default indexer based on the allocation name
type AllocationNameIndexer struct {
	metaGen nomad.MetaGenerator
}

// NewAllocationNameIndexer initializes and returns a AllocationNameIndexer
func NewAllocationNameIndexer(_ common.Config, metaGen nomad.MetaGenerator) (Indexer, error) {
	return &AllocationNameIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given pod, if it matches the index
func (p *AllocationNameIndexer) GetMetadata(alloc *nomad.Resource) []MetadataIndex {
	meta := p.metaGen.ResourceMetadata(*alloc)

	return []MetadataIndex{
		{
			Index: fmt.Sprintf("%s/%s", alloc.Namespace, alloc.Name),
			Data:  meta,
		},
	}
}

// GetIndexes returns the indexes for the given Pod
func (p *AllocationNameIndexer) GetIndexes(alloc *nomad.Resource) []string {
	return []string{fmt.Sprintf("%s/%s", alloc.Namespace, alloc.Name)}
}

// AllocationUUIDIndexer indexes pods based on the pod UID
type AllocationUUIDIndexer struct {
	metaGen nomad.MetaGenerator
}

// NewAllocationUUIDIndexer initializes and returns a AllocationUUIDIndexer
func NewAllocationUUIDIndexer(_ common.Config, metaGen nomad.MetaGenerator) (Indexer, error) {
	return &AllocationUUIDIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns the composed metadata from AllocationNameIndexer and the pod UID
func (p *AllocationUUIDIndexer) GetMetadata(alloc *nomad.Resource) []MetadataIndex {
	data := p.metaGen.ResourceMetadata(*alloc)

	return []MetadataIndex{
		{
			Index: alloc.ID,
			Data:  data,
		},
	}
}

// GetIndexes returns the indexes for the given Pod
func (p *AllocationUUIDIndexer) GetIndexes(alloc *nomad.Resource) []string {
	return []string{alloc.ID}
}
