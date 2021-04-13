// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package info

import (
	"testing"

	"github.com/elastic/beats/v7/x-pack/elastic-agent/pkg/config"
	"github.com/stretchr/testify/require"
)

func TestPackageController(t *testing.T) {
	pc := newPackageController()

	pp := pc.Packages()
	require.Equal(t, 0, len(pp))

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

	cfg, err := config.NewConfigFrom(singlePackageMap)
	require.NoError(t, err)

	err = pc.Reload(cfg)
	require.NoError(t, err)

	pp = pc.Packages()
	require.Equal(t, 1, len(pp))
	require.Equal(t, "single1", pp[0])

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

	cfg, err = config.NewConfigFrom(singlePackageMap)
	require.NoError(t, err)

	err = pc.Reload(cfg)
	require.NoError(t, err)

	pp = pc.Packages()
	require.Equal(t, 1, len(pp))
	require.Equal(t, "single2", pp[0])

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

	cfg, err = config.NewConfigFrom(singlePackageMap)
	require.NoError(t, err)

	err = pc.Reload(cfg)
	require.NoError(t, err)

	pp = pc.Packages()
	require.Equal(t, 4, len(pp))
	require.Equal(t, "double1", pp[0])
	require.Equal(t, "double2", pp[1])
	require.Equal(t, "triple1", pp[2])
	require.Equal(t, "zeroToOne", pp[3])

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

	cfg, err = config.NewConfigFrom(singlePackageMap)
	require.NoError(t, err)

	err = pc.Reload(cfg)
	require.NoError(t, err)

	pp = pc.Packages()
	require.Equal(t, 1, len(pp))
	require.Equal(t, "singel1", pp[0])
}
