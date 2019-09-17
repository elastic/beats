package azure

import (
	"sync"
	"time"
)

// Resource will contain the main azure resource details
type Resource struct {
	ID           string
	Name         string
	Location     string
	Type         string
	Group        string
	Tags         map[string]string
	Subscription string
}

// Metric will contain the main azure metric details
type Metric struct {
	Resource     Resource
	Namespace    string
	Names        []string
	Aggregations string
	Dimensions   []Dimension
	Values       []MetricValue
	TimeGrain    string
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
