package jobs

import (
	"testing"

	"github.com/elastic/beats/libbeat/beat"
)

func ExecJobsAndConts(t *testing.T, jobs []Job) ([]*beat.Event, error) {
	var results []*beat.Event
	for _, j := range jobs {
		resultEvents, err := ExecJobAndConts(t, j)
		if err != nil {
			return nil, err
		}
		for _, re := range resultEvents {
			results = append(results, re)
		}
	}

	return results, nil
}

// Helper to recursively execute a job and gather its results
func ExecJobAndConts(t *testing.T, j Job) ([]*beat.Event, error) {
	var results []*beat.Event
	event := &beat.Event{}
	results = append(results, event)
	cont, err := j(event)
	if err != nil {
		return nil, err
	}

	for _, cj := range cont {
		cjResults, err := ExecJobAndConts(t, cj)
		if err != nil {
			return nil, err
		}
		for _, cjResults := range cjResults {
			results = append(results, cjResults)
		}
	}

	return results, nil
}
