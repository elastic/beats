package pandora

import (
	"testing"
	"time"

	"github.com/elastic/beats/libbeat/common"
	psdk "qiniu.com/pandora/pipeline"
)

func sliceToMap(points []psdk.PointField) map[string]string {
	m := map[string]string{}
	for _, p := range points {
		var actualV string
		switch p.Value.(type) {
		case string:
			actualV = p.Value.(string)
		case *string:
			actualV = *p.Value.(*string)
		}
		m[p.Key] = actualV
	}

	return m
}

func compare(expected map[string]string, actual map[string]string, t *testing.T) {
	if len(expected) != len(actual) {
		t.Error("len(expected) = ", len(expected), ", len(actual) = ", len(actual))
		return
	}
	for k, v := range expected {
		if actual[k] != v {
			t.Error("expected = ", v, ", actual = ", actual[k])
			return
		}
	}
}

func Test_mapStrToSlice(t *testing.T) {
	logTime := time.Now()
	logMessage := "this is a test log"
	hostname := "xs66"
	// stdout
	mapStr := map[string]interface{}{
		"type":       "stdout",
		"message":    logMessage,
		"@timestamp": common.Time(logTime),
		"source":     "/disk1/mesos/slaves/964ead97-8786-4e59-acf9-df3ecc19ee00-S16/frameworks/964ead97-8786-4e59-acf9-df3ecc19ee00-0000/executors/image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048/runs/latest/stdout",
	}

	expectedSchema := map[string]string{
		"source":      "stdout",
		"message":     logMessage,
		"timestamp":   logTime.Format(time.RFC3339),
		"app":         "image",
		"hostname":    hostname,
		"launch_id":   "add97e1f-80ba-11e6-ba0c-6c92bf2f06d8",
		"instance_id": "image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048",
	}

	points := mapStrToSlice(hostname, common.MapStr(mapStr))
	schema := sliceToMap(points)
	compare(expectedSchema, schema, t)

	// stderr
	mapStr = map[string]interface{}{
		"type":       "stderr",
		"message":    logMessage,
		"@timestamp": common.Time(logTime),
		"source":     "/disk1/mesos/slaves/964ead97-8786-4e59-acf9-df3ecc19ee00-S16/frameworks/964ead97-8786-4e59-acf9-df3ecc19ee00-0000/executors/image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048/runs/latest/stderr",
	}

	expectedSchema = map[string]string{
		"source":      "stderr",
		"message":     logMessage,
		"timestamp":   logTime.Format(time.RFC3339),
		"app":         "image",
		"hostname":    hostname,
		"launch_id":   "add97e1f-80ba-11e6-ba0c-6c92bf2f06d8",
		"instance_id": "image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048",
	}

	points = mapStrToSlice(hostname, common.MapStr(mapStr))
	schema = sliceToMap(points)
	compare(expectedSchema, schema, t)

	// user logs
	mapStr = map[string]interface{}{
		"type":       "sandbox",
		"message":    logMessage,
		"@timestamp": common.Time(logTime),
		"source":     "/disk1/mesos/slaves/964ead97-8786-4e59-acf9-df3ecc19ee00-S16/frameworks/964ead97-8786-4e59-acf9-df3ecc19ee00-0000/executors/image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048/runs/latest/log/0/1",
	}

	expectedSchema = map[string]string{
		"source":      "0",
		"message":     logMessage,
		"timestamp":   logTime.Format(time.RFC3339),
		"app":         "image",
		"hostname":    hostname,
		"launch_id":   "add97e1f-80ba-11e6-ba0c-6c92bf2f06d8",
		"instance_id": "image.add97e1f-80ba-11e6-ba0c-6c92bf2f06d8.1474876587454367048",
	}

	points = mapStrToSlice(hostname, common.MapStr(mapStr))
	schema = sliceToMap(points)
	compare(expectedSchema, schema, t)
}
