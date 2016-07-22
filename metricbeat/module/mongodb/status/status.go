package status

import (
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/metricbeat/mb"

	"github.com/pkg/errors"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

/*
TODOs:
	* add support for username/password
	* add metricset for "locks" data
	* add a metricset for "metrics" data
*/

func init() {
	if err := mb.Registry.AddMetricSet("mongodb", "status", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Hosts []string `config:"hosts"    validate:"nonzero,required"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {

	session, err := mgo.DialWithTimeout(m.Host(), m.Module().Config().Timeout)
	if err != nil {
		return nil, err
	}
	defer session.Close()

	session.SetMode(mgo.Monotonic, true)
	result := map[string]interface{}{}
	if err := session.DB("admin").Run(bson.D{{"serverStatus", 1}}, &result); err != nil {
		return nil, errors.Wrap(err, "mongodb fetch failed")
	}

	return eventMapping(result), nil
}
