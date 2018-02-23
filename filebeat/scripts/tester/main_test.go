package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPipelinePath(t *testing.T) {
	testCases := []struct {
		pipelinePath string
		count        int
	}{
		{
			pipelinePath: "../../module/postgresql/log/ingest/pipeline.json",
			count:        1,
		},
		{
			pipelinePath: "../../module/postgresql/log/ingest",
			count:        1,
		},
		{
			pipelinePath: "postgresql/log",
			count:        1,
		},
	}

	for _, tc := range testCases {
		paths, err := getPipelinePath(tc.pipelinePath, "../../module")
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, tc.count, len(paths))
	}

	testCasesError := []string{
		"non-such-pipeline.json",
		"no/such/path/to/pipeline",
		"not/module",
	}
	for _, p := range testCasesError {
		paths, err := getPipelinePath(p, "./module")
		if err == nil {
			t.Fatal(paths)
		}
		assert.Equal(t, 0, len(paths))
	}
}
