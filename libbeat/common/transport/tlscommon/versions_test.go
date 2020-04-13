package tlscommon

import (
	"crypto/tls"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestTLSVersion(t *testing.T) {
	// These tests are a bit verbose, but given the sensitivity to changes here, it's not a bad idea.
	tests := []struct {
		name string
		v    uint16
		want *TLSVersionDetails
	}{
		{
			"unknown",
			0x0,
			nil,
		},
		{
			"SSLv3",
			tls.VersionSSL30,
			&TLSVersionDetails{Version: "3.0", Protocol: "ssl", Combined: "SSLv3"},
		},
		{
			"TLSv1.0",
			tls.VersionTLS10,
			&TLSVersionDetails{Version: "1.0", Protocol: "tls", Combined: "TLSv1.0"},
		},
		{
			"TLSv1.1",
			tls.VersionTLS11,
			&TLSVersionDetails{Version: "1.1", Protocol: "tls", Combined: "TLSv1.1"},
		},
		{
			"TLSv1.2",
			tls.VersionTLS12,
			&TLSVersionDetails{Version: "1.2", Protocol: "tls", Combined: "TLSv1.2"},
		},
		{
			"TLSv1.3",
			tls.VersionTLS13,
			&TLSVersionDetails{Version: "1.3", Protocol: "tls", Combined: "TLSv1.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tv := TLSVersion(tt.v)
			require.Equal(t, tt.want, tv.Details())
			if tt.want == nil {
				require.Equal(t, tt.want, tv.Details())
				require.Equal(t, tt.name, "unknown")
			} else {
				require.Equal(t, tt.name, tv.String())
			}
		})
	}
}
