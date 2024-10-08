// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package azure

import (
	"sync"
	"time"
)

// Resource will contain the main azure resource details
type Resource struct {
	Id           string
	Name         string
	Location     string
	Group        string
	Tags         map[string]string
	Subscription string
	Type         string
	// will be filled if cloud data is necessary, atm only in case of compute_vm and compute_vm_scaleset
	Vms []VmResource
}

// VmResource contains details specific to a vm type of resource
type VmResource struct {
	Size string
	Id   string
	Name string
}

// Metric will contain the main azure metric details
type Metric struct {
	Namespace    string
	Names        []string
	Aggregations string
	Dimensions   []Dimension
	Values       []MetricValue
	TimeGrain    string
	ResourceId   string
	// ResourceSubId is used for the metric values api as namespaces can apply to sub resources ex. storage account: container, blob, vm scaleset: vms
	ResourceSubId string
}

// Dimension represents the azure metric dimension details
type Dimension struct {
	Name  string
	Value string
}

// MetricValue represents the azure metric values
type MetricValue struct {
	name       string
	avg        *float64
	min        *float64
	max        *float64
	total      *float64
	count      *float64
	timestamp  time.Time
	dimensions []Dimension
}

// ResourceConfiguration represents the resource related configuration entered by the user
type ResourceConfiguration struct {
	Metrics         []Metric
	RefreshInterval time.Duration
	lastUpdate      struct {
		time.Time
		sync.Mutex
	}
}

// Expired will check for an expiration time and assign a new one
func (p *ResourceConfiguration) Expired() bool {
	if p.RefreshInterval <= 0 {
		return true
	}
	p.lastUpdate.Lock()
	defer p.lastUpdate.Unlock()
	if p.lastUpdate.Add(p.RefreshInterval).After(time.Now()) {
		return false
	}
	p.lastUpdate.Time = time.Now()
	return true
}
