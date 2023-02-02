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

const (
	testUUID1Str = "187f924c-e867-477e-8d74-dd762d6379dd"
	testUUID2Str = "80c3f9af-75ae-45f5-b22b-53f005d5880d"
	testUUID3Str = "ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"
	testUUID4Str = "ec8b17ae-ce9d-4099-97ee-4a959638bc29"
	testUUID5Str = "fcda226a-c920-4d99-81bc-d2d691a6c212"
)

var (
	testUUID1 = uuid.MustParse(testUUID1Str)
	testUUID2 = uuid.MustParse(testUUID2Str)
	testUUID3 = uuid.MustParse(testUUID3Str)
	testUUID4 = uuid.MustParse(testUUID4Str)
	testUUID5 = uuid.MustParse(testUUID5Str)
)

func TestNewUUIDSet(t *testing.T) {
	tests := map[string]struct {
		In   []uuid.UUID
		Want UUIDSet
	}{
		"nil": {
			In:   nil,
			Want: UUIDSet{},
		},
		"empty": {
			In:   []uuid.UUID{},
			Want: UUIDSet{},
		},
		"testUUID1-elem": {
			In:   []uuid.UUID{testUUID1},
			Want: UUIDSet{m: map[uuid.UUID]struct{}{testUUID1: {}}},
		},
		"testUUID3-elem": {
			In:   []uuid.UUID{testUUID1, testUUID2, testUUID3},
			Want: UUIDSet{m: map[uuid.UUID]struct{}{testUUID1: {}, testUUID2: {}, testUUID3: {}}},
		},
		"dup-elem": {
			In:   []uuid.UUID{testUUID1, testUUID2, testUUID2},
			Want: UUIDSet{m: map[uuid.UUID]struct{}{testUUID1: {}, testUUID2: {}}},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := NewUUIDSet(tc.In...)
			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDSet_UnmarshalJSON(t *testing.T) {
	tests := map[string]struct {
		In      []byte
		Want    UUIDSet
		WantErr string
	}{
		"ok-nil": {
			In:   []byte(`null`),
			Want: NewUUIDSet(),
		},
		"ok-empty": {
			In:   []byte(`[]`),
			Want: NewUUIDSet(),
		},
		"ok-testUUID1-elem": {
			In:   []byte(fmt.Sprintf(`["%s"]`, testUUID1Str)),
			Want: NewUUIDSet(testUUID1),
		},
		"ok-testUUID3-elem": {
			In:   []byte(fmt.Sprintf(`["%s", "%s", "%s"]`, testUUID1Str, testUUID2Str, testUUID3Str)),
			Want: NewUUIDSet(testUUID1, testUUID2, testUUID3),
		},
		"ok-dup-elem": {
			In:   []byte(fmt.Sprintf(`["%s","%s","%s"]`, testUUID1Str, testUUID2Str, testUUID2Str)),
			Want: NewUUIDSet(testUUID1, testUUID2),
		},
		"err-mixed-types": {
			In:      []byte(fmt.Sprintf(`["%s",0]`, testUUID1Str)),
			WantErr: "json: cannot unmarshal number into Go value of type uuid.UUID",
		},
		"err-not-list-str": {
			In:      []byte(fmt.Sprintf(`"%s"`, testUUID1Str)),
			WantErr: "json: cannot unmarshal string into Go value of type []uuid.UUID",
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var s UUIDSet

			gotErr := json.Unmarshal(tc.In, &s)

			if tc.WantErr != "" {
				require.ErrorContains(t, gotErr, tc.WantErr)
			} else {
				require.NoError(t, gotErr)
				require.Equal(t, tc.Want, s)
			}
		})
	}
}

func TestUUIDSet_MarshalJSON(t *testing.T) {
	tests := map[string]struct {
		In      UUIDSet
		Want    []byte
		WantErr string
	}{
		"ok-empty": {
			In:   NewUUIDSet(),
			Want: []byte("null"),
		},
		"ok-testUUID1-elem": {
			In:   NewUUIDSet(testUUID1),
			Want: []byte(fmt.Sprintf(`["%s"]`, testUUID1Str)),
		},
		"ok-testUUID3-elem": {
			In:   NewUUIDSet(testUUID1, testUUID2, testUUID3),
			Want: []byte(fmt.Sprintf(`["%s","%s","%s"]`, testUUID1Str, testUUID2Str, testUUID3Str)),
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

func TestUUIDSet_Len(t *testing.T) {
	tests := map[string]struct {
		In   UUIDSet
		Want int
	}{
		"empty": {
			In:   NewUUIDSet(),
			Want: 0,
		},
		"elements": {
			In:   NewUUIDSet(testUUID1, testUUID2, testUUID3),
			Want: 3,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.Len()
			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDSet_Values(t *testing.T) {
	tests := map[string]struct {
		In   UUIDSet
		Want []uuid.UUID
	}{
		"empty": {
			In:   NewUUIDSet(),
			Want: nil,
		},
		"elements": {
			In:   NewUUIDSet(testUUID1, testUUID2, testUUID3),
			Want: []uuid.UUID{testUUID1, testUUID2, testUUID3},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.Values()
			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDSet_Add(t *testing.T) {
	tests := map[string]struct {
		In    []uuid.UUID
		InSet UUIDSet
		Want  UUIDSet
	}{
		"empty": {
			In:    nil,
			InSet: NewUUIDSet(),
			Want:  NewUUIDSet(),
		},
		"elements": {
			In:    []uuid.UUID{testUUID1, testUUID2, testUUID3},
			InSet: NewUUIDSet(),
			Want:  NewUUIDSet(testUUID1, testUUID2, testUUID3),
		},
		"dup-elements": {
			In:    []uuid.UUID{testUUID1, testUUID2, testUUID2},
			InSet: NewUUIDSet(),
			Want:  NewUUIDSet(testUUID1, testUUID2),
		},
		"existing-elements": {
			In:    []uuid.UUID{testUUID1, testUUID2},
			InSet: NewUUIDSet(testUUID1, testUUID2),
			Want:  NewUUIDSet(testUUID1, testUUID2),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.InSet.Add(tc.In...)

			require.Equal(t, tc.Want, tc.InSet)
		})
	}
}

func TestUUIDSet_Remove(t *testing.T) {
	tests := map[string]struct {
		In       UUIDSet
		InValues []uuid.UUID
		Want     UUIDSet
	}{
		"empty": {
			In:       NewUUIDSet(),
			InValues: []uuid.UUID{testUUID1},
			Want:     NewUUIDSet(),
		},
		"elements": {
			In:       NewUUIDSet(testUUID1, testUUID2, testUUID3),
			InValues: []uuid.UUID{testUUID1, testUUID2, testUUID3},
			Want:     NewUUIDSet(),
		},
		"elements-mix": {
			In:       NewUUIDSet(testUUID1),
			InValues: []uuid.UUID{testUUID1, testUUID2, testUUID3},
			Want:     NewUUIDSet(),
		},
		"elements-incomplete": {
			In:       NewUUIDSet(testUUID1, testUUID2, testUUID3),
			InValues: []uuid.UUID{testUUID1},
			Want:     NewUUIDSet(testUUID2, testUUID3),
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.Remove(tc.InValues...)
			require.Equal(t, tc.Want, tc.In)
		})
	}
}

func TestUUIDSet_Contains(t *testing.T) {
	tests := map[string]struct {
		In      UUIDSet
		InValue uuid.UUID
		Want    bool
	}{
		"true": {
			In:      NewUUIDSet(testUUID1),
			InValue: testUUID1,
			Want:    true,
		},
		"false-no-match": {
			In:      NewUUIDSet(testUUID2),
			InValue: testUUID1,
			Want:    false,
		},
		"false-empty": {
			In:      NewUUIDSet(),
			InValue: testUUID1,
			Want:    false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.In.Contains(tc.InValue)

			require.Equal(t, tc.Want, got)
		})
	}
}

func TestUUIDSet_ForEach(t *testing.T) {
	t.Run("elements", func(t *testing.T) {
		t.Parallel()

		expected := []uuid.UUID{testUUID1, testUUID2, testUUID3}
		var got []uuid.UUID

		s := NewUUIDSet(expected...)
		s.ForEach(func(elem uuid.UUID) {
			got = append(got, elem)
		})

		require.ElementsMatch(t, expected, got)
	})

	t.Run("empty", func(t *testing.T) {
		t.Parallel()

		called := 0
		s := NewUUIDSet()
		s.ForEach(func(elem uuid.UUID) {
			called++
		})

		require.Zero(t, called)
	})
}
