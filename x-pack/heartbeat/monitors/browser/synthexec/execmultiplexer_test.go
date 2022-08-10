// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package synthexec

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestExecMultiplexer(t *testing.T) {
	em := NewExecMultiplexer()

	// Generate three fake journeys with three fake steps
	var testJourneys []*Journey
	var testEvents []*SynthEvent
	time := float64(0)
	for jIdx := 0; jIdx < 3; jIdx++ {
		time++ // fake time to make events seem spaced out
		journey := &Journey{
			Name: fmt.Sprintf("J%d", jIdx),
			Id:   fmt.Sprintf("j-%d", jIdx),
		}
		testJourneys = append(testJourneys, journey)
		testEvents = append(testEvents, &SynthEvent{
			Journey:              journey,
			Type:                 "journey/start",
			TimestampEpochMicros: time,
		})

		for sIdx := 0; sIdx < 3; sIdx++ {
			step := &Step{
				Name:   fmt.Sprintf("S%d", sIdx),
				Index:  sIdx,
				Status: "failed",
			}

			testEvents = append(testEvents, &SynthEvent{
				Journey:              journey,
				Step:                 step,
				TimestampEpochMicros: time,
			})
		}
		testEvents = append(testEvents, &SynthEvent{
			Journey:              journey,
			Type:                 "journey/end",
			TimestampEpochMicros: time,
		})
	}

	// Write the test events in another go routine since writes block
	var results []*SynthEvent
	go func() {
		for _, se := range testEvents {
			em.writeSynthEvent(se)
		}
		em.Close()
	}()

	// Wait for all results
Loop:
	for {
		select {
		case result := <-em.synthEvents:
			if result == nil {
				break Loop
			}
			results = append(results, result)
		}
	}

	require.Len(t, results, len(testEvents))
	i := 0 // counter for index, resets on journey change
	for _, se := range results {
		require.Equal(t, i, se.index)
		if se.Type == "journey/end" {
			i = 0
		} else {
			i++
		}
	}
}
