/*
© Copyright IBM Corporation 2018

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package ibmmqi

import (
	"testing"
)

// Tests for mqistr.go
func TestMqstrerror(t *testing.T) {
	output := mqstrerror("test", 0, 0)
	expected := "test: MQCC = MQCC_OK [0] MQRC = MQRC_NONE [0]"
	if output != expected {
		t.Logf("Gave 0, 0. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = mqstrerror("test", 1, 2393)
	expected = "test: MQCC = MQCC_WARNING [1] MQRC = MQRC_SSL_INITIALIZATION_ERROR [2393]"
	if output != expected {
		t.Logf("Gave 1, 2393. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = mqstrerror("test", 2, 2035)
	expected = "test: MQCC = MQCC_FAILED [2] MQRC = MQRC_NOT_AUTHORIZED [2035]"
	if output != expected {
		t.Logf("Gave 2, 2035. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}
}

func TestMQItoString(t *testing.T) {
	output := MQItoString("BACF", 7019)
	expected := "MQBACF_ALTERNATE_SECURITYID"
	if output != expected {
		t.Logf("Gave BACF, 7019. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CA", 2030)
	expected = "MQCA_CLUSTER_NAMELIST"
	if output != expected {
		t.Logf("Gave CA, 2030. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CA", 3134)
	expected = "MQCACF_ACTIVITY_DESC"
	if output != expected {
		t.Logf("Gave CA, 3134. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CA", 3529)
	expected = "MQCACH_CHANNEL_START_DATE"
	if output != expected {
		t.Logf("Gave CA, 3529. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CA", 2708)
	expected = "MQCAMO_END_TIME"
	if output != expected {
		t.Logf("Gave CA, 2708. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CC", -1)
	expected = "MQCC_UNKNOWN"
	if output != expected {
		t.Logf("Gave CC, -1. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("CMD", 208)
	expected = "MQCMD_CHANGE_PROT_POLICY"
	if output != expected {
		t.Logf("Gave CMD, 208. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 102)
	expected = "MQIA_ADOPTNEWMCA_CHECK"
	if output != expected {
		t.Logf("Gave IA, 102. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 1019)
	expected = "MQIACF_AUTH_INFO_ATTRS"
	if output != expected {
		t.Logf("Gave IA, 1019. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 1584)
	expected = "MQIACH_ADAPS_MAX"
	if output != expected {
		t.Logf("Gave IA, 1584. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 770)
	expected = "MQIAMO_CBS_FAILED"
	if output != expected {
		t.Logf("Gave IA, 770. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 745)
	expected = "MQIAMO64_BROWSE_BYTES"
	if output != expected {
		t.Logf("Gave IA, 745. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("OT", 1008)
	expected = "MQOT_SERVER_CHANNEL"
	if output != expected {
		t.Logf("Gave OT, 1008. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("RC", 2277)
	expected = "MQRC_CD_ERROR"
	if output != expected {
		t.Logf("Gave RC, 2277. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("RC", 3049)
	expected = "MQRCCF_CCSID_ERROR"
	if output != expected {
		t.Logf("Gave RC, 3049. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("BADVALUE", 0)
	expected = ""
	if output != expected {
		t.Logf("Gave BADVALUE, 0. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}

	output = MQItoString("IA", 0123123123123)
	expected = ""
	if output != expected {
		t.Logf("Gave IA, 0123123123123. Expected: %s, Got: %s", expected, output)
		t.Fail()
	}
}

// Tests for mqiPCF.go
func TestReadPCFHeader(t *testing.T) {
	testHeader := NewMQCFH()
	returned, offset := ReadPCFHeader(testHeader.Bytes())
	if returned.Type != testHeader.Type {
		t.Logf("Returned 'Type' does not match Initial: Expected: %d Got: %d", testHeader.Type, returned.Type)
		t.Fail()
	}
	if returned.StrucLength != testHeader.StrucLength {
		t.Logf("Returned 'StrucLength' does not match Initial: Expected: %d Got: %d", testHeader.StrucLength, returned.StrucLength)
		t.Fail()
	}
	if returned.Version != testHeader.Version {
		t.Logf("Returned 'Version' does not match Initial: Expected: %d Got: %d", testHeader.Version, returned.Version)
		t.Fail()
	}
	if returned.Command != testHeader.Command {
		t.Logf("Returned 'Command' does not match Initial: Expected: %d Got: %d", testHeader.Command, returned.Command)
		t.Fail()
	}
	if returned.MsgSeqNumber != testHeader.MsgSeqNumber {
		t.Logf("Returned 'MsgSeqNumber' does not match Initial: Expected: %d Got: %d", testHeader.MsgSeqNumber, returned.MsgSeqNumber)
		t.Fail()
	}
	if returned.Control != testHeader.Control {
		t.Logf("Returned 'Control' does not match Initial: Expected: %d Got: %d", testHeader.Control, returned.Control)
		t.Fail()
	}
	if returned.CompCode != testHeader.CompCode {
		t.Logf("Returned 'CompCode' does not match Initial: Expected: %d Got: %d", testHeader.CompCode, returned.CompCode)
		t.Fail()
	}
	if returned.Reason != testHeader.Reason {
		t.Logf("Returned 'Reason' does not match Initial: Expected: %d Got: %d", testHeader.Reason, returned.Reason)
		t.Fail()
	}
	if returned.ParameterCount != testHeader.ParameterCount {
		t.Logf("Returned 'ParameterCount' does not match Initial: Expected: %d Got: %d", testHeader.ParameterCount, returned.ParameterCount)
		t.Fail()
	}
	if offset != 36 {
		t.Logf("Expected offset to be 36 but was %d", offset)
		t.Fail()
	}
}

func TestReadPCFParameter(t *testing.T) {
	start := PCFParameter{
		Parameter:      MQCACF_APPL_NAME,
		Int64Value:     []int64{100},
		String:         []string{"HELLOTEST"},
		ParameterCount: 1,
	}

	t.Log("-MQCFT_INTEGER-")
	start.Type = MQCFT_INTEGER
	back, _ := ReadPCFParameter(start.Bytes())
	verifyParam(t, &start, back)

	t.Log("-MQCFT_STRING-")
	start.Type = MQCFT_STRING
	back, _ = ReadPCFParameter(start.Bytes())
	verifyParam(t, &start, back)

	// The rest of the types are not implemented in the Bytes()
	// function so cannot be tested.
}

func verifyParam(t *testing.T, given, returned *PCFParameter) {
	t.Log("Testing Type")
	if given.Type != returned.Type {
		t.Logf("Returned 'Type' does not match Initial: Expected: %d Got: %d", given.Type, returned.Type)
		t.Fail()
	}
	t.Log("Testing Parameter")
	if given.Parameter != returned.Parameter {
		t.Logf("Returned 'Parameter' does not match Initial: Expected: %d Got: %d", given.Parameter, returned.Parameter)
		t.Fail()
	}
	if given.Type == MQCFT_INTEGER || given.Type == MQCFT_INTEGER64 || given.Type == MQCFT_INTEGER_LIST || given.Type == MQCFT_INTEGER64_LIST {
		t.Log("Testing Length")
		if len(given.Int64Value) != len(returned.Int64Value) {
			t.Logf("Length of Returned 'Int64Value' does not match Initial: Expected: %d Got: %d", len(given.Int64Value), len(returned.Int64Value))
			t.Fail()
		} else if given.Int64Value[0] != returned.Int64Value[0] {
			t.Logf("Returned parameter 'Int64Value' did not match. Expected: %d, Got: %d", given.Int64Value[0], returned.Int64Value[0])
			t.Fail()
		}
	}

	if given.Type == MQCFT_STRING || given.Type == MQCFT_STRING_LIST {
		if len(given.String) != len(returned.String) {
			t.Logf("Length of Returned 'String' does not match Initial: Expected: %d Got: %d", len(given.String), len(returned.String))
			t.Fail()
		} else if given.String[0] != returned.String[0] {
			t.Logf("Returned parameter 'String' did not match. Expected: %s, Got: %s", given.String[0], returned.String[0])
			t.Fail()
		}
	}

	if given.Type == MQCFT_GROUP {
		if given.ParameterCount != returned.ParameterCount {
			t.Logf("Returned 'ParameterCount' does not match Initial: Expected: %d Got: %d", given.ParameterCount, returned.ParameterCount)
			t.Fail()
		}
	}

	if len(given.GroupList) != len(returned.GroupList) {
		t.Logf("Length of Returned 'GroupList' does not match Initial: Expected: %d Got: %d", len(given.GroupList), len(returned.GroupList))
		t.Fail()
	} // Should be nil
}

func TestRoundTo4(t *testing.T) {
	start := []int32{12, 13, 14, 15, 16, 17}
	expected := []int32{12, 16, 16, 16, 16, 20}

	for i, e := range start {
		back := roundTo4(e)
		if back != expected[i] {
			t.Logf("Passed: %d. Expected: %d. Got: %d", e, expected[i], back)
			t.Fail()
		}
	}
}
