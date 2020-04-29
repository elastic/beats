// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

package netflow

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCamelCaseToSnakeCase(t *testing.T) {
	for _, testCase := range [][2]string{
		{"aBCDe", "a_bc_de"},
		{"postNATSourceIPv4Address", "post_nat_source_ipv4_address"},
		{"selectorIDTotalFlowsObserved", "selector_id_total_flows_observed"},
		{"engineId", "engine_id"},
		{"samplerRandomInterval", "sampler_random_interval"},
		{"dot1qVlanId", "dot1q_vlan_id"},
		{"messageMD5Checksum", "message_md5_checksum"},
		{"hashIPPayloadSize", "hash_ip_payload_size"},
		{"upperCILimit", "upper_ci_limit"},
		{"virtualStationUUID", "virtual_station_uuid"},
		{"selectorIDTotalFlowsObserved", "selector_id_total_flows_observed"},
		{"postMCastLayer2OctetDeltaCount", "post_mcast_layer2_octet_delta_count"},
		{"IPSecSPI", "ip_sec_spi"},
		{"VRFname", "vrf_name"},
	} {
		s, found := fieldNameConverter.conversion[testCase[0]]
		if !found {
			s = CamelCaseToSnakeCase(testCase[0])
		}
		assert.Equal(t, testCase[1], s)
	}
}
