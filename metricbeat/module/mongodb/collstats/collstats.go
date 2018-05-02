package collstats

import (
	"errors"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"

	"gopkg.in/mgo.v2"
)

var debugf = logp.MakeDebug("mongodb.collstats")

func init() {
	mb.Registry.MustAddMetricSet("mongodb", "collstats", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet(),
	)
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
	// events is the list of events collected from each of the collections.
	var events []common.MapStr

	// instantiate direct connections to each of the configured Mongo hosts
	mongoSession, err := mongodb.NewDirectSession(m.dialInfo)
	if err != nil {
		return nil, err
	}
	defer mongoSession.Close()

	result := common.MapStr{}

	err = mongoSession.Run("top", &result)
	if err != nil {
		logp.Err("Error retrieving collection totals from Mongo instance")
		return events, err
	}

	if _, ok := result["totals"]; !ok {
		err = errors.New("Error accessing collection totals in returned data")
		logp.Err(err.Error())
		return events, err
	}

	totals, ok := result["totals"].(common.MapStr)
	if !ok {
		err = errors.New("Collection totals are not a map")
		logp.Err(err.Error())
		return events, err
	}

	for group, info := range totals {
		if group == "note" {
			continue
		}

		infoMap, ok := info.(common.MapStr)
		if !ok {
			err = errors.New("Unexpected data returned by mongodb")
			logp.Err(err.Error())
			continue
		}

		event, err := eventMapping(group, infoMap)
		if err != nil {
			logp.Err("Mapping of the event data filed")
			continue
		}

		events = append(events, event)
	}

	return events, nil
}
