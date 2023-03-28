package host

import (
	"github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/go-sysinfo/types"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestMapHostInfo(t *testing.T) {
	bootTime := time.Now()
	containerized := true
	osInfo := types.OSInfo{
		Type:     "darwin",
		Family:   "family",
		Platform: "platform",
		Name:     "macos ventura",
		Version:  "13.2.1",
		Major:    13,
		Minor:    2,
		Patch:    1,
		Build:    "build",
		Codename: "ventura",
	}
	hostInfo := types.HostInfo{
		Architecture:      "x86_64",
		BootTime:          bootTime,
		Containerized:     &containerized,
		Hostname:          "foo",
		IPs:               []string{"1.2.3.4", "192.168.1.1"},
		KernelVersion:     "22.3.0",
		MACs:              []string{"56:9c:17:54:19:15", "5c:e9:1e:c4:37:66"},
		OS:                &osInfo,
		Timezone:          "",
		TimezoneOffsetSec: 0,
		UniqueID:          "a39b4c1ee4",
	}

	tests := map[string]struct {
		fqdn     string
		expected mapstr.M
	}{
		"with_fqdn": {
			fqdn: "foo.bar.local",
			expected: mapstr.M{
				"host": mapstr.M{
					"architecture":  "x86_64",
					"containerized": true,
					"hostname":      "foo",
					"id":            "a39b4c1ee4",
					"name":          "foo.bar.local",
					"os": mapstr.M{
						"build":    "build",
						"codename": "ventura",
						"family":   "family",
						"kernel":   "22.3.0",
						"name":     "macos ventura",
						"platform": "platform",
						"type":     "darwin",
						"version":  "13.2.1",
					},
				},
			},
		},
		"without_fqdn": {
			expected: mapstr.M{
				"host": mapstr.M{
					"architecture":  "x86_64",
					"containerized": true,
					"hostname":      "foo",
					"id":            "a39b4c1ee4",
					"name":          "foo",
					"os": mapstr.M{
						"build":    "build",
						"codename": "ventura",
						"family":   "family",
						"kernel":   "22.3.0",
						"name":     "macos ventura",
						"platform": "platform",
						"type":     "darwin",
						"version":  "13.2.1",
					},
				},
			},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			actual := MapHostInfo(hostInfo, test.fqdn)
			require.Equal(t, test.expected, actual)
		})
	}
}
