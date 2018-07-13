package oplog

import (
	"errors"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/logp"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"
	"gopkg.in/mgo.v2/bson"
)

const oplog_col = "oplog.rs"

var debugf = logp.MakeDebug("mongodb.oplog")

func init() {
	logp.Info("initializing oplog")
	mb.Registry.MustAddMetricSet("mongodb", "oplog", New,
		mb.WithHostParser(mongodb.ParseURL),
		mb.DefaultMetricSet())
}

type MetricSet struct {
	*mongodb.MetricSet
}

func contains(s []string, x string) bool {
	for _, n := range s {
		if x == n {
			return true
		}
	}
	return false
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	ms, err := mongodb.NewMetricSet(base)
	if err != nil {
		return nil, err
	}
	return &MetricSet{ms}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch() (common.MapStr, error) {
	// instantiate direct connections to each of the configured Mongo hosts
	mongoSession, err := mongodb.NewDirectSession(m.DialInfo)
	if err != nil {
		return nil, err
	}
	defer mongoSession.Close()

	// get oplog.rs collection
	db := mongoSession.DB("local")
	if collections, err := db.CollectionNames(); err != nil || !contains(collections, oplog_col) {
		if err == nil {
			err = errors.New("Collection oplog.rs was not found")
		}

		logp.Err(err.Error())
		return nil, err
	}
	collection := db.C(oplog_col)

	//  oplog size
	var oplogStatus map[string]interface{}	
	if err := db.Run(bson.D{{Name: "collStats", Value: oplog_col}}, &oplogStatus); err != nil {
		return nil, err
	}

	allocated := oplogStatus["maxSize"].(int64)
	used := int64(oplogStatus["size"].(float64))

	// get first and last items in the oplog
	oplog_iter := collection.Find(nil).Sort("$natural").Iter()
	oplog_reverse_iter := collection.Find(nil).Sort("-$natural").Iter()
	var first, last interface{}
	if !oplog_iter.Next(&first) || !oplog_reverse_iter.Next(&last) {
		err := errors.New("Objects not found in local.oplog.rs -- Is this a new and empty db instance?")
		logp.Err(err.Error())
		return nil, err
	}
	
	firstTs := int64(first.(bson.M)["ts"].(bson.MongoTimestamp))
	lastTs := int64(last.(bson.M)["ts"].(bson.MongoTimestamp))
	diff := lastTs - firstTs

	result := map[string]interface{} {
		"logSize": allocated,
		"used": used,
		"tFirst": firstTs,
		"tLast": lastTs,
		"timeDiff": diff,
	}
	event, _ := schema.Apply(result)
	
	return event, nil
}
