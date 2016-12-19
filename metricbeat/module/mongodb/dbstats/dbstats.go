package dbstats

import (
	"errors"

	"github.com/beats/libbeat/logp"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"gopkg.in/mgo.v2"
)

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("mongodb", "dbstats", New); err != nil {
		panic(err)
	}
}

// MetricSet type defines all fields of the MetricSet
// As a minimum it must inherit the mb.BaseMetricSet fields, but can be extended with
// additional entries. These variables can be used to persist data or configuration between
// multiple fetch calls.
type MetricSet struct {
	mb.BaseMetricSet
	dialInfo *mgo.DialInfo
}

// New create a new instance of the MetricSet
// Part of new is also setting up the configuration by processing additional
// configuration entries if needed.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	dialInfo, err := mgo.ParseURL(base.HostData().URI)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = base.Module().Config().Timeout

	return &MetricSet{
		BaseMetricSet: base,
		dialInfo:      dialInfo,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	// establish connection to mongo
	session, err := mgo.DialWithInfo(m.dialInfo)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)

	// Get the list of databases database names, which we'll use to call db.stats() on each
	dbNames, err := session.DatabaseNames()
	if err != nil {
		logp.Err("Error retrieving database names from Mongo instance")
		return []common.MapStr{}, err
	}

	// events is the list of events collected from each of the databases.
	events := []common.MapStr{}

	// for each database, call db.stats() and append to events
	for _, dbName := range dbNames {
		db := session.DB(dbName)

		result := map[string]interface{}{}

		err := db.Run("dbStats", &result)
		if err != nil {
			logp.Err("Failed to retrieve stats for db %s", dbName)
			continue
		}
		events = append(events, eventMapping(result))
	}

	// if we failed to collect on any databases, return an error
	if len(events) == 0 {
		err = errors.New("Failed to fetch stats for all databases in mongo instance")
		return []common.MapStr{}, err
	}

	return events, nil
}
