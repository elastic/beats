package status

import (
	"fmt"
	"sync"

	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/elastic/beats/metricbeat/module/mongodb"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
TODOs:
	* add metricset for "locks" data
	* add a metricset for "metrics" data
*/

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
	dialInfo      *mgo.DialInfo
	mongoSessions []*mgo.Session
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

	// instantiate direct connections to each of the configured Mongo hosts
	mongoSessions, err := mongodb.NewDirectSessions(dialInfo.Addrs, dialInfo)
	if err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		dialInfo:      dialInfo,
		mongoSessions: mongoSessions,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right format
// It returns the event which is then forward to the output. In case of an error, a
// descriptive error must be returned.
func (m *MetricSet) Fetch() ([]common.MapStr, error) {

	// create a wait group because we're going to spawn a goroutine for each host target
	var wg sync.WaitGroup
	wg.Add(len(m.mongoSessions))

	// events is the value returned by this function
	var events []common.MapStr

	// created buffered channel to receive async results from each of the nodes
	channel := make(chan interface{}, len(m.mongoSessions))

	for _, mongo := range m.mongoSessions {
		go func(mongo *mgo.Session) {
			defer wg.Done()
			channel <- m.fetchNodeStatus(mongo)
		}(mongo)
	}

	// wait for goroutines to complete
	wg.Wait()
	close(channel)

	// pull results off of the channel and append to events
	for data := range channel {
		events = append(events, data.(common.MapStr))
	}

	// if we didn't get results from any node, return an error
	if len(events) == 0 {
		err := errors.New("Failed to retrieve db stats from all nodes")
		return events, err
	}

	fmt.Printf("%v", events)

	return events, nil
}

func (m *MetricSet) fetchNodeStatus(session *mgo.Session) common.MapStr {
	result := common.MapStr{}
	if err := session.DB("admin").Run(bson.D{{"serverStatus", 1}}, &result); err != nil {
		return nil
	}

	return eventMapping(result)
}
