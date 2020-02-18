package users

import (
	"github.com/coreos/go-systemd/login1"
	"github.com/elastic/beats/libbeat/common"
	"github.com/elastic/beats/libbeat/common/cfgwarn"
	"github.com/elastic/beats/metricbeat/mb"
	"github.com/pkg/errors"
)

type userInfo struct {
	UID      uint32
	User     string
	Path     string
	Sessions []login1.Session
}

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("system", "users", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
	conn    *login1.Conn
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The system users metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	conn, err := login1.New()
	if err != nil {
		return nil, errors.Wrap(err, "error connecting to dbus")
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
		conn:          conn,
	}, nil
}

// Fetch methods implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	users, err := m.conn.ListUsers()
	if err != nil {
		return errors.Wrap(err, "error listing users")
	}
	sessions, err := m.conn.ListSessions()
	if err != nil {
		return errors.Wrap(err, "error listing sessions")
	}

	eventMapping(users, sessions, report)

	return nil
}

// eventMapping iterates through the lists of users and sessions, combining the two
func eventMapping(users []login1.User, sessions []login1.Session, report mb.ReporterV2) error {
	sessionList := []string{}
	for _, user := range users {
		for _, session := range sessions {
			if session.UID == user.UID {
				sessionList = append(sessionList, session.ID)
			}
		}
		reported := report.Event(mb.Event{
			RootFields: common.MapStr{
				"user": common.MapStr{
					"name": user.Name,
					"id":   user.UID,
				},
			},
			MetricSetFields: common.MapStr{
				"path":     user.Path,
				"sessions": sessionList,
			},
		},
		)
		//if the channel is closed and metricbeat is shutting down, just return
		if !reported {
			break
		}
	}
	return nil
}
