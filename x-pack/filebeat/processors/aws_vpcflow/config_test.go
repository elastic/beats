// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package aws_vpcflow

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"

	conf "github.com/elastic/elastic-agent-libs/config"
)

func TestConfigUnpack(t *testing.T) {
	testCases := []struct {
		yamlConfig string
		error      bool
	}{
		{
			yamlConfig: `
---
mode: ecs_and_original
id: us-east-vpcflow
format: instance-id interface-id srcaddr dstaddr pkt-srcaddr pkt-dstaddr
`,
		},
		{
			yamlConfig: `
---
mode: original
format: version interface-id account-id vpc-id subnet-id instance-id srcaddr dstaddr srcport dstport protocol tcp-flags type pkt-srcaddr pkt-dstaddr action log-status
`,
		},
		{
			yamlConfig: `
---
mode: ecs
format: version srcaddr dstaddr srcport dstport protocol start end type packets bytes account-id vpc-id subnet-id instance-id interface-id region az-id sublocation-type sublocation-id action tcp-flags pkt-srcaddr pkt-dstaddr pkt-src-aws-service pkt-dst-aws-service traffic-path flow-direction log-status
`,
		},
		{
			yamlConfig: `
---
mode: ecs
format: version srcaddr dstaddr srcport dstport protocol start end type packets bytes account-id vpc-id subnet-id instance-id interface-id region az-id sublocation-type sublocation-id action tcp-flags pkt-srcaddr pkt-dstaddr pkt-src-aws-service pkt-dst-aws-service traffic-path flow-direction log-status
`,
		},
		{
			error: true,
			yamlConfig: `
---
mode: invalid
format: version
`,
		},
	}

	for i, tc := range testCases {
		tc := tc
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			rawConfig := conf.MustNewConfigFrom(tc.yamlConfig)

			c := defaultConfig()
			err := rawConfig.Unpack(&c)
			if tc.error {
				t.Log("Error:", err)
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
