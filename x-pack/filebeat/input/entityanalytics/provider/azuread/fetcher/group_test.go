// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package fetcher

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestGroup_ToECS(t *testing.T) {
	in := Group{
		ID:   uuid.MustParse("88ecb4e8-5a1a-461e-a062-f1d3c5aa4ca4"),
		Name: "group1",
	}
	want := GroupECS{
		ID:   "88ecb4e8-5a1a-461e-a062-f1d3c5aa4ca4",
		Name: "group1",
	}

	got := in.ToECS()
	require.Equal(t, want, got)
}
