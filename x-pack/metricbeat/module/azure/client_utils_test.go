package azure

import (
	"testing"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/preview/monitor/mgmt/2019-06-01/insights"
	"github.com/Azure/go-autorest/autorest/date"
	"github.com/stretchr/testify/assert"
)

func TestMetricExists(t *testing.T) {
	fl := 12.4
	fl1 := 1.0
	location := time.Location{}
	date1 := time.Date(2019, 12, 12, 12, 12, 12, 12, &location)
	stamp := date.Time{
		Time: date1,
	}
	var name = "Requests"
	insightValue := insights.MetricValue{
		TimeStamp: &stamp,
		Average:   &fl,
		Minimum:   &fl1,
		Maximum:   nil,
		Total:     nil,
		Count:     nil,
	}
	var metricValues = []MetricValue{
		{
			name:      "Requests",
			avg:       &fl,
			min:       &fl1,
			max:       nil,
			total:     nil,
			count:     nil,
			timestamp: date1,
		},
		{
			name:      "TotalRequests",
			avg:       &fl,
			min:       &fl1,
			max:       nil,
			total:     nil,
			count:     &fl1,
			timestamp: date1,
		},
	}

	result := metricExists(name, insightValue, metricValues)
	assert.True(t, result)
	metricValues[0].name = "TotalRequests"
	result = metricExists(name, insightValue, metricValues)
	assert.False(t, result)
}

func TestMatchMetrics(t *testing.T) {
	prev := Metric{
		Resource:     Resource{Name: "vm", Group: "group", ID: "id"},
		Namespace:    "namespace",
		Names:        []string{"TotalRequests,Capacity"},
		Aggregations: "Average,Total",
		Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
		Values:       nil,
		TimeGrain:    "1PM",
	}
	current := Metric{
		Resource:     Resource{Name: "vm", Group: "group", ID: "id"},
		Namespace:    "namespace",
		Names:        []string{"TotalRequests,Capacity"},
		Aggregations: "Average,Total",
		Dimensions:   []Dimension{{Name: "location", Value: "West Europe"}},
		Values:       []MetricValue{},
		TimeGrain:    "1PM",
	}
	result := matchMetrics(prev, current)
	assert.True(t, result)
	current.Resource.ID = "id1"
	result = matchMetrics(prev, current)
	assert.False(t, result)
}

func TestMetricIsEmpty(t *testing.T) {
	fl := 12.4
	location := time.Location{}
	stamp := date.Time{
		Time: time.Date(2019, 12, 12, 12, 12, 12, 12, &location),
	}
	insightValue := insights.MetricValue{
		TimeStamp: &stamp,
		Average:   &fl,
		Minimum:   nil,
		Maximum:   nil,
		Total:     nil,
		Count:     nil,
	}
	result := metricIsEmpty(insightValue)
	assert.False(t, result)
	insightValue.Average = nil
	result = metricIsEmpty(insightValue)
	assert.True(t, result)
}

func TestMapResourceGroupFormID(t *testing.T) {
	path := "subscriptions/qw3e45r6t-23ws-1234-6587-1234ed4532/resourceGroups/obs-infrastructure/providers/Microsoft.Compute/virtualMachines/obstestmemleak"
	group := getResourceGroupFormID(path)
	assert.Equal(t, group, "obs-infrastructure")
}

func TestExpired(t *testing.T) {
	resConfig := ResourceConfiguration{}
	result := resConfig.Expired()
	assert.True(t, result)
}

func TestStringInSlice(t *testing.T) {
	s := "test"
	exists := []string{"hello", "test", "goodbye"}
	noExists := []string{"hello", "goodbye", "notest"}
	result := StringInSlice(s, exists)
	assert.True(t, result)
	result = StringInSlice(s, noExists)
	assert.False(t, result)
}
