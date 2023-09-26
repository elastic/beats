package metrics

import (
	"github.com/elastic/beats/v7/metricbeat/mb"
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestCreateDimensionsKey(t *testing.T) {
	kv := KeyValuePoint{
		Key:   "metric1",
		Value: 1,
		Labels: mapstr.M{
			"user.deployment":     "deploy-1",
			"user.division":       "div-1",
			"user.index":          "n0",
			"user.instance_group": "ig1",
			"user.job":            "j1",
			"user.name":           "name-1",
			"user.org":            "obs",
			"user.project":        "project-1",
		},
		ECS: mapstr.M{
			"cloud.account.id":        "obs",
			"cloud.availability_zone": "us-west-1",
			"cloud.instance.id":       "1",
			"cloud.provider":          "gcp",
			"cloud.region":            "us-west",
		},
		Timestamp: time.Time{},
	}

	dimensionsKey := createDimensionsKey(kv)
	require.Equal(t, "obs_us-west-1_1_gcp_us-west_{\"user.deployment\":\"deploy-1\",\"user.division\":\"div-1\",\"user.index\":\"n0\",\"user.instance_group\":\"ig1\",\"user.job\":\"j1\",\"user.name\":\"name-1\",\"user.org\":\"obs\",\"user.project\":\"project-1\"}", dimensionsKey)
}

func TestGroupMetricsByDimensions(t *testing.T) {
	t.Run("same dimensions", func(t *testing.T) {
		kvs := []KeyValuePoint{
			{
				Key:   "metric1",
				Value: 1,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric2",
				Value: 2,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric3",
				Value: 3,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: time.Time{},
			},
		}

		expectedGroup1 := kvs[:]

		groups := groupMetricsByDimensions(kvs)
		require.Len(t, groups, 1)

		group1, ok := groups["obs_us-west-1_1_gcp_us-west_{\"user.deployment\":\"deploy-1\",\"user.division\":\"div-1\",\"user.index\":\"n0\",\"user.instance_group\":\"ig1\",\"user.job\":\"j1\",\"user.name\":\"name-1\",\"user.org\":\"obs\",\"user.project\":\"project-1\"}"]
		require.True(t, ok)
		require.Len(t, group1, 3) // all 3 metrics in this group
		require.ElementsMatch(t, group1, expectedGroup1)
	})

	t.Run("different dimensions", func(t *testing.T) {
		kvs := []KeyValuePoint{
			{
				Key:   "metric1",
				Value: 1,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric2",
				Value: 2,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric3",
				Value: 3,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric4",
				Value: 4,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: time.Time{},
			},
			{
				Key:   "metric5",
				Value: 5,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "2",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: time.Time{},
			},
		}

		expectedGroup1 := kvs[:2]          // first 2 metrics
		expectedGroup2 := kvs[2:4]         // next 2 metrics; different AZ and region
		expectedGroup3 := kvs[len(kvs)-1:] // last metric; same as previous but different cloud.instance.id

		groups := groupMetricsByDimensions(kvs)
		// we should have 3 groups
		// key 1: obs_us-west-1_1_gcp_us-west_ + gcp.labels as JSON
		// key 2: obs_us-east-1_1_gcp_us-east_ + gcp.labels as JSON
		// key 3: obs_us-east-1_2_gcp_us-east_ + gcp.labels as JSON

		require.Len(t, groups, 3)

		group1, ok := groups["obs_us-west-1_1_gcp_us-west_{\"user.deployment\":\"deploy-1\",\"user.division\":\"div-1\",\"user.index\":\"n0\",\"user.instance_group\":\"ig1\",\"user.job\":\"j1\",\"user.name\":\"name-1\",\"user.org\":\"obs\",\"user.project\":\"project-1\"}"]
		require.True(t, ok)
		require.Len(t, group1, 2) // should have 2 metrics
		require.ElementsMatch(t, group1, expectedGroup1)

		group2, ok := groups["obs_us-east-1_1_gcp_us-east_{\"user.deployment\":\"deploy-1\",\"user.division\":\"div-1\",\"user.index\":\"n0\",\"user.instance_group\":\"ig1\",\"user.job\":\"j1\",\"user.name\":\"name-1\",\"user.org\":\"obs\",\"user.project\":\"project-1\"}"]
		require.True(t, ok)
		require.Len(t, group2, 2) // should have 2 metrics
		require.ElementsMatch(t, group2, expectedGroup2)

		group3, ok := groups["obs_us-east-1_2_gcp_us-east_{\"user.deployment\":\"deploy-1\",\"user.division\":\"div-1\",\"user.index\":\"n0\",\"user.instance_group\":\"ig1\",\"user.job\":\"j1\",\"user.name\":\"name-1\",\"user.org\":\"obs\",\"user.project\":\"project-1\"}"]
		require.True(t, ok)
		require.Len(t, group3, 1) // should have 1 metric
		require.ElementsMatch(t, group3, expectedGroup3)
	})
}

func TestCreateEventsFromGroup(t *testing.T) {
	timestampGroup1 := time.Now()
	timestampGroup2 := time.Now().Add(5 * time.Minute)
	timestampGroup3 := time.Now().Add(10 * time.Minute)

	t.Run("different dimensions", func(t *testing.T) {
		kvs := []KeyValuePoint{
			{
				Key:   "metric1",
				Value: 1,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: timestampGroup1,
			},
			{
				Key:   "metric2",
				Value: 2,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
				Timestamp: timestampGroup1,
			},
			{
				Key:   "metric3",
				Value: 3,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: timestampGroup2,
			},
			{
				Key:   "metric4",
				Value: 4,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-1",
					"user.org":            "obs",
					"user.project":        "project-1",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: timestampGroup2,
			},
			{
				Key:   "metric5",
				Value: 5,
				Labels: mapstr.M{
					"user.deployment":     "deploy-1",
					"user.division":       "div-1",
					"user.index":          "n0",
					"user.instance_group": "ig1",
					"user.job":            "j1",
					"user.name":           "name-2",
					"user.org":            "obs",
					"user.project":        "project-2",
				},
				ECS: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "2",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
				Timestamp: timestampGroup3,
			},
		}

		groups := groupMetricsByDimensions(kvs)

		expectedEvents := []mb.Event{
			{
				Timestamp: timestampGroup1,
				ModuleFields: mapstr.M{
					"labels": mapstr.M{
						"user.deployment":     "deploy-1",
						"user.division":       "div-1",
						"user.index":          "n0",
						"user.instance_group": "ig1",
						"user.job":            "j1",
						"user.name":           "name-1",
						"user.org":            "obs",
						"user.project":        "project-1",
					},
				},
				MetricSetFields: mapstr.M{
					"metric1": 1,
					"metric2": 2,
				},
				RootFields: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-west-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-west",
				},
			},
			{
				Timestamp: timestampGroup2,
				ModuleFields: mapstr.M{
					"labels": mapstr.M{
						"user.deployment":     "deploy-1",
						"user.division":       "div-1",
						"user.index":          "n0",
						"user.instance_group": "ig1",
						"user.job":            "j1",
						"user.name":           "name-1",
						"user.org":            "obs",
						"user.project":        "project-1",
					},
				},
				MetricSetFields: mapstr.M{
					"metric3": 3,
					"metric4": 4,
				},
				RootFields: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "1",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
			},
			{
				Timestamp: timestampGroup3,
				ModuleFields: mapstr.M{
					"labels": mapstr.M{
						"user.deployment":     "deploy-1",
						"user.division":       "div-1",
						"user.index":          "n0",
						"user.instance_group": "ig1",
						"user.job":            "j1",
						"user.name":           "name-2",
						"user.org":            "obs",
						"user.project":        "project-2",
					},
				},
				MetricSetFields: mapstr.M{
					"metric5": 5,
				},
				RootFields: mapstr.M{
					"cloud.account.id":        "obs",
					"cloud.availability_zone": "us-east-1",
					"cloud.instance.id":       "2",
					"cloud.provider":          "gcp",
					"cloud.region":            "us-east",
				},
			},
		}

		events := createEventsFromGroups("redis", groups)
		require.Len(t, events, 3)
		require.ElementsMatch(t, events, expectedEvents)
	})
}
