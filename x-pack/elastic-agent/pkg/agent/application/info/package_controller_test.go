// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
)

func TestPackageController(t *testing.T) {
	testFn := func(pc *PackageController, input map[string]interface{}, expected ...string) {
		cfg, err := config.NewConfigFrom(input)
		require.NoError(t, err)

		err = pc.Reload(cfg)
		require.NoError(t, err)

		pp := pc.Packages()
		require.Equal(t, len(expected), len(pp))
		for i, expVal := range expected {
			require.Equal(t, expVal, pp[i])
		}
	}

	pc := newPackageController()
	require.Equal(t, 0, len(pc.Packages()))

	// init with single
	singlePackageMap := map[string]interface{}{
		"inputs": []interface{}{
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "single1"},
				},
			},
		},
	}
	testFn(pc, singlePackageMap, "single1")

	// rewrite single
	singlePackageMap = map[string]interface{}{
		"inputs": []interface{}{
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "single2"},
				},
			},
		},
	}
	testFn(pc, singlePackageMap, "single2")

	// more inputs are sorted no dups
	singlePackageMap = map[string]interface{}{
		"inputs": []interface{}{
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "triple1"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "double1"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "zeroToOne"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "double2"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "zeroToOne"},
				},
			},
		},
	}
	testFn(pc, singlePackageMap, "double1", "double2", "triple1", "zeroToOne")

	// spaces
	singlePackageMap = map[string]interface{}{
		"inputs": []interface{}{
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "singel1"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"version": "1.2.3"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "   "},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": " singel1"},
				},
			},
			map[string]interface{}{
				"meta": map[string]interface{}{
					"package": map[string]interface{}{
						"name": "singel1 "},
				},
			},
		},
	}
	testFn(pc, singlePackageMap, "singel1")
}
