// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package registrar

import (
	"github.com/elastic/beats/v7/filebeat/input/file"
	"github.com/elastic/beats/v7/libbeat/registry"
)

func readStatesFrom(store *registry.Store) ([]file.State, error) {
	var states []file.State

	err := store.View(func(tx *registry.Tx) error {
		return tx.Each(func(k registry.Key, v registry.ValueDecoder) (bool, error) {
			var st file.State

			// try to decode. Ingore faulty/incompatible values.
			if err := v.Decode(&st); err != nil {
				// XXX: Do we want to log here? In case we start to store other
				// state types in the registry, then this operation will likely fail
				// quite often, producing some false-positives in the logs...
				return true, nil
			}

			st.Id = string(k)
			states = append(states, st)
			return true, nil
		})
	})
	if err != nil {
		return nil, err
	}

	states = resetStates(fixStates(states))
	return states, nil
}

// fixStates cleans up the registry states when updating from an older version
// of filebeat potentially writing invalid entries.
func fixStates(states []file.State) []file.State {
	if len(states) == 0 {
		return states
	}

	// we use a map of states here, so to identify and merge duplicate entries.
	idx := map[string]*file.State{}
	for i := range states {
		state := &states[i]
		fixState(state)

		id := state.ID()
		old, exists := idx[id]
		if !exists {
			idx[id] = state
		} else {
			mergeStates(old, state) // overwrite the entry in 'old'
		}
	}

	if len(idx) == len(states) {
		return states
	}

	i := 0
	newStates := make([]file.State, len(idx))
	for _, state := range idx {
		newStates[i] = *state
		i++
	}
	return newStates
}

// fixState updates a read state to fullfil required invariantes:
// - "Meta" must be nil if len(Meta) == 0
func fixState(st *file.State) {
	if len(st.Meta) == 0 {
		st.Meta = nil
	}
}

// mergeStates merges 2 states by trying to determine the 'newer' state.
// The st state is overwritten with the updated fields.
func mergeStates(st, other *file.State) {
	st.Finished = st.Finished || other.Finished
	if st.Offset < other.Offset { // always select the higher offset
		st.Offset = other.Offset
	}

	// update file meta-data. As these are updated concurrently by the
	// inputs, select the newer state based on the update timestamp.
	var meta, metaOld, metaNew map[string]string
	if st.Timestamp.Before(other.Timestamp) {
		st.Source = other.Source
		st.Timestamp = other.Timestamp
		st.TTL = other.TTL
		st.FileStateOS = other.FileStateOS

		metaOld, metaNew = st.Meta, other.Meta
	} else {
		metaOld, metaNew = other.Meta, st.Meta
	}

	if len(metaOld) == 0 || len(metaNew) == 0 {
		meta = metaNew
	} else {
		meta = map[string]string{}
		for k, v := range metaOld {
			meta[k] = v
		}
		for k, v := range metaNew {
			meta[k] = v
		}
	}

	if len(meta) == 0 {
		meta = nil
	}
	st.Meta = meta
}

// resetStates sets all states to finished and disable TTL on restart
// For all states covered by an input, TTL will be overwritten with the input value
func resetStates(states []file.State) []file.State {
	for key, state := range states {
		state.Finished = true
		// Set ttl to -2 to easily spot which states are not managed by a input
		state.TTL = -2
		states[key] = state
	}
	return states
}
