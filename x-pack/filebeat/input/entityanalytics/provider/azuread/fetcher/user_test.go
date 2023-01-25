// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/elastic/beats/v7/x-pack/filebeat/input/entityanalytics/internal/collections"
)

func TestUser_Merge(t *testing.T) {
	tests := map[string]struct {
		In      *User
		InOther *User
		Want    *User
	}{
		"id-mismatch": {
			In:      &User{ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd")},
			InOther: &User{ID: uuid.MustParse("80c3f9af-75ae-45f5-b22b-53f005d5880d")},
			Want:    &User{ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd")},
		},
		"ok": {
			In: &User{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"a": "alpha",
				},
				MemberOf:           collections.NewSet[uuid.UUID](uuid.MustParse("fcda226a-c920-4d99-81bc-d2d691a6c212")),
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InOther: &User{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"b": "beta",
				},
				MemberOf:           collections.NewSet[uuid.UUID](uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")),
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("c550d32c-09b2-4851-b0f2-1bc431e26d01")),
			},
			Want: &User{
				ID: uuid.MustParse("187f924c-e867-477e-8d74-dd762d6379dd"),
				Fields: map[string]interface{}{
					"a": "alpha",
					"b": "beta",
				},
				MemberOf: collections.NewSet[uuid.UUID](
					uuid.MustParse("fcda226a-c920-4d99-81bc-d2d691a6c212"),
					uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
				),
				TransitiveMemberOf: collections.NewSet[uuid.UUID](
					uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
					uuid.MustParse("c550d32c-09b2-4851-b0f2-1bc431e26d01"),
				),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.In.Merge(tc.InOther)

			assert.Equal(t, tc.Want.ID, tc.In.ID)
			assert.Equal(t, tc.Want.Fields, tc.In.Fields)
			assert.ElementsMatch(t, tc.Want.MemberOf.Values(), tc.In.MemberOf.Values(), "list A: Expected, listB: Actual")
			assert.ElementsMatch(t, tc.Want.TransitiveMemberOf.Values(), tc.In.TransitiveMemberOf.Values(), "list A: Expected, listB: Actual")
		})
	}
}

func TestUser_IsMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    bool
	}{
		"nil": {
			InUser:  &User{},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want:    false,
		},
		"match": {
			InUser: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want:    true,
		},
		"mismatch": {
			InUser: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
			Want:    false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.InUser.IsMemberOf(tc.InValue)

			assert.Equal(t, tc.Want, got)
		})
	}
}

func TestUser_AddMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    *User
	}{
		"nil": {
			InUser:  &User{},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
		},
		"match": {
			InUser: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
		},
		"mismatch": {
			InUser: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				MemberOf: collections.NewSet[uuid.UUID](
					uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
					uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
				),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.InUser.AddMemberOf(tc.InValue)

			assert.ElementsMatch(t, tc.Want.MemberOf.Values(), tc.InUser.MemberOf.Values(), "list A: Expected, listB: Actual")
		})
	}
}

func TestUser_RemoveMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    *User
	}{
		"match": {
			InUser: &User{
				MemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				MemberOf: collections.NewSet[uuid.UUID](),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.InUser.RemoveMemberOf(tc.InValue)

			assert.ElementsMatch(t, tc.Want.MemberOf.Values(), tc.InUser.MemberOf.Values(), "list A: Expected, listB: Actual")
		})
	}
}

func TestUser_IsTransitiveMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    bool
	}{
		"nil": {
			InUser:  &User{},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want:    false,
		},
		"match": {
			InUser: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want:    true,
		},
		"mismatch": {
			InUser: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
			Want:    false,
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := tc.InUser.IsTransitiveMemberOf(tc.InValue)

			assert.Equal(t, tc.Want, got)
		})
	}
}

func TestUser_AddTransitiveMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    *User
	}{
		"nil": {
			InUser:  &User{},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
		},
		"match": {
			InUser: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
		},
		"mismatch": {
			InUser: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](
					uuid.MustParse("a77e8cbb-27a5-49d3-9d5e-801997621f87"),
					uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
				),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.InUser.AddTransitiveMemberOf(tc.InValue)

			assert.ElementsMatch(t, tc.Want.TransitiveMemberOf.Values(), tc.InUser.TransitiveMemberOf.Values(), "list A: Expected, listB: Actual")
		})
	}
}

func TestUser_RemoveTransitiveMemberOf(t *testing.T) {
	tests := map[string]struct {
		InUser  *User
		InValue uuid.UUID
		Want    *User
	}{
		"match": {
			InUser: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28")),
			},
			InValue: uuid.MustParse("ca777ad5-9abf-4c9b-be1f-c38c6ec28f28"),
			Want: &User{
				TransitiveMemberOf: collections.NewSet[uuid.UUID](),
			},
		},
	}

	for name, tc := range tests {
		tc := tc
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			tc.InUser.RemoveTransitiveMemberOf(tc.InValue)

			assert.ElementsMatch(t, tc.Want.TransitiveMemberOf.Values(), tc.InUser.TransitiveMemberOf.Values(), "list A: Expected, listB: Actual")
		})
	}
}
