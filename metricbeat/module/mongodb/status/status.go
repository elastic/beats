package status

import (
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
	if err := mb.Registry.AddMetricSet("mongodb", "status", New); err != nil {
		panic(err)
	}
}

type MetricSet struct {
	mb.BaseMetricSet
	dialInfo *mgo.DialInfo
}

func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct {
		Hosts    []string `config:"hosts"    validate:"nonzero,required"`
		Username string   `config:"username"`
		Password string   `config:"username"`
	}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	info, err := mongodb.ParseURL(base.Host(), config.Username, config.Password)
	if err != nil {
		return nil, err
	}
	info.Timeout = base.Module().Config().Timeout

	return &MetricSet{
		BaseMetricSet: base,
		dialInfo:      info,
	}, nil
}

func (m *MetricSet) Fetch() (common.MapStr, error) {

	session, err := mgo.DialWithInfo(m.dialInfo)
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
