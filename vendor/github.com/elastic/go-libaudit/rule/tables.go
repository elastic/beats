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

import "github.com/elastic/go-libaudit/auparse"

var (
	reverseSyscall          map[string]map[string]int
	reverseArch             map[string]uint32
	reverseOperatorsTable   map[operator]string
	reverseFieldsTable      map[field]string
	reverseComparisonsTable map[comparison][2]field
)

func init() {
	buildReverseSyscallTable()
	buildReverseArchTable()
	buildReverseOperatorsTable()
	buildReverseFieldsTable()
	buildReverseComparisonsTable()
}

func buildReverseSyscallTable() {
	reverseSyscall = make(map[string]map[string]int, len(auparse.AuditSyscalls))

	for arch, syscallToName := range auparse.AuditSyscalls {
		archTable := make(map[string]int, len(syscallToName))
		reverseSyscall[arch] = archTable

		for syscallNum, syscallName := range syscallToName {
			archTable[syscallName] = syscallNum
		}
	}
}

func buildReverseArchTable() {
	reverseArch = make(map[string]uint32, len(auparse.AuditArchNames))

	for arch, name := range auparse.AuditArchNames {
		reverseArch[name] = uint32(arch)
	}
}

func buildReverseOperatorsTable() {
	reverseOperatorsTable = make(map[operator]string, len(operatorsTable))
	for k, v := range operatorsTable {
		reverseOperatorsTable[v] = k
	}
}

func buildReverseFieldsTable() {
	reverseFieldsTable = make(map[field]string, len(fieldsTable))
	for k, v := range fieldsTable {
		reverseFieldsTable[v] = k
	}
}

func buildReverseComparisonsTable() {
	reverseComparisonsTable = make(map[comparison][2]field, len(comparisonsTable))
	for lhs, table := range comparisonsTable {
		for rhs, comp := range table {
			if _, found := reverseComparisonsTable[comp]; !found {
				reverseComparisonsTable[comp] = [2]field{lhs, rhs}
			}
		}
	}
}

var operatorsTable = map[string]operator{
	"&":  bitMaskOperator,
	"<":  lessThanOperator,
	">":  greaterThanOperator,
	"!=": notEqualOperator,
	"=":  equalOperator,
	"&=": bitTestOperator,
	"<=": lessThanOrEqualOperator,
	">=": greaterThanOrEqualOperator,
}

var fieldsTable = map[string]field{
	"auid":         auidField,
	"arch":         archField,
	"a0":           arg0Field,
	"a1":           arg1Field,
	"a2":           arg2Field,
	"a3":           arg3Field,
	"devmajor":     devMajorField,
	"devminor":     devMinorField,
	"dir":          dirField,
	"egid":         egidField,
	"euid":         euidField,
	"exe":          exeField,
	"exit":         exitField,
	"fsgid":        fsgidField,
	"fsuid":        fsuidField,
	"filetype":     filetypeField,
	"gid":          gidField,
	"inode":        inodeField,
	"key":          keyField,
	"msgtype":      msgTypeField,
	"obj_gid":      objectGIDField,
	"obj_lev_high": objectLevelHighField,
	"obj_lev_low":  objectLevelLowField,
	"obj_role":     objectRoleField,
	"obj_type":     objectTypeField,
	"obj_uid":      objectUIDField,
	"obj_user":     objectUserField,
	"path":         pathField,
	"pid":          pidField,
	"ppid":         ppidField,
	"perm":         permField,
	"pers":         persField,
	"sgid":         sgidField,
	"suid":         suidField,
	"subj_clr":     subjectClearanceField,
	"subj_role":    subjectRoleField,
	"subj_sen":     subjectSensitivityField,
	"subj_type":    subjectTypeField,
	"subj_user":    subjectUserField,
	"success":      successField,
	"uid":          uidField,
}

var comparisonsTable = map[field]map[field]comparison{
	euidField: {
		auidField:      _AUDIT_COMPARE_AUID_TO_EUID,
		fsuidField:     _AUDIT_COMPARE_EUID_TO_FSUID,
		objectUIDField: _AUDIT_COMPARE_EUID_TO_OBJ_UID,
		suidField:      _AUDIT_COMPARE_EUID_TO_SUID,
		uidField:       _AUDIT_COMPARE_UID_TO_EUID,
	},
	fsuidField: {
		auidField:      _AUDIT_COMPARE_AUID_TO_FSUID,
		euidField:      _AUDIT_COMPARE_EUID_TO_FSUID,
		objectUIDField: _AUDIT_COMPARE_FSUID_TO_OBJ_UID,
		suidField:      _AUDIT_COMPARE_SUID_TO_FSUID,
		uidField:       _AUDIT_COMPARE_UID_TO_FSUID,
	},
	auidField: {
		euidField:      _AUDIT_COMPARE_AUID_TO_EUID,
		fsuidField:     _AUDIT_COMPARE_AUID_TO_FSUID,
		objectUIDField: _AUDIT_COMPARE_AUID_TO_OBJ_UID,
		suidField:      _AUDIT_COMPARE_AUID_TO_SUID,
		uidField:       _AUDIT_COMPARE_UID_TO_AUID,
	},
	suidField: {
		auidField:      _AUDIT_COMPARE_AUID_TO_SUID,
		euidField:      _AUDIT_COMPARE_EUID_TO_SUID,
		fsuidField:     _AUDIT_COMPARE_SUID_TO_FSUID,
		objectUIDField: _AUDIT_COMPARE_SUID_TO_OBJ_UID,
		uidField:       _AUDIT_COMPARE_UID_TO_SUID,
	},
	objectUIDField: {
		auidField:  _AUDIT_COMPARE_AUID_TO_OBJ_UID,
		euidField:  _AUDIT_COMPARE_EUID_TO_OBJ_UID,
		fsuidField: _AUDIT_COMPARE_FSUID_TO_OBJ_UID,
		uidField:   _AUDIT_COMPARE_UID_TO_OBJ_UID,
		suidField:  _AUDIT_COMPARE_SUID_TO_OBJ_UID,
	},
	uidField: {
		auidField:      _AUDIT_COMPARE_UID_TO_AUID,
		euidField:      _AUDIT_COMPARE_UID_TO_EUID,
		fsuidField:     _AUDIT_COMPARE_UID_TO_FSUID,
		objectUIDField: _AUDIT_COMPARE_UID_TO_OBJ_UID,
		suidField:      _AUDIT_COMPARE_UID_TO_SUID,
	},
	egidField: {
		fsgidField:     _AUDIT_COMPARE_EGID_TO_FSGID,
		gidField:       _AUDIT_COMPARE_GID_TO_EGID,
		objectGIDField: _AUDIT_COMPARE_EGID_TO_OBJ_GID,
		sgidField:      _AUDIT_COMPARE_EGID_TO_SGID,
	},
	fsgidField: {
		sgidField:      _AUDIT_COMPARE_SGID_TO_FSGID,
		gidField:       _AUDIT_COMPARE_GID_TO_FSGID,
		objectGIDField: _AUDIT_COMPARE_FSGID_TO_OBJ_GID,
		egidField:      _AUDIT_COMPARE_EGID_TO_FSGID,
	},
	gidField: {
		egidField:      _AUDIT_COMPARE_GID_TO_EGID,
		fsgidField:     _AUDIT_COMPARE_GID_TO_FSGID,
		objectGIDField: _AUDIT_COMPARE_GID_TO_OBJ_GID,
		sgidField:      _AUDIT_COMPARE_GID_TO_SGID,
	},
	objectGIDField: {
		egidField:  _AUDIT_COMPARE_EGID_TO_OBJ_GID,
		fsgidField: _AUDIT_COMPARE_FSGID_TO_OBJ_GID,
		gidField:   _AUDIT_COMPARE_GID_TO_OBJ_GID,
		sgidField:  _AUDIT_COMPARE_SGID_TO_OBJ_GID,
	},
	sgidField: {
		fsgidField:     _AUDIT_COMPARE_SGID_TO_FSGID,
		gidField:       _AUDIT_COMPARE_GID_TO_SGID,
		objectGIDField: _AUDIT_COMPARE_SGID_TO_OBJ_GID,
		egidField:      _AUDIT_COMPARE_EGID_TO_SGID,
	},
}
