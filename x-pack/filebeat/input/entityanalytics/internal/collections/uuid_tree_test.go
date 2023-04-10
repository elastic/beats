// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package collections

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestUUIDTree_UnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		In      []byte
		Want    UUIDTree
		WantErr string
	}{
		"ok": {
			In: []byte(fmt.Sprintf(`{"%s":["%s","%s"],"%s":["%s"]}`, testUUID1Str, testUUID2Str, testUUID3Str, testUUID4Str, testUUID5Str)),
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID3: {}}},
				testUUID4: {m: map[uuid.UUID]struct{}{testUUID5: {}}},
			}},
		},
		"nil": {
			In:   []byte("null"),
			Want: UUIDTree{},
		},
		"empty": {
			In:   []byte("{}"),
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{}},
		},
		"err-bad-uuid-key": {
			In:      []byte(fmt.Sprintf(`{"1":["%s"]}`, testUUID1)),
			WantErr: "invalid UUID length: 1",
		},
		"err-bad-uuid-set": {
			In:      []byte(fmt.Sprintf(`{"%s":[1]}`, testUUID1)),
			WantErr: "json: cannot unmarshal number into Go value of type uuid.UUID",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var tree UUIDTree

			gotErr := json.Unmarshal(tc.In, &tree)

			if tc.WantErr != "" {
				require.ErrorContains(t, gotErr, tc.WantErr)
			} else {
				require.NoError(t, gotErr)
				require.Equal(t, tc.Want, tree)
			}
		})
	}
}

func TestUUIDTree_MarshalJSON(t *testing.T) {
	tests := map[string]struct {
		In      UUIDTree
		Want    []byte
		WantErr string
	}{
		"ok-empty": {
			In:   UUIDTree{},
			Want: []byte("null"),
		},
		"ok-elements": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID3: {}}},
				testUUID4: {m: map[uuid.UUID]struct{}{testUUID5: {}}},
			}},
			Want: []byte(fmt.Sprintf(`{"%s":["%s","%s"],"%s":["%s"]}`, testUUID1Str, testUUID2Str, testUUID3Str, testUUID4Str, testUUID5Str)),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got, gotErr := json.Marshal(&tc.In)

			if tc.WantErr != "" {
				require.ErrorContains(t, gotErr, tc.WantErr)
			} else {
				require.NoError(t, gotErr)
				require.Equal(t, tc.Want, got)
			}
		})
	}
}

func TestUUIDTree_RemoveVertex(t *testing.T) {
	tests := map[string]struct {
		In      UUIDTree
		InValue uuid.UUID
		Want    UUIDTree
	}{
		"empty": {
			In:      UUIDTree{},
			InValue: testUUID1,
			Want:    UUIDTree{},
		},
		"ok-elements": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID5: {}}},
				testUUID4: {m: map[uuid.UUID]struct{}{testUUID1: {}}},
			}},
			InValue: testUUID1,
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID5: {}}},
			}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.RemoveVertex(tc.InValue)

			require.Equal(t, tc.Want, tc.In)
		})
	}
}

func TestUUIDTree_ContainsVertex(t *testing.T) {
	tests := map[string]struct {
		In      UUIDTree
		InValue uuid.UUID
		Want    bool
	}{
		"empty": {
			In:      UUIDTree{},
			InValue: testUUID1,
			Want:    false,
		},
		"match": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID5: {}}},
				testUUID4: {m: map[uuid.UUID]struct{}{testUUID1: {}}},
			}},
			InValue: testUUID1,
			Want:    true,
		},
		"no-match": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InValue: testUUID3,
			Want:    false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.ContainsVertex(tc.InValue)

			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDTree_AddEdge(t *testing.T) {
	tests := map[string]struct {
		In     UUIDTree
		InFrom uuid.UUID
		InTo   []uuid.UUID
		Want   UUIDTree
	}{
		"empty": {
			In:     UUIDTree{},
			InFrom: testUUID1,
			InTo:   []uuid.UUID{testUUID2},
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
		},
		"exists": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InFrom: testUUID1,
			InTo:   []uuid.UUID{testUUID2},
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
		},
		"add-to-existing": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InFrom: testUUID1,
			InTo:   []uuid.UUID{testUUID3},
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID3: {}}},
			}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.AddEdge(tc.InFrom, tc.InTo...)

			require.Equal(t, tc.Want, tc.In)
		})
	}
}

func TestUUIDTree_RemoveEdge(t *testing.T) {
	tests := map[string]struct {
		In     UUIDTree
		InFrom uuid.UUID
		InTo   uuid.UUID
		Want   UUIDTree
	}{
		"empty": {
			In:     UUIDTree{},
			InFrom: testUUID1,
			InTo:   testUUID2,
			Want:   UUIDTree{},
		},
		"exists": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InFrom: testUUID1,
			InTo:   testUUID2,
			Want:   UUIDTree{edges: map[uuid.UUID]*UUIDSet{}},
		},
		"not-exists": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID4: {}}},
			}},
			InFrom: testUUID1,
			InTo:   testUUID2,
			Want: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID4: {}}},
			}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.RemoveEdge(tc.InFrom, tc.InTo)

			require.Equal(t, tc.Want, tc.In)
		})
	}
}

func TestUUIDTree_ContainsEdge(t *testing.T) {
	tests := map[string]struct {
		In     UUIDTree
		InFrom uuid.UUID
		InTo   uuid.UUID
		Want   bool
	}{
		"empty": {
			In:     UUIDTree{},
			InFrom: testUUID1,
			InTo:   testUUID2,
			Want:   false,
		},
		"match": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InFrom: testUUID1,
			InTo:   testUUID2,
			Want:   true,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.ContainsEdge(tc.InFrom, tc.InTo)

			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDTree_Expand(t *testing.T) {
	tests := map[string]struct {
		In       UUIDTree
		InValues []uuid.UUID
		Want     UUIDSet
	}{
		"empty": {
			In:       UUIDTree{},
			InValues: []uuid.UUID{testUUID1},
			Want:     NewUUIDSet(),
		},
		"elements": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID4: {}}},
				testUUID2: {m: map[uuid.UUID]struct{}{testUUID3: {}, testUUID5: {}}},
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InValues: []uuid.UUID{testUUID1},
			Want:     NewUUIDSet(testUUID1, testUUID2, testUUID3, testUUID4, testUUID5),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.Expand(tc.InValues...)

			require.Equal(t, tc.Want.Values(), got.Values())
		})
	}
}

func TestUUIDTree_ExpandFromSet(t *testing.T) {
	tests := map[string]struct {
		In       UUIDTree
		InValues UUIDSet
		Want     UUIDSet
	}{
		"empty": {
			In:       UUIDTree{},
			InValues: NewUUIDSet(testUUID1),
			Want:     NewUUIDSet(),
		},
		"elements": {
			In: UUIDTree{edges: map[uuid.UUID]*UUIDSet{
				testUUID1: {m: map[uuid.UUID]struct{}{testUUID2: {}, testUUID4: {}}},
				testUUID2: {m: map[uuid.UUID]struct{}{testUUID3: {}, testUUID5: {}}},
				testUUID3: {m: map[uuid.UUID]struct{}{testUUID2: {}}},
			}},
			InValues: NewUUIDSet(testUUID1),
			Want:     NewUUIDSet(testUUID1, testUUID2, testUUID3, testUUID4, testUUID5),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.ExpandFromSet(tc.InValues)

			require.Equal(t, tc.Want.Values(), got.Values())
		})
	}
}
