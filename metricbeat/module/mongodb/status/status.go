package status

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
TODOs:
	* add metricset for "locks" data
	* add a metricset for "metrics" data
*/

var debugf = logp.MakeDebug("mongodb.status")

func init() {
	if err := mb.Registry.AddMetricSet("mongodb", "status", New, mongodb.ParseURL); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	mongoSession *mgo.Session
}

// New creates a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	dialInfo, err := mgo.ParseURL(base.HostData().URI)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = base.Module().Config().Timeout

	// instantiate direct connections to Mongo host
	mongoSession, err := mongodb.NewDirectSession(dialInfo)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		mongoSession:  mongoSession,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() (common.MapStr, error) {
	result := map[string]interface{}{}
	if err := m.mongoSession.DB("admin").Run(bson.D{{Name: "serverStatus", Value: 1}}, &result); err != nil {
		return nil, err
	}

	data, _ := schema.Apply(result)
	return data, nil
}
