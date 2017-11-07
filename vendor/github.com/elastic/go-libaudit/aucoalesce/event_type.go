// Copyright 2017 Elasticsearch Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package aucoalesce

import (
	. "github.com/elastic/go-libaudit/auparse"
)

// AuditEventType is a categorization of a simple or compound audit event.
type AuditEventType uint16

const (
	EventTypeUnknown AuditEventType = iota
	EventTypeUserspace
	EventTypeSystemServices
	EventTypeConfig
	EventTypeTTY
	EventTypeUserAccount
	EventTypeUserLogin
	EventTypeAuditDaemon
	EventTypeMACDecision
	EventTypeAnomoly
	EventTypeIntegrity
	EventTypeAnomolyResponse
	EventTypeMAC
	EventTypeCrypto
	EventTypeVirt
	EventTypeAuditRule
	EventTypeDACDecision
	EventTypeGroupChange
)

var auditEventTypeNames = map[AuditEventType]string{
	EventTypeUnknown:         "unknown",
	EventTypeUserspace:       "user-space",
	EventTypeSystemServices:  "system-services",
	EventTypeConfig:          "configuration",
	EventTypeTTY:             "TTY",
	EventTypeUserAccount:     "user-account",
	EventTypeUserLogin:       "user-login",
	EventTypeAuditDaemon:     "audit-daemon",
	EventTypeMACDecision:     "mac-decision",
	EventTypeAnomoly:         "anomoly",
	EventTypeIntegrity:       "integrity",
	EventTypeAnomolyResponse: "anomaly-response",
	EventTypeMAC:             "mac",
	EventTypeCrypto:          "crypto",
	EventTypeVirt:            "virt",
	EventTypeAuditRule:       "audit-rule",
	EventTypeDACDecision:     "dac-decision",
	EventTypeGroupChange:     "group-change",
}

func (t AuditEventType) String() string {
	name, found := auditEventTypeNames[t]
	if found {
		return name
	}
	return auditEventTypeNames[EventTypeUnknown]
}

func (t AuditEventType) MarshalText() (text []byte, err error) {
	return []byte(t.String()), nil
}

func GetAuditEventType(t AuditMessageType) AuditEventType {
	// Ported from: https://github.com/linux-audit/audit-userspace/blob/v2.7.5/auparse/normalize.c#L681
	switch {
	case t >= AUDIT_USER_AUTH && t <= AUDIT_USER_END,
		t >= AUDIT_USER_CHAUTHTOK && t <= AUDIT_CRED_REFR,
		t >= AUDIT_USER_LOGIN && t <= AUDIT_USER_LOGOUT,
		t == AUDIT_GRP_AUTH:
		return EventTypeUserLogin
	case t >= AUDIT_ADD_USER && t <= AUDIT_DEL_GROUP,
		t >= AUDIT_GRP_MGMT && t <= AUDIT_GRP_CHAUTHTOK,
		t >= AUDIT_ACCT_LOCK && t <= AUDIT_ACCT_UNLOCK:
		return EventTypeUserAccount
	case t == AUDIT_KERNEL,
		t >= AUDIT_SYSTEM_BOOT && t <= AUDIT_SERVICE_STOP:
		return EventTypeSystemServices
	case t == AUDIT_USYS_CONFIG,
		t == AUDIT_CONFIG_CHANGE,
		t == AUDIT_NETFILTER_CFG,
		t >= AUDIT_FEATURE_CHANGE && t <= AUDIT_REPLACE:
		return EventTypeConfig
	case t == AUDIT_SECCOMP:
		return EventTypeDACDecision
	case t >= AUDIT_CHGRP_ID && t <= AUDIT_TRUSTED_APP,
		t == AUDIT_USER_CMD,
		t == AUDIT_CHUSER_ID:
		return EventTypeUserspace
	case t == AUDIT_USER_TTY, t == AUDIT_TTY:
		return EventTypeTTY
	case t >= AUDIT_DAEMON_START && t <= AUDIT_LAST_DAEMON:
		return EventTypeAuditDaemon
	case t == AUDIT_USER_SELINUX_ERR,
		t == AUDIT_USER_AVC,
		t >= AUDIT_APPARMOR_ALLOWED && t <= AUDIT_APPARMOR_DENIED,
		t == AUDIT_APPARMOR_ERROR,
		t >= AUDIT_AVC && t <= AUDIT_AVC_PATH:
		return EventTypeMACDecision
	case t >= AUDIT_INTEGRITY_DATA && t <= AUDIT_INTEGRITY_LAST_MSG,
		t == AUDIT_ANOM_RBAC_INTEGRITY_FAIL:
		return EventTypeIntegrity
	case t >= AUDIT_ANOM_PROMISCUOUS && t <= AUDIT_LAST_KERN_ANOM_MSG,
		t >= AUDIT_ANOM_LOGIN_FAILURES && t <= AUDIT_ANOM_RBAC_FAIL,
		t >= AUDIT_ANOM_CRYPTO_FAIL && t <= AUDIT_LAST_ANOM_MSG:
		return EventTypeAnomoly
	case t >= AUDIT_RESP_ANOMALY && t <= AUDIT_LAST_ANOM_RESP:
		return EventTypeAnomolyResponse
	case t >= AUDIT_MAC_POLICY_LOAD && t <= AUDIT_LAST_SELINUX,
		t >= AUDIT_AA && t <= AUDIT_APPARMOR_AUDIT,
		t >= AUDIT_APPARMOR_HINT && t <= AUDIT_APPARMOR_STATUS,
		t >= AUDIT_USER_ROLE_CHANGE && t <= AUDIT_LAST_USER_LSPP_MSG:
		return EventTypeMAC
	case t >= AUDIT_FIRST_KERN_CRYPTO_MSG && t <= AUDIT_LAST_KERN_CRYPTO_MSG,
		t >= AUDIT_CRYPTO_TEST_USER && t <= AUDIT_LAST_CRYPTO_MSG:
		return EventTypeCrypto
	case t >= AUDIT_VIRT_CONTROL && t <= AUDIT_LAST_VIRT_MSG:
		return EventTypeVirt
	case t >= AUDIT_SYSCALL && t <= AUDIT_SOCKETCALL,
		t >= AUDIT_SOCKADDR && t <= AUDIT_MQ_GETSETATTR,
		t >= AUDIT_FD_PAIR && t <= AUDIT_OBJ_PID,
		t >= AUDIT_BPRM_FCAPS && t <= AUDIT_NETFILTER_PKT:
		return EventTypeAuditRule
	default:
		return EventTypeUnknown
	}
}
