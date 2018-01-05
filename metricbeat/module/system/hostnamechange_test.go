package system

import (
	"bytes"
	"io/ioutil"
	"testing"
)

// Checks that the Host Overview dashboard contains the CHANGEME_HOSTNAME variable
// that the dashboard loader code magically changes to the hostname on which the Beat
// is running.
func TestHostDashboardHasChangeableHost(t *testing.T) {
	dashPath := "_meta/kibana/6/dashboard/Metricbeat-host-overview.json"
	contents, err := ioutil.ReadFile(dashPath)
	if err != nil {
		t.Fatalf("Error reading file %s: %v", dashPath, err)
	}
	if !bytes.Contains(contents, []byte("CHANGEME_HOSTNAME")) {
		t.Errorf("Dashboard '%s' doesn't contain string 'CHANGEME_HOSTNAME'. See elastic/beats#5340", dashPath)
	}
}
