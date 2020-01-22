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

//+build ignore

package rule

/*
#include <linux/audit.h>
#include <linux/stat.h>
*/
import "C"

type filter uint32

// https://github.com/linux-audit/audit-kernel/blob/v3.15/include/uapi/linux/audit.h#L147-L157
const (
	userFilter    filter = C.AUDIT_FILTER_USER  /* Apply rule to user-generated messages */
	taskFilter    filter = C.AUDIT_FILTER_TASK  /* Apply rule at task creation (not syscall) */
	entryFilter   filter = C.AUDIT_FILTER_ENTRY /* Apply rule at syscall entry */
	watchFilter   filter = C.AUDIT_FILTER_WATCH /* Apply rule to file system watches */
	exitFilter    filter = C.AUDIT_FILTER_EXIT  /* Apply rule at syscall exit */
	typeFilter    filter = C.AUDIT_FILTER_TYPE  /* Apply rule at audit_log_start */
	excludeFilter        = typeFilter

	prependFilter filter = C.AUDIT_FILTER_PREPEND /* Prepend to front of list */
)

type action uint32

// https://github.com/linux-audit/audit-kernel/blob/v3.15/include/uapi/linux/audit.h#L159-L162
const (
	neverAction    action = C.AUDIT_NEVER    /* Do not build context if rule matches */
	possibleAction action = C.AUDIT_POSSIBLE /* Build context if rule matches  */
	alwaysAction   action = C.AUDIT_ALWAYS   /* Generate audit record if rule matches */
)

type field uint32

/* Rule fields */
// Values >= 100 are ONLY useful when checking at syscall exit time (AUDIT_AT_EXIT).
const (
	auidField               field = C.AUDIT_LOGINUID
	archField               field = C.AUDIT_ARCH
	arg0Field               field = C.AUDIT_ARG0
	arg1Field               field = C.AUDIT_ARG1
	arg2Field               field = C.AUDIT_ARG2
	arg3Field               field = C.AUDIT_ARG3
	devMajorField           field = C.AUDIT_DEVMAJOR
	devMinorField           field = C.AUDIT_DEVMINOR
	dirField                field = C.AUDIT_DIR
	egidField               field = C.AUDIT_EGID
	euidField               field = C.AUDIT_EUID
	exeField                field = C.AUDIT_EXE // Added in v4.3.
	exitField               field = C.AUDIT_EXIT
	fsgidField              field = C.AUDIT_FSGID
	fsuidField              field = C.AUDIT_FSUID
	filetypeField           field = C.AUDIT_FILETYPE
	gidField                field = C.AUDIT_GID
	inodeField              field = C.AUDIT_INODE
	keyField                field = C.AUDIT_FILTERKEY
	msgTypeField            field = C.AUDIT_MSGTYPE
	objectGIDField          field = C.AUDIT_OBJ_GID
	objectLevelHighField    field = C.AUDIT_OBJ_LEV_HIGH
	objectLevelLowField     field = C.AUDIT_OBJ_LEV_LOW
	objectRoleField         field = C.AUDIT_OBJ_ROLE
	objectTypeField         field = C.AUDIT_OBJ_TYPE
	objectUIDField          field = C.AUDIT_OBJ_UID
	objectUserField         field = C.AUDIT_OBJ_USER
	pathField               field = C.AUDIT_WATCH
	pidField                field = C.AUDIT_PID
	ppidField               field = C.AUDIT_PPID
	permField               field = C.AUDIT_PERM
	persField               field = C.AUDIT_PERS
	sgidField               field = C.AUDIT_SGID
	suidField               field = C.AUDIT_SUID
	subjectClearanceField   field = C.AUDIT_SUBJ_CLR
	subjectRoleField        field = C.AUDIT_SUBJ_ROLE
	subjectSensitivityField field = C.AUDIT_SUBJ_SEN
	subjectTypeField        field = C.AUDIT_SUBJ_TYPE
	subjectUserField        field = C.AUDIT_SUBJ_USER
	successField            field = C.AUDIT_SUCCESS
	uidField                field = C.AUDIT_UID

	fieldCompare field = C.AUDIT_FIELD_COMPARE

	//SessionIDField          field = C.AUDIT_SESSIONID // Added in v4.10.
)

type operator uint32

// https://github.com/linux-audit/audit-kernel/blob/v3.15/include/uapi/linux/audit.h#L294-L301
const (
	bitMaskOperator            operator = C.AUDIT_BIT_MASK
	lessThanOperator           operator = C.AUDIT_LESS_THAN
	greaterThanOperator        operator = C.AUDIT_GREATER_THAN
	notEqualOperator           operator = C.AUDIT_NOT_EQUAL
	equalOperator              operator = C.AUDIT_EQUAL
	bitTestOperator            operator = C.AUDIT_BIT_TEST
	lessThanOrEqualOperator    operator = C.AUDIT_LESS_THAN_OR_EQUAL
	greaterThanOrEqualOperator operator = C.AUDIT_GREATER_THAN_OR_EQUAL
)

type comparison uint32

const (
	_AUDIT_COMPARE_UID_TO_OBJ_UID   comparison = C.AUDIT_COMPARE_UID_TO_OBJ_UID
	_AUDIT_COMPARE_GID_TO_OBJ_GID   comparison = C.AUDIT_COMPARE_GID_TO_OBJ_GID
	_AUDIT_COMPARE_EUID_TO_OBJ_UID  comparison = C.AUDIT_COMPARE_EUID_TO_OBJ_UID
	_AUDIT_COMPARE_EGID_TO_OBJ_GID  comparison = C.AUDIT_COMPARE_EGID_TO_OBJ_GID
	_AUDIT_COMPARE_AUID_TO_OBJ_UID  comparison = C.AUDIT_COMPARE_AUID_TO_OBJ_UID
	_AUDIT_COMPARE_SUID_TO_OBJ_UID  comparison = C.AUDIT_COMPARE_SUID_TO_OBJ_UID
	_AUDIT_COMPARE_SGID_TO_OBJ_GID  comparison = C.AUDIT_COMPARE_SGID_TO_OBJ_GID
	_AUDIT_COMPARE_FSUID_TO_OBJ_UID comparison = C.AUDIT_COMPARE_FSUID_TO_OBJ_UID
	_AUDIT_COMPARE_FSGID_TO_OBJ_GID comparison = C.AUDIT_COMPARE_FSGID_TO_OBJ_GID

	_AUDIT_COMPARE_UID_TO_AUID  comparison = C.AUDIT_COMPARE_UID_TO_AUID
	_AUDIT_COMPARE_UID_TO_EUID  comparison = C.AUDIT_COMPARE_UID_TO_EUID
	_AUDIT_COMPARE_UID_TO_FSUID comparison = C.AUDIT_COMPARE_UID_TO_FSUID
	_AUDIT_COMPARE_UID_TO_SUID  comparison = C.AUDIT_COMPARE_UID_TO_SUID

	_AUDIT_COMPARE_AUID_TO_FSUID comparison = C.AUDIT_COMPARE_AUID_TO_FSUID
	_AUDIT_COMPARE_AUID_TO_SUID  comparison = C.AUDIT_COMPARE_AUID_TO_SUID
	_AUDIT_COMPARE_AUID_TO_EUID  comparison = C.AUDIT_COMPARE_AUID_TO_EUID

	_AUDIT_COMPARE_EUID_TO_SUID  comparison = C.AUDIT_COMPARE_EUID_TO_SUID
	_AUDIT_COMPARE_EUID_TO_FSUID comparison = C.AUDIT_COMPARE_EUID_TO_FSUID

	_AUDIT_COMPARE_SUID_TO_FSUID comparison = C.AUDIT_COMPARE_SUID_TO_FSUID

	_AUDIT_COMPARE_GID_TO_EGID  comparison = C.AUDIT_COMPARE_GID_TO_EGID
	_AUDIT_COMPARE_GID_TO_FSGID comparison = C.AUDIT_COMPARE_GID_TO_FSGID
	_AUDIT_COMPARE_GID_TO_SGID  comparison = C.AUDIT_COMPARE_GID_TO_SGID

	_AUDIT_COMPARE_EGID_TO_FSGID comparison = C.AUDIT_COMPARE_EGID_TO_FSGID
	_AUDIT_COMPARE_EGID_TO_SGID  comparison = C.AUDIT_COMPARE_EGID_TO_SGID
	_AUDIT_COMPARE_SGID_TO_FSGID comparison = C.AUDIT_COMPARE_SGID_TO_FSGID
)

type permission uint32

const (
	execPerm  permission = C.AUDIT_PERM_EXEC
	writePerm permission = C.AUDIT_PERM_WRITE
	readPerm  permission = C.AUDIT_PERM_READ
	attrPerm  permission = C.AUDIT_PERM_ATTR
)

type filetype uint32

const (
	fileFiletype      filetype = C.S_IFREG
	socketFiletype    filetype = C.S_IFSOCK
	linkFiletype      filetype = C.S_IFLNK
	blockFiletype     filetype = C.S_IFBLK
	dirFiletype       filetype = C.S_IFDIR
	characterFiletype filetype = C.S_IFCHR
	fifoFiletype      filetype = C.S_IFIFO
)
