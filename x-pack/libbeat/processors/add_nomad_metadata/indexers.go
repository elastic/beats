// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package add_nomad_metadata

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/v7/libbeat/logp"
	"github.com/elastic/beats/v7/x-pack/libbeat/common/nomad"
	conf "github.com/elastic/elastic-agent-libs/config"
	"github.com/elastic/elastic-agent-libs/mapstr"
)

const (
	AllocationNameIndexerName = "allocation_name"
	AllocationUUIDIndexerName = "allocation_uid"
)

// Indexer takes an allocation and generate all the metadata we need to enrich events in a efficient
// way. By preindexing the metadata in the way it will be checked when matching events
type Indexer interface {
	// GetMetadata generates event metadata for the given allocation, then returns the
	// list of indexes to create, with the metadata to put on them
	GetMetadata(alloc *nomad.Resource) []MetadataIndex

	// GetIndexes return the list of indexes the given allocation belongs to. This function must
	// return the same indexes than GetMetadata
	GetIndexes(alloc *nomad.Resource) []string
}

// MetadataIndex holds a pair of index to metadata
type MetadataIndex struct {
	Index string
	Data  mapstr.M
}

// Indexers holds a collections of Indexer objects and the associated lock
type Indexers struct {
	sync.RWMutex
	indexers []Indexer
}

// IndexerConstructor builds a new indexer from its settings
type IndexerConstructor func(config conf.C, metaGen nomad.MetaGenerator) (Indexer, error)

// NewIndexers builds an Indexers object from its configurations
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
func NewAllocationNameIndexer(_ conf.C, metaGen nomad.MetaGenerator) (Indexer, error) {
	return &AllocationNameIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns metadata for the given resource, if it matches the index
func (p *AllocationNameIndexer) GetMetadata(alloc *nomad.Resource) []MetadataIndex {
	meta := p.metaGen.ResourceMetadata(*alloc)

	return []MetadataIndex{
		{
			Index: fmt.Sprintf("%s/%s", alloc.Namespace, alloc.Name),
			Data:  meta,
		},
	}
}

// GetIndexes returns the indexes for the given allocation
func (p *AllocationNameIndexer) GetIndexes(alloc *nomad.Resource) []string {
	return []string{fmt.Sprintf("%s/%s", alloc.Namespace, alloc.Name)}
}

// AllocationUUIDIndexer indexes allocations based on the allocation id
type AllocationUUIDIndexer struct {
	metaGen nomad.MetaGenerator
}

// NewAllocationUUIDIndexer initializes and returns a AllocationUUIDIndexer
func NewAllocationUUIDIndexer(_ conf.C, metaGen nomad.MetaGenerator) (Indexer, error) {
	return &AllocationUUIDIndexer{metaGen: metaGen}, nil
}

// GetMetadata returns the composed metadata from AllocationNameIndexer and the allocation id
func (p *AllocationUUIDIndexer) GetMetadata(alloc *nomad.Resource) []MetadataIndex {
	data := p.metaGen.ResourceMetadata(*alloc)

	return []MetadataIndex{
		{
			Index: alloc.ID,
			Data:  data,
		},
	}
}

// GetIndexes returns the indexes for the given allocation
func (p *AllocationUUIDIndexer) GetIndexes(alloc *nomad.Resource) []string {
	return []string{alloc.ID}
}
