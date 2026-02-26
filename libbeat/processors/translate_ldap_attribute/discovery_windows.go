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

//go:build windows && !requirefips

package translate_ldap_attribute

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// discoverDomainInPlatform Chain: Env -> API -> Reg(TCP) -> Reg(Krb) -> Hostname
func discoverDomainInPlatform() (string, error) {
	if d := os.Getenv("USERDNSDOMAIN"); d != "" {
		return d, nil
	}
	if d, err := getDomainAPI(); err == nil && d != "" {
		return d, nil
	}
	if d, err := getDomainReg(); err == nil && d != "" {
		return d, nil
	}
	if d, err := getDomainKrb(); err == nil && d != "" {
		return d, nil
	}
	if d, err := getDomainHostname(); err == nil && d != "" {
		return d, nil
	}
	return "", fmt.Errorf("domain discovery failed")
}

func getDomainAPI() (string, error) {
	const ComputerNameDnsDomain = 2
	k32 := windows.NewLazySystemDLL("kernel32.dll")
	proc := k32.NewProc("GetComputerNameExW")
	var n uint32
	proc.Call(uintptr(ComputerNameDnsDomain), 0, uintptr(unsafe.Pointer(&n)))
	if n == 0 {
		return "", fmt.Errorf("size 0")
	}
	b := make([]uint16, n)
	r, _, _ := proc.Call(uintptr(ComputerNameDnsDomain), uintptr(unsafe.Pointer(&b[0])), uintptr(unsafe.Pointer(&n)))
	if r == 0 {
		return "", fmt.Errorf("failed")
	}
	return syscall.UTF16ToString(b), nil
}

func getDomainReg() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Services\Tcpip\Parameters`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	val, _, err := k.GetStringValue("Domain")
	return val, err
}

func getDomainKrb() (string, error) {
	k, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\Lsa\Kerberos\Parameters`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer k.Close()
	val, _, err := k.GetStringValue("DefaultRealm")
	return val, err
}
