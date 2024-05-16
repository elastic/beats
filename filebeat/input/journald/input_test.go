// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

//go:build linux && cgo && withjournald

package journald

import (
	"context"
	"fmt"
	"path"
	"testing"

	"github.com/elastic/elastic-agent-libs/mapstr"
)

func TestInputFieldsTranslation(t *testing.T) {
	// A few random keys to verify
	keysToCheck := map[string]string{
		"systemd.user_unit": "log-service.service",
		"process.pid":       "2084785",
		"systemd.transport": "stdout",
		"host.hostname":     "x-wing",
	}

	testCases := map[string]struct {
		saveRemoteHostname bool
	}{
		"Save hostname enabled":  {saveRemoteHostname: true},
		"Save hostname disabled": {saveRemoteHostname: true},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			env := newInputTestingEnvironment(t)

			inp := env.mustCreateInput(mapstr.M{
				"paths":                 []string{path.Join("testdata", "input-multiline-parser.journal")},
				"include_matches.match": []string{"_SYSTEMD_USER_UNIT=log-service.service"},
				"save_remote_hostname":  tc.saveRemoteHostname,
			})

			ctx, cancelInput := context.WithCancel(context.Background())
			env.startInput(ctx, inp)
			env.waitUntilEventCount(6)

			for eventIdx, event := range env.pipeline.clients[0].GetEvents() {
				for k, v := range keysToCheck {
					got, err := event.Fields.GetValue(k)
					if err == nil {
						if got, want := fmt.Sprint(got), v; got != want {
							t.Errorf("expecting key %q to have value '%#v', but got '%#v' instead", k, want, got)
						}
					} else {
						t.Errorf("key %q not found on event %d", k, eventIdx)
					}
				}
				if tc.saveRemoteHostname {
					v, err := event.Fields.GetValue("log.source.address")
					if err != nil {
						t.Errorf("key 'log.source.address' not found on evet %d", eventIdx)
					}

					if got, want := fmt.Sprint(v), "x-wing"; got != want {
						t.Errorf("expecting key 'log.source.address' to have value '%#v', but got '%#v' instead", want, got)
					}
				}
			}
			cancelInput()
		})
	}
}

func TestParseJournaldVersion(t *testing.T) {
	foo := map[string]struct {
		data     string
		expected int
	}{
		"Archlinux": {
			expected: 255,
			data: `systemd 255 (255.6-1-arch)
+PAM +AUDIT -SELINUX -APPARMOR -IMA +SMACK +SECCOMP +GCRYPT +GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +IPTC +KMOD +LIBCRYPTSETUP +LIBFDISK +PCRE2 +PWQUALITY +P11KIT +QRENCODE +TPM2 +BZIP2 +LZ4 +XZ +ZLIB +ZSTD +BPF_FRAMEWORK +XKBCOMMON +UTMP -SYSVINIT default-hierarchy=unified`,
		},
		"AmazonLinux2": {
			expected: 219,
			data: `systemd 219
				+PAM +AUDIT +SELINUX +IMA -APPARMOR +SMACK +SYSVINIT +UTMP +LIBCRYPTSETUP +GCRYPT +GNUTLS +ACL +XZ +LZ4 -SECCOMP +BLKID +ELFUTILS +KMOD +IDN`,
		},
		"Ubuntu 2204": {
			expected: 249,
			data: `systemd 249 (249.11-0ubuntu3.12)
		+PAM +AUDIT +SELINUX +APPARMOR +IMA +SMACK +SECCOMP +GCRYPT +GNUTLS +OPENSSL +ACL +BLKID +CURL +ELFUTILS +FIDO2 +IDN2 -IDN +IPTC +KMOD +LIBCRYPTSETUP +LIBFDISK +PCRE2 -PWQUALITY -P11KIT -QRENCODE +BZIP2 +LZ4 +XZ +ZLIB +ZSTD -XKBCOMMON +UTMP +SYSVINIT default-hierarchy=unified`,
		},
	}

	for name, tc := range foo {
		t.Run(name, func(t *testing.T) {
			version, err := parseJournaldVersion(tc.data)
			if err != nil {
				t.Errorf("did not expect an error: %s", err)
			}

			if version != tc.expected {
				t.Errorf("expecting version %d, got %d", tc.expected, version)
			}
		})
	}
}

func TestGetJounraldVersion(t *testing.T) {
	// This test already has build tags to only run on systems
	// with systemd. So there should be no problem calling
	// journalctl directly.
	version, err := getJournaldVersion()
	if err != nil {
		t.Fatalf("did not expect an error: %s", err)
	}

	if version == 0 {
		t.Fatal("version must be grater than 0")
	}
}
