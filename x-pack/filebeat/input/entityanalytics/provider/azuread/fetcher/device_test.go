// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
)

func TestDevice_Merge(t *testing.T) {
	tests := map[string]struct {
		In      *Device
		InOther *Device
		Want    *Device
	}{
		"id-mismatch": {
			In:      &Device{ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd")},
			InOther: &Device{ID: uuid.MustParse("80c3f9af-75ae-45f5-b22b-53f005d5880d")},
			Want:    &Device{ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd")},
		},
		"ok": {
			In: &Device{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"a": "alpha",
				},
				MemberOf:           collections.NewUUIDSet(uuid.MustParse("fcda226a-c920-4d99-81bc-d2d691a6c212")),
				TransitiveMemberOf: collections.NewUUIDSet(uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
				RegisteredOwners:   collections.NewUUIDSet(uuid.MustParse("c59fbdb8-e442-46b1-8d72-c8ac0b78ec0a")),
				RegisteredUsers: collections.NewUUIDSet(
					uuid.MustParse("27cea005-7377-4175-b2ef-e9d64c977f4d"),
					uuid.MustParse("c59fbdb8-e442-46b1-8d72-c8ac0b78ec0a"),
				),
			},
			InOther: &Device{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"b": "beta",
				},
				MemberOf:           collections.NewUUIDSet(uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")),
				TransitiveMemberOf: collections.NewUUIDSet(uuid.MustParse("c550d32c-09b2-4851-b0f2-1bc431e26d01")),
				RegisteredOwners:   collections.NewUUIDSet(uuid.MustParse("81d1b5cd-7cd6-469d-9fe8-0a5c6cf2a7b6")),
				RegisteredUsers: collections.NewUUIDSet(
					uuid.MustParse("5e6d279a-ce2b-43b8-a38f-3110907e1974"),
					uuid.MustParse("c59fbdb8-e442-46b1-8d72-c8ac0b78ec0a"),
				),
			},
			Want: &Device{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"a": "alpha",
					"b": "beta",
				},
				MemberOf: collections.NewUUIDSet(
					uuid.MustParse("fcda226a-c920-4d99-81bc-d2d691a6c212"),
					uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
				),
				TransitiveMemberOf: collections.NewUUIDSet(
					uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
					uuid.MustParse("c550d32c-09b2-4851-b0f2-1bc431e26d01"),
				),
				RegisteredOwners: collections.NewUUIDSet(
					uuid.MustParse("81d1b5cd-7cd6-469d-9fe8-0a5c6cf2a7b6"),
					uuid.MustParse("c59fbdb8-e442-46b1-8d72-c8ac0b78ec0a"),
				),
				RegisteredUsers: collections.NewUUIDSet(
					uuid.MustParse("27cea005-7377-4175-b2ef-e9d64c977f4d"),
					uuid.MustParse("5e6d279a-ce2b-43b8-a38f-3110907e1974"),
					uuid.MustParse("c59fbdb8-e442-46b1-8d72-c8ac0b78ec0a"),
				),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.Merge(tc.InOther)

			require.Equal(t, tc.Want.ID, tc.In.ID)
			require.Equal(t, tc.Want.Fields, tc.In.Fields)
			require.ElementsMatch(t, tc.Want.MemberOf.Values(), tc.In.MemberOf.Values(), "list A: Expected, listB: Actual")
			require.ElementsMatch(t, tc.Want.TransitiveMemberOf.Values(), tc.In.TransitiveMemberOf.Values(), "list A: Expected, listB: Actual")
			require.ElementsMatch(t, tc.Want.RegisteredOwners.Values(), tc.In.RegisteredOwners.Values(), "list A: Expected, listB: Actual")
			require.ElementsMatch(t, tc.Want.RegisteredUsers.Values(), tc.In.RegisteredUsers.Values(), "list A: Expected, listB: Actual")
		})
	}
}
