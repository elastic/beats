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

package rule

import "fmt"

// Type specifies the audit rule type.
type Type int

// The rule types supported by this package.
const (
	DeleteAllRuleType      Type = iota + 1 // DeleteAllRule
	FileWatchRuleType                      // FileWatchRule
	AppendSyscallRuleType                  // SyscallRule
	PrependSyscallRuleType                 // SyscallRule
)

// Rule is the generic interface that all rule types implement.
type Rule interface {
	TypeOf() Type // TypeOf returns the type of rule.
}

// DeleteAllRule deletes all existing rules.
type DeleteAllRule struct {
	Type Type
	Keys []string // Delete rules that have these keys.
}

// TypeOf returns DeleteAllRuleType.
func (r *DeleteAllRule) TypeOf() Type { return r.Type }

// FileWatchRule is used to audit access to particular files or directories
// that you may be interested in.
type FileWatchRule struct {
	Type        Type
	Path        string
	Permissions []AccessType
	Keys        []string
}

// TypeOf returns FileWatchRuleType.
func (r *FileWatchRule) TypeOf() Type { return r.Type }

// SyscallRule is used to audit invocations of specific syscalls.
type SyscallRule struct {
	Type     Type
	List     string
	Action   string
	Filters  []FilterSpec
	Syscalls []string
	Keys     []string
}

// TypeOf returns either AppendSyscallRuleType or PrependSyscallRuleType.
func (r *SyscallRule) TypeOf() Type { return r.Type }

// AccessType specifies the type of file access to audit.
type AccessType uint8

// The access types that can be audited for file watches.
const (
	ReadAccessType AccessType = iota + 1
	WriteAccessType
	ExecuteAccessType
	AttributeChangeAccessType
)

var accessTypeName = map[AccessType]string{
	ReadAccessType:            "read",
	WriteAccessType:           "write",
	ExecuteAccessType:         "execute",
	AttributeChangeAccessType: "attribute",
}

func (t AccessType) String() string {
	name, found := accessTypeName[t]
	if found {
		return name
	}
	return "unknown"
}

// FilterType specifies a type of filter to apply to a syscall rule.
type FilterType uint8

// The type of filters that can be applied.
const (
	InterFieldFilterType FilterType = iota + 1 // Inter-field comparison filtering (-C).
	ValueFilterType                            // Filtering based on values (-F).
)

// FilterSpec defines a filter to apply to a syscall rule.
type FilterSpec struct {
	Type       FilterType
	LHS        string
	Comparator string
	RHS        string
}

func (f *FilterSpec) String() string {
	return fmt.Sprintf("%v %v %v", f.LHS, f.Comparator, f.RHS)
}
