package beat

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFQDNAwareHostname(t *testing.T) {
	info := Info{
		Hostname: "foo",
		FQDN:     "foo.bar.internal",
	}
	cases := map[string]struct {
		useFQDN bool
		want    string
	}{
		"fqdn_flag_enabled": {
			useFQDN: true,
			want:    "foo.bar.internal",
		},
		"fqdn_flag_disabled": {
			useFQDN: false,
			want:    "foo",
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			got := info.FQDNAwareHostname(tc.useFQDN)
			require.Equal(t, tc.want, got)
		})
	}
}
