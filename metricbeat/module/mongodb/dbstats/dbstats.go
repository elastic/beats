package dbstats

import (
	"errors"

	"gopkg.in/mgo.v2"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"
)

var debugf = logp.MakeDebug("mongodb.dbstats")

// init registers the MetricSet with the central registry.
// The New method will be called after the setup of the module and before starting to fetch data
func init() {
	if err := mb.Registry.AddMetricSet("mongodb", "dbstats", New, mongodb.ParseURL); err != nil {
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
	logp.Experimental("The %v %v metricset is experimental", base.Module().Name(), base.Name())

	dialInfo, err := mgo.ParseURL(base.HostData().URI)
	if err != nil {
		return nil, err
	}
	dialInfo.Timeout = base.Module().Config().Timeout

	// instantiate direct connections to each of the configured Mongo hosts
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
func (m *MetricSet) Fetch() ([]common.MapStr, error) {
	// events is the list of events collected from each of the databases.
	var events []common.MapStr

	// Get the list of databases names, which we'll use to call db.stats() on each
	dbNames, err := m.mongoSession.DatabaseNames()
	if err != nil {
		logp.Err("Error retrieving database names from Mongo instance")
		return events, err
	}

	// for each database, call db.stats() and append to events
	for _, dbName := range dbNames {
		db := m.mongoSession.DB(dbName)

		result := common.MapStr{}

		err := db.Run("dbStats", &result)
		if err != nil {
			logp.Err("Failed to retrieve stats for db %s", dbName)
			continue
		}
		data, _ := schema.Apply(result)
		events = append(events, data)
	}

	if len(events) == 0 {
		err = errors.New("Failed to retrieve dbStats from any databases")
		logp.Err(err.Error())
		return events, err
	}

	return events, nil
}
