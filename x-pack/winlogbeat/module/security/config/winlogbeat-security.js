// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var security = (function () {
    var path = require("path");
    var processor = require("processor");
    var winlogbeat = require("winlogbeat");

    // Logon Types
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/basic-audit-logon-events
    var logonTypes = {
        "2": "Interactive",
        "3": "Network",
        "4": "Batch",
        "5": "Service",
        "7": "Unlock",
        "8": "NetworkCleartext",
        "9": "NewCredentials",
        "10": "RemoteInteractive",
        "11": "CachedInteractive",
    };

    // ECS Allowed Event Outcome
    // https://www.elastic.co/guide/en/ecs/current/ecs-allowed-values-event-outcome.html
    var eventOutcomes = {
        "Audit Success": "success",
        "Audit Failure": "failure",
    };

    // User Account Control Attributes Table
    // https://support.microsoft.com/es-us/help/305144/how-to-use-useraccountcontrol-to-manipulate-user-account-properties
    var uacFlags = [
        [0x0001, 'SCRIPT'],
        [0x0002, 'ACCOUNTDISABLE'],
        [0x0008, 'HOMEDIR_REQUIRED'],
        [0x0010, 'LOCKOUT'],
        [0x0020, 'PASSWD_NOTREQD'],
        [0x0040, 'PASSWD_CANT_CHANGE'],
        [0x0080, 'ENCRYPTED_TEXT_PWD_ALLOWED'],
        [0x0100, 'TEMP_DUPLICATE_ACCOUNT'],
        [0x0200, 'NORMAL_ACCOUNT'],
        [0x0800, 'INTERDOMAIN_TRUST_ACCOUNT'],
        [0x1000, 'WORKSTATION_TRUST_ACCOUNT'],
        [0x2000, 'SERVER_TRUST_ACCOUNT'],
        [0x10000, 'DONT_EXPIRE_PASSWORD'],
        [0x20000, 'MNS_LOGON_ACCOUNT'],
        [0x40000, 'SMARTCARD_REQUIRED'],
        [0x80000, 'TRUSTED_FOR_DELEGATION'],
        [0x100000, 'NOT_DELEGATED'],
        [0x200000, 'USE_DES_KEY_ONLY'],
        [0x400000, 'DONT_REQ_PREAUTH'],
        [0x800000, 'PASSWORD_EXPIRED'],
        [0x1000000, 'TRUSTED_TO_AUTH_FOR_DELEGATION'],
        [0x04000000, 'PARTIAL_SECRETS_ACCOUNT'],
    ];

    // Kerberos TGT and TGS Ticket Options
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4768
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4769
    var ticketOptions = [
        "Reserved",
        "Forwardable",
        "Forwarded",
        "Proxiable",
        "Proxy",
        "Allow-postdate",
        "Postdated",
        "Invalid",
        "Renewable",
        "Initial",
        "Pre-authent",
        "Opt-hardware-auth",
        "Transited-policy-checked",
        "Ok-as-delegate",
        "Request-anonymous",
        "Name-canonicalize",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Unused",
        "Disable-transited-check",
        "Renewable-ok",
        "Enc-tkt-in-skey",
        "Unused",
        "Renew",
        "Validate"];

    // Kerberos Encryption Types
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4768
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4768
    var ticketEncryptionTypes = {
        "0x1": "DES-CBC-CRC",
        "0x3": "DES-CBC-MD5",
        "0x11": "AES128-CTS-HMAC-SHA1-96",
        "0x12": "AES256-CTS-HMAC-SHA1-96",
        "0x17": "RC4-HMAC",
        "0x18": "RC4-HMAC-EXP",
        "0xffffffff": "FAIL",
   };

    // Kerberos Result Status Codes
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4768
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4768
    var kerberosTktStatusCodes = {
        "0x0": "KDC_ERR_NONE",
        "0x1": "KDC_ERR_NAME_EXP",
        "0x2": "KDC_ERR_SERVICE_EXP",
        "0x3": "KDC_ERR_BAD_PVNO",
        "0x4": "KDC_ERR_C_OLD_MAST_KVNO",
        "0x5": "KDC_ERR_S_OLD_MAST_KVNO",
        "0x6": "KDC_ERR_C_PRINCIPAL_UNKNOWN",
        "0x7": "KDC_ERR_S_PRINCIPAL_UNKNOWN",
        "0x8": "KDC_ERR_PRINCIPAL_NOT_UNIQUE",
        "0x9": "KDC_ERR_NULL_KEY",
        "0xA": "KDC_ERR_CANNOT_POSTDATE",
        "0xB": "KDC_ERR_NEVER_VALID",
        "0xC": "KDC_ERR_POLICY",
        "0xD": "KDC_ERR_BADOPTION",
        "0xE": "KDC_ERR_ETYPE_NOTSUPP",
        "0xF": "KDC_ERR_SUMTYPE_NOSUPP",
        "0x10": "KDC_ERR_PADATA_TYPE_NOSUPP",
        "0x11": "KDC_ERR_TRTYPE_NO_SUPP",
        "0x12": "KDC_ERR_CLIENT_REVOKED",
        "0x13": "KDC_ERR_SERVICE_REVOKED",
        "0x14": "KDC_ERR_TGT_REVOKED",
        "0x15": "KDC_ERR_CLIENT_NOTYET",
        "0x16": "KDC_ERR_SERVICE_NOTYET",
        "0x17": "KDC_ERR_KEY_EXPIRED",
        "0x18": "KDC_ERR_PREAUTH_FAILED",
        "0x19": "KDC_ERR_PREAUTH_REQUIRED",
        "0x1A": "KDC_ERR_SERVER_NOMATCH",
        "0x1B": "KDC_ERR_MUST_USE_USER2USER",
        "0x1F": "KRB_AP_ERR_BAD_INTEGRITY",
        "0x20": "KRB_AP_ERR_TKT_EXPIRED",
        "0x21": "KRB_AP_ERR_TKT_NYV",
        "0x22": "KRB_AP_ERR_REPEAT",
        "0x23": "KRB_AP_ERR_NOT_US",
        "0x24": "KRB_AP_ERR_BADMATCH",
        "0x25": "KRB_AP_ERR_SKEW",
        "0x26": "KRB_AP_ERR_BADADDR",
        "0x27": "KRB_AP_ERR_BADVERSION",
        "0x28": "KRB_AP_ERR_MSG_TYPE",
        "0x29": "KRB_AP_ERR_MODIFIED",
        "0x2A": "KRB_AP_ERR_BADORDER",
        "0x2C": "KRB_AP_ERR_BADKEYVER",
        "0x2D": "KRB_AP_ERR_NOKEY",
        "0x2E": "KRB_AP_ERR_MUT_FAIL",
        "0x2F": "KRB_AP_ERR_BADDIRECTION",
        "0x30": "KRB_AP_ERR_METHOD",
        "0x31": "KRB_AP_ERR_BADSEQ",
        "0x32": "KRB_AP_ERR_INAPP_CKSUM",
        "0x33": "KRB_AP_PATH_NOT_ACCEPTED",
        "0x34": "KRB_ERR_RESPONSE_TOO_BIG",
        "0x3C": "KRB_ERR_GENERIC",
        "0x3D": "KRB_ERR_FIELD_TOOLONG",
        "0x3E": "KDC_ERR_CLIENT_NOT_TRUSTED",
        "0x3F": "KDC_ERR_KDC_NOT_TRUSTED",
        "0x40": "KDC_ERR_INVALID_SIG",
        "0x41": "KDC_ERR_KEY_TOO_WEAK",
        "0x42": "KRB_AP_ERR_USER_TO_USER_REQUIRED",
        "0x43": "KRB_AP_ERR_NO_TGT",
        "0x44": "KDC_ERR_WRONG_REALM",
    };

    // event.category, event.type, event.action
    var eventActionTypes = {
        "1100": ["process","end","logging-service-shutdown"],
        "1102": ["iam", "admin", "audit-log-cleared"],
        "1104": ["iam","admin","logging-full"],
        "1105": ["iam","admin","auditlog-archieved"],
        "1108": ["iam","admin","logging-processing-error"],
        "4624": ["authentication","start","logged-in"],
        "4625": ["authentication","start","logon-failed"],
        "4634": ["authentication","end","logged-out"],
        "4647": ["authentication","end","logged-out"],
        "4648": ["authentication","start","logged-in-explicit"],
        "4672": ["iam","admin","logged-in-special"],
        "4673": ["iam","admin","privileged-service-called"],
        "4674": ["iam","admin","privileged-operation"],
        "4688": ["process","start","created-process"],
        "4689": ["process", "end", "exited-process"],
        "4697": ["iam","admin","service-installed"],
        "4698": ["iam","creation","scheduled-task-created"],
        "4699": ["iam","deletion","scheduled-task-deleted"],
        "4700": ["iam","change","scheduled-task-enabled"],
        "4701": ["iam","change","scheduled-task-disabled"],
        "4702": ["iam","change","scheduled-task-updated"],
        "4719": ["iam","admin","changed-audit-config"],
        "4720": ["iam","creation","added-user-account"],
        "4722": ["iam","creation","enabled-user-account"],
        "4723": ["iam","change","changed-password"],
        "4724": ["iam","change","reset-password"],
        "4725": ["iam","deletion","disabled-user-account"],
        "4726": ["iam","deletion","deleted-user-account"],
        "4727": ["iam","creation","added-group-account"],
        "4728": ["iam","change","added-member-to-group"],
        "4729": ["iam","change","removed-member-from-group"],
        "4730": ["iam","deletion","deleted-group-account"],
        "4731": ["iam","creation","added-group-account"],
        "4732": ["iam","change","added-member-to-group"],
        "4733": ["iam","change","removed-member-from-group"],
        "4734": ["iam","deletion","deleted-group-account"],
        "4735": ["iam","change","modified-group-account"],
        "4737": ["iam","change","modified-group-account"],
        "4738": ["iam","change","modified-user-account"],
        "4740": ["iam","change","locked-out-user-account"],
        "4741": ["iam","creation","added-computer-account"],
        "4742": ["iam","change","changed-computer-account"],
        "4743": ["iam","deletion","deleted-computer-account"],
        "4744": ["iam","creation","added-distribution-group-account"],
        "4745": ["iam","change","changed-distribution-group-account"],
        "4746": ["iam","change","added-member-to-distribution-group"],
        "4747": ["iam","change","removed-member-from-distribution-group"],
        "4748": ["iam","deletion","deleted-distribution-group-account"],
        "4749": ["iam","creation","added-distribution-group-account"],
        "4750": ["iam","change","changed-distribution-group-account"],
        "4751": ["iam","change","added-member-to-distribution-group"],
        "4752": ["iam","change","removed-member-from-distribution-group"],
        "4753": ["iam","deletion","deleted-distribution-group-account"],
        "4754": ["iam","creation","added-group-account"],
        "4755": ["iam","change","modified-group-account"],
        "4756": ["iam","change","added-member-to-group"],
        "4757": ["iam","change","removed-member-from-group"],
        "4758": ["iam","deletion","deleted-group-account"],
        "4759": ["iam","creation","added-distribution-group-account"],
        "4760": ["iam","change","changed-distribution-group-account"],
        "4761": ["iam","change","added-member-to-distribution-group"],
        "4762": ["iam","change","removed-member-from-distribution-group"],
        "4763": ["iam","deletion","deleted-distribution-group-account"],
        "4764": ["iam","change","type-changed-group-account"],
        "4767": ["iam","change","unlocked-user-account"],
        "4768": ["authentication","start","kerberos-authentication-ticket-requested"],
        "4769": ["authentication","start","kerberos-service-ticket-requested"],
        "4770": ["authentication","start","kerberos-service-ticket-renewed"],
        "4771": ["authentication","start","kerberos-preauth-failed"],
        "4776": ["authentication","start","credential-validated"],
        "4778": ["authentication","start","session-reconnected"],
        "4779": ["authentication","end","session-disconnected"],
        "4781": ["iam","change","renamed-user-account","dummy"],
        "4798": ["iam","info","group-membership-enumerated"],
        "4799": ["iam","info","user-member-enumerated","dummy"],
        "4964": ["iam","admin","logged-in-special"],
    };


    // Audit Policy Changes Table
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4719
    var auditActions = {
        "8448": "Success Removed",
        "8450": "Failure Removed",
        "8449": "Success Added",
        "8451": "Failure Added",
    };

    // Services Types
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4697
    var serviceTypes = {
        "0x1": "Kernel Driver",
        "0x2": "File System Driver",
        "0x8": "Recognizer Driver",
        "0x10": "Win32 Own Process",
        "0x20": "Win32 Share Process",
        "0x110": "Interactive Own Process",
        "0x120": "Interactive Share Process",
    };


    // Audit Categories Description
    // https://docs.microsoft.com/en-us/openspecs/windows_protocols/ms-gpac/77878370-0712-47cd-997d-b07053429f6d
    var auditDescription = {
        "0CCE9210-69AE-11D9-BED3-505054503030":["Security State Change", "System"],
        "0CCE9211-69AE-11D9-BED3-505054503030":["Security System Extension", "System"],
        "0CCE9212-69AE-11D9-BED3-505054503030":["System Integrity", "System"],
        "0CCE9213-69AE-11D9-BED3-505054503030":["IPsec Driver", "System"],
        "0CCE9214-69AE-11D9-BED3-505054503030":["Other System Events", "System"],
        "0CCE9215-69AE-11D9-BED3-505054503030":["Logon", "Logon/Logoff"],
        "0CCE9216-69AE-11D9-BED3-505054503030":["Logoff","Logon/Logoff"],
        "0CCE9217-69AE-11D9-BED3-505054503030":["Account Lockout","Logon/Logoff"],
        "0CCE9218-69AE-11D9-BED3-505054503030":["IPsec Main Mode","Logon/Logoff"],
        "0CCE9219-69AE-11D9-BED3-505054503030":["IPsec Quick Mode","Logon/Logoff"],
        "0CCE921A-69AE-11D9-BED3-505054503030":["IPsec Extended Mode","Logon/Logoff"],
        "0CCE921B-69AE-11D9-BED3-505054503030":["Special Logon","Logon/Logoff"],
        "0CCE921C-69AE-11D9-BED3-505054503030":["Other Logon/Logoff Events","Logon/Logoff"],
        "0CCE9243-69AE-11D9-BED3-505054503030":["Network Policy Server","Logon/Logoff"],
        "0CCE9247-69AE-11D9-BED3-505054503030":["User / Device Claims","Logon/Logoff"],
        "0CCE921D-69AE-11D9-BED3-505054503030":["File System","Object Access"],
        "0CCE921E-69AE-11D9-BED3-505054503030":["Registry","Object Access"],
        "0CCE921F-69AE-11D9-BED3-505054503030":["Kernel Object","Object Access"],
        "0CCE9220-69AE-11D9-BED3-505054503030":["SAM","Object Access"],
        "0CCE9221-69AE-11D9-BED3-505054503030":["Certification Services","Object Access"],
        "0CCE9222-69AE-11D9-BED3-505054503030":["Application Generated","Object Access"],
        "0CCE9223-69AE-11D9-BED3-505054503030":["Handle Manipulation","Object Access"],
        "0CCE9224-69AE-11D9-BED3-505054503030":["File Share","Object Access"],
        "0CCE9225-69AE-11D9-BED3-505054503030":["Filtering Platform Packet Drop","Object Access"],
        "0CCE9226-69AE-11D9-BED3-505054503030":["Filtering Platform Connection ","Object Access"],
        "0CCE9227-69AE-11D9-BED3-505054503030":["Other Object Access Events","Object Access"],
        "0CCE9244-69AE-11D9-BED3-505054503030":["Detailed File Share","Object Access"],
        "0CCE9245-69AE-11D9-BED3-505054503030":["Removable Storage","Object Access"],
        "0CCE9246-69AE-11D9-BED3-505054503030":["Central Policy Staging","Object Access"],
        "0CCE9228-69AE-11D9-BED3-505054503030":["Sensitive Privilege Use","Privilege Use"],
        "0CCE9229-69AE-11D9-BED3-505054503030":["Non Sensitive Privilege Use","Privilege Use"],
        "0CCE922A-69AE-11D9-BED3-505054503030":["Other Privilege Use Events","Privilege Use"],
        "0CCE922B-69AE-11D9-BED3-505054503030":["Process Creation","Detailed Tracking"],
        "0CCE922C-69AE-11D9-BED3-505054503030":["Process Termination","Detailed Tracking"],
        "0CCE922D-69AE-11D9-BED3-505054503030":["DPAPI Activity","Detailed Tracking"],
        "0CCE922E-69AE-11D9-BED3-505054503030":["RPC Events","Detailed Tracking"],
        "0CCE9248-69AE-11D9-BED3-505054503030":["Plug and Play Events","Detailed Tracking"],
        "0CCE922F-69AE-11D9-BED3-505054503030":["Audit Policy Change","Policy Change"],
        "0CCE9230-69AE-11D9-BED3-505054503030":["Authentication Policy Change","Policy Change"],
        "0CCE9231-69AE-11D9-BED3-505054503030":["Authorization Policy Change","Policy Change"],
        "0CCE9232-69AE-11D9-BED3-505054503030":["MPSSVC Rule-Level Policy Change","Policy Change"],
        "0CCE9233-69AE-11D9-BED3-505054503030":["Filtering Platform Policy Change","Policy Change"],
        "0CCE9234-69AE-11D9-BED3-505054503030":["Other Policy Change Events","Policy Change"],
        "0CCE9235-69AE-11D9-BED3-505054503030":["User Account Management","Account Management"],
        "0CCE9236-69AE-11D9-BED3-505054503030":["Computer Account Management","Account Management"],
        "0CCE9237-69AE-11D9-BED3-505054503030":["Security Group Management","Account Management"],
        "0CCE9238-69AE-11D9-BED3-505054503030":["Distribution Group Management","Account Management"],
        "0CCE9239-69AE-11D9-BED3-505054503030":["Application Group Management","Account Management"],
        "0CCE923A-69AE-11D9-BED3-505054503030":["Other Account Management Events","Account Management"],
        "0CCE923B-69AE-11D9-BED3-505054503030":["Directory Service Access","Account Management"],
        "0CCE923C-69AE-11D9-BED3-505054503030":["Directory Service Changes","Account Management"],
        "0CCE923D-69AE-11D9-BED3-505054503030":["Directory Service Replication","Account Management"],
        "0CCE923E-69AE-11D9-BED3-505054503030":["Detailed Directory Service Replication","Account Management"],
        "0CCE923F-69AE-11D9-BED3-505054503030":["Credential Validation","Account Logon"],
        "0CCE9240-69AE-11D9-BED3-505054503030":["Kerberos Service Ticket Operations","Account Logon"],
        "0CCE9241-69AE-11D9-BED3-505054503030":["Other Account Logon Events","Account Logon"],
        "0CCE9242-69AE-11D9-BED3-505054503030":["Kerberos Authentication Service","Account Logon"],
    };


    // Descriptions of failure status codes.
    // https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4625
   //  https://docs.microsoft.com/en-us/windows/security/threat-protection/auditing/event-4776
    var logonFailureStatus = {
        "0xc000005e": "There are currently no logon servers available to service the logon request.",
        "0xc0000064": "User logon with misspelled or bad user account",
        "0xc000006a": "User logon with misspelled or bad password",
        "0xc000006d": "This is either due to a bad username or authentication information",
        "0xc000006e": "Unknown user name or bad password.",
        "0xc000006f": "User logon outside authorized hours",
        "0xc0000070": "User logon from unauthorized workstation",
        "0xc0000071": "User logon with expired password",
        "0xc0000072": "User logon to account disabled by administrator",
        "0xc00000dc": "Indicates the Sam Server was in the wrong state to perform the desired operation.",
        "0xc0000133": "Clocks between DC and other computer too far out of sync",
        "0xc000015b": "The user has not been granted the requested logon type (aka logon right) at this machine",
        "0xc000018c": "The logon request failed because the trust relationship between the primary domain and the trusted domain failed.",
        "0xc0000192": "An attempt was made to logon, but the Netlogon service was not started.",
        "0xc0000193": "User logon with expired account",
        "0xc0000224": "User is required to change password at next logon",
        "0xc0000225": "Evidently a bug in Windows and not a risk",
        "0xc0000234": "User logon with account locked",
        "0xc00002ee": "Failure Reason: An Error occurred during Logon",
        "0xc0000413": "Logon Failure: The machine you are logging onto is protected by an authentication firewall. The specified account is not allowed to authenticate to the machine.",
        "0xc0000371": "The local account store does not contain secret material for the specified account",
        "0x0": "Status OK.",
    };

    // Message table extracted from msobjs.dll on Windows 2019.
    // https://gist.github.com/andrewkroh/665dca0682bd0e4daf194ab291694012
    var msobjsMessageTable = {
        "279": "Undefined Access (no effect) Bit 7",
        "1536": "Unused message ID",
        "1537": "DELETE",
        "1538": "READ_CONTROL",
        "1539": "WRITE_DAC",
        "1540": "WRITE_OWNER",
        "1541": "SYNCHRONIZE",
        "1542": "ACCESS_SYS_SEC",
        "1543": "MAX_ALLOWED",
        "1552": "Unknown specific access (bit 0)",
        "1553": "Unknown specific access (bit 1)",
        "1554": "Unknown specific access (bit 2)",
        "1555": "Unknown specific access (bit 3)",
        "1556": "Unknown specific access (bit 4)",
        "1557": "Unknown specific access (bit 5)",
        "1558": "Unknown specific access (bit 6)",
        "1559": "Unknown specific access (bit 7)",
        "1560": "Unknown specific access (bit 8)",
        "1561": "Unknown specific access (bit 9)",
        "1562": "Unknown specific access (bit 10)",
        "1563": "Unknown specific access (bit 11)",
        "1564": "Unknown specific access (bit 12)",
        "1565": "Unknown specific access (bit 13)",
        "1566": "Unknown specific access (bit 14)",
        "1567": "Unknown specific access (bit 15)",
        "1601": "Not used",
        "1603": "Assign Primary Token Privilege",
        "1604": "Lock Memory Privilege",
        "1605": "Increase Memory Quota Privilege",
        "1606": "Unsolicited Input Privilege",
        "1607": "Trusted Computer Base Privilege",
        "1608": "Security Privilege",
        "1609": "Take Ownership Privilege",
        "1610": "Load/Unload Driver Privilege",
        "1611": "Profile System Privilege",
        "1612": "Set System Time Privilege",
        "1613": "Profile Single Process Privilege",
        "1614": "Increment Base Priority Privilege",
        "1615": "Create Pagefile Privilege",
        "1616": "Create Permanent Object Privilege",
        "1617": "Backup Privilege",
        "1618": "Restore From Backup Privilege",
        "1619": "Shutdown System Privilege",
        "1620": "Debug Privilege",
        "1621": "View or Change Audit Log Privilege",
        "1622": "Change Hardware Environment Privilege",
        "1623": "Change Notify (and Traverse) Privilege",
        "1624": "Remotely Shut System Down Privilege",
        "1792": "<value changed",
        "1793": "<value not set>",
        "1794": "<never>",
        "1795": "Enabled",
        "1796": "Disabled",
        "1797": "All",
        "1798": "None",
        "1799": "Audit Policy query/set API Operation",
        "1800": "<Value change auditing for this registry type is not supported>",
        "1801": "Granted by",
        "1802": "Denied by",
        "1803": "Denied by Integrity Policy check",
        "1804": "Granted by Ownership",
        "1805": "Not granted",
        "1806": "Granted by NULL DACL",
        "1807": "Denied by Empty DACL",
        "1808": "Granted by NULL Security Descriptor",
        "1809": "Unknown or unchecked",
        "1810": "Not granted due to missing",
        "1811": "Granted by ACE on parent folder",
        "1812": "Denied by ACE on parent folder",
        "1813": "Granted by Central Access Rule",
        "1814": "NOT Granted by Central Access Rule",
        "1815": "Granted by parent folder's Central Access Rule",
        "1816": "NOT Granted by parent folder's Central Access Rule",
        "1817": "Unknown Type",
        "1818": "String",
        "1819": "Unsigned 64-bit Integer",
        "1820": "64-bit Integer",
        "1821": "FQBN",
        "1822": "Blob",
        "1823": "Sid",
        "1824": "Boolean",
        "1825": "TRUE",
        "1826": "FALSE",
        "1827": "Invalid",
        "1828": "an ACE too long to display",
        "1829": "a Security Descriptor too long to display",
        "1830": "Not granted to AppContainers",
        "1831": "...",
        "1832": "Identification",
        "1833": "Impersonation",
        "1840": "Delegation",
        "1841": "Denied by Process Trust Label ACE",
        "1842": "Yes",
        "1843": "No",
        "1844": "System",
        "1845": "Not Available",
        "1846": "Default",
        "1847": "DisallowMmConfig",
        "1848": "Off",
        "1849": "Auto",
        "1872": "REG_NONE",
        "1873": "REG_SZ",
        "1874": "REG_EXPAND_SZ",
        "1875": "REG_BINARY",
        "1876": "REG_DWORD",
        "1877": "REG_DWORD_BIG_ENDIAN",
        "1878": "REG_LINK",
        "1879": "REG_MULTI_SZ (New lines are replaced with *. A * is replaced with **)",
        "1880": "REG_RESOURCE_LIST",
        "1881": "REG_FULL_RESOURCE_DESCRIPTOR",
        "1882": "REG_RESOURCE_REQUIREMENTS_LIST",
        "1883": "REG_QWORD",
        "1904": "New registry value created",
        "1905": "Existing registry value modified",
        "1906": "Registry value deleted",
        "1920": "Sunday",
        "1921": "Monday",
        "1922": "Tuesday",
        "1923": "Wednesday",
        "1924": "Thursday",
        "1925": "Friday",
        "1926": "Saturday",
        "1936": "TokenElevationTypeDefault (1)",
        "1937": "TokenElevationTypeFull (2)",
        "1938": "TokenElevationTypeLimited (3)",
        "2048": "Account Enabled",
        "2049": "Home Directory Required' - Disabled",
        "2050": "Password Not Required' - Disabled",
        "2051": "Temp Duplicate Account' - Disabled",
        "2052": "Normal Account' - Disabled",
        "2053": "MNS Logon Account' - Disabled",
        "2054": "Interdomain Trust Account' - Disabled",
        "2055": "Workstation Trust Account' - Disabled",
        "2056": "Server Trust Account' - Disabled",
        "2057": "Don't Expire Password' - Disabled",
        "2058": "Account Unlocked",
        "2059": "Encrypted Text Password Allowed' - Disabled",
        "2060": "Smartcard Required' - Disabled",
        "2061": "Trusted For Delegation' - Disabled",
        "2062": "Not Delegated' - Disabled",
        "2063": "Use DES Key Only' - Disabled",
        "2064": "Don't Require Preauth' - Disabled",
        "2065": "Password Expired' - Disabled",
        "2066": "Trusted To Authenticate For Delegation' - Disabled",
        "2067": "Exclude Authorization Information' - Disabled",
        "2068": "Undefined UserAccountControl Bit 20' - Disabled",
        "2069": "Protect Kerberos Service Tickets with AES Keys' - Disabled",
        "2070": "Undefined UserAccountControl Bit 22' - Disabled",
        "2071": "Undefined UserAccountControl Bit 23' - Disabled",
        "2072": "Undefined UserAccountControl Bit 24' - Disabled",
        "2073": "Undefined UserAccountControl Bit 25' - Disabled",
        "2074": "Undefined UserAccountControl Bit 26' - Disabled",
        "2075": "Undefined UserAccountControl Bit 27' - Disabled",
        "2076": "Undefined UserAccountControl Bit 28' - Disabled",
        "2077": "Undefined UserAccountControl Bit 29' - Disabled",
        "2078": "Undefined UserAccountControl Bit 30' - Disabled",
        "2079": "Undefined UserAccountControl Bit 31' - Disabled",
        "2080": "Account Disabled",
        "2081": "Home Directory Required' - Enabled",
        "2082": "Password Not Required' - Enabled",
        "2083": "Temp Duplicate Account' - Enabled",
        "2084": "Normal Account' - Enabled",
        "2085": "MNS Logon Account' - Enabled",
        "2086": "Interdomain Trust Account' - Enabled",
        "2087": "Workstation Trust Account' - Enabled",
        "2088": "Server Trust Account' - Enabled",
        "2089": "Don't Expire Password' - Enabled",
        "2090": "Account Locked",
        "2091": "Encrypted Text Password Allowed' - Enabled",
        "2092": "Smartcard Required' - Enabled",
        "2093": "Trusted For Delegation' - Enabled",
        "2094": "Not Delegated' - Enabled",
        "2095": "Use DES Key Only' - Enabled",
        "2096": "Don't Require Preauth' - Enabled",
        "2097": "Password Expired' - Enabled",
        "2098": "Trusted To Authenticate For Delegation' - Enabled",
        "2099": "Exclude Authorization Information' - Enabled",
        "2100": "Undefined UserAccountControl Bit 20' - Enabled",
        "2101": "Protect Kerberos Service Tickets with AES Keys' - Enabled",
        "2102": "Undefined UserAccountControl Bit 22' - Enabled",
        "2103": "Undefined UserAccountControl Bit 23' - Enabled",
        "2104": "Undefined UserAccountControl Bit 24' - Enabled",
        "2105": "Undefined UserAccountControl Bit 25' - Enabled",
        "2106": "Undefined UserAccountControl Bit 26' - Enabled",
        "2107": "Undefined UserAccountControl Bit 27' - Enabled",
        "2108": "Undefined UserAccountControl Bit 28' - Enabled",
        "2109": "Undefined UserAccountControl Bit 29' - Enabled",
        "2110": "Undefined UserAccountControl Bit 30' - Enabled",
        "2111": "Undefined UserAccountControl Bit 31' - Enabled",
        "2304": "An Error occured during Logon.",
        "2305": "The specified user account has expired.",
        "2306": "The NetLogon component is not active.",
        "2307": "Account locked out.",
        "2308": "The user has not been granted the requested logon type at this machine.",
        "2309": "The specified account's password has expired.",
        "2310": "Account currently disabled.",
        "2311": "Account logon time restriction violation.",
        "2312": "User not allowed to logon at this computer.",
        "2313": "Unknown user name or bad password.",
        "2314": "Domain sid inconsistent.",
        "2315": "Smartcard logon is required and was not used.",
        "2432": "Not Available.",
        "2436": "Random number generator failure.",
        "2437": "Random number generation failed FIPS-140 pre-hash check.",
        "2438": "Failed to zero secret data.",
        "2439": "Key failed pair wise consistency check.",
        "2448": "Failed to unprotect persistent cryptographic key.",
        "2449": "Key export checks failed.",
        "2450": "Validation of public key failed.",
        "2451": "Signature verification failed.",
        "2456": "Open key file.",
        "2457": "Delete key file.",
        "2458": "Read persisted key from file.",
        "2459": "Write persisted key to file.",
        "2464": "Export of persistent cryptographic key.",
        "2465": "Import of persistent cryptographic key.",
        "2480": "Open Key.",
        "2481": "Create Key.",
        "2482": "Delete Key.",
        "2483": "Encrypt.",
        "2484": "Decrypt.",
        "2485": "Sign hash.",
        "2486": "Secret agreement.",
        "2487": "Domain settings",
        "2488": "Local settings",
        "2489": "Add provider.",
        "2490": "Remove provider.",
        "2491": "Add context.",
        "2492": "Remove context.",
        "2493": "Add function.",
        "2494": "Remove function.",
        "2495": "Add function provider.",
        "2496": "Remove function provider.",
        "2497": "Add function property.",
        "2498": "Remove function property.",
        "2499": "Machine key.",
        "2500": "User key.",
        "2501": "Key Derivation.",
        "4352": "Device Access Bit 0",
        "4353": "Device Access Bit 1",
        "4354": "Device Access Bit 2",
        "4355": "Device Access Bit 3",
        "4356": "Device Access Bit 4",
        "4357": "Device Access Bit 5",
        "4358": "Device Access Bit 6",
        "4359": "Device Access Bit 7",
        "4360": "Device Access Bit 8",
        "4361": "Undefined Access (no effect) Bit 9",
        "4362": "Undefined Access (no effect) Bit 10",
        "4363": "Undefined Access (no effect) Bit 11",
        "4364": "Undefined Access (no effect) Bit 12",
        "4365": "Undefined Access (no effect) Bit 13",
        "4366": "Undefined Access (no effect) Bit 14",
        "4367": "Undefined Access (no effect) Bit 15",
        "4368": "Query directory",
        "4369": "Traverse",
        "4370": "Create object in directory",
        "4371": "Create sub-directory",
        "4372": "Undefined Access (no effect) Bit 4",
        "4373": "Undefined Access (no effect) Bit 5",
        "4374": "Undefined Access (no effect) Bit 6",
        "4375": "Undefined Access (no effect) Bit 7",
        "4376": "Undefined Access (no effect) Bit 8",
        "4377": "Undefined Access (no effect) Bit 9",
        "4378": "Undefined Access (no effect) Bit 10",
        "4379": "Undefined Access (no effect) Bit 11",
        "4380": "Undefined Access (no effect) Bit 12",
        "4381": "Undefined Access (no effect) Bit 13",
        "4382": "Undefined Access (no effect) Bit 14",
        "4383": "Undefined Access (no effect) Bit 15",
        "4384": "Query event state",
        "4385": "Modify event state",
        "4386": "Undefined Access (no effect) Bit 2",
        "4387": "Undefined Access (no effect) Bit 3",
        "4388": "Undefined Access (no effect) Bit 4",
        "4389": "Undefined Access (no effect) Bit 5",
        "4390": "Undefined Access (no effect) Bit 6",
        "4391": "Undefined Access (no effect) Bit 7",
        "4392": "Undefined Access (no effect) Bit 8",
        "4393": "Undefined Access (no effect) Bit 9",
        "4394": "Undefined Access (no effect) Bit 10",
        "4395": "Undefined Access (no effect) Bit 11",
        "4396": "Undefined Access (no effect) Bit 12",
        "4397": "Undefined Access (no effect) Bit 13",
        "4398": "Undefined Access (no effect) Bit 14",
        "4399": "Undefined Access (no effect) Bit 15",
        "4416": "ReadData (or ListDirectory)",
        "4417": "WriteData (or AddFile)",
        "4418": "AppendData (or AddSubdirectory or CreatePipeInstance)",
        "4419": "ReadEA",
        "4420": "WriteEA",
        "4421": "Execute/Traverse",
        "4422": "DeleteChild",
        "4423": "ReadAttributes",
        "4424": "WriteAttributes",
        "4425": "Undefined Access (no effect) Bit 9",
        "4426": "Undefined Access (no effect) Bit 10",
        "4427": "Undefined Access (no effect) Bit 11",
        "4428": "Undefined Access (no effect) Bit 12",
        "4429": "Undefined Access (no effect) Bit 13",
        "4430": "Undefined Access (no effect) Bit 14",
        "4431": "Undefined Access (no effect) Bit 15",
        "4432": "Query key value",
        "4433": "Set key value",
        "4434": "Create sub-key",
        "4435": "Enumerate sub-keys",
        "4436": "Notify about changes to keys",
        "4437": "Create Link",
        "4438": "Undefined Access (no effect) Bit 6",
        "4439": "Undefined Access (no effect) Bit 7",
        "4440": "Enable 64(or 32) bit application to open 64 bit key",
        "4441": "Enable 64(or 32) bit application to open 32 bit key",
        "4442": "Undefined Access (no effect) Bit 10",
        "4443": "Undefined Access (no effect) Bit 11",
        "4444": "Undefined Access (no effect) Bit 12",
        "4445": "Undefined Access (no effect) Bit 13",
        "4446": "Undefined Access (no effect) Bit 14",
        "4447": "Undefined Access (no effect) Bit 15",
        "4448": "Query mutant state",
        "4449": "Undefined Access (no effect) Bit 1",
        "4450": "Undefined Access (no effect) Bit 2",
        "4451": "Undefined Access (no effect) Bit 3",
        "4452": "Undefined Access (no effect) Bit 4",
        "4453": "Undefined Access (no effect) Bit 5",
        "4454": "Undefined Access (no effect) Bit 6",
        "4455": "Undefined Access (no effect) Bit 7",
        "4456": "Undefined Access (no effect) Bit 8",
        "4457": "Undefined Access (no effect) Bit 9",
        "4458": "Undefined Access (no effect) Bit 10",
        "4459": "Undefined Access (no effect) Bit 11",
        "4460": "Undefined Access (no effect) Bit 12",
        "4461": "Undefined Access (no effect) Bit 13",
        "4462": "Undefined Access (no effect) Bit 14",
        "4463": "Undefined Access (no effect) Bit 15",
        "4464": "Communicate using port",
        "4465": "Undefined Access (no effect) Bit 1",
        "4466": "Undefined Access (no effect) Bit 2",
        "4467": "Undefined Access (no effect) Bit 3",
        "4468": "Undefined Access (no effect) Bit 4",
        "4469": "Undefined Access (no effect) Bit 5",
        "4470": "Undefined Access (no effect) Bit 6",
        "4471": "Undefined Access (no effect) Bit 7",
        "4472": "Undefined Access (no effect) Bit 8",
        "4473": "Undefined Access (no effect) Bit 9",
        "4474": "Undefined Access (no effect) Bit 10",
        "4475": "Undefined Access (no effect) Bit 11",
        "4476": "Undefined Access (no effect) Bit 12",
        "4477": "Undefined Access (no effect) Bit 13",
        "4478": "Undefined Access (no effect) Bit 14",
        "4479": "Undefined Access (no effect) Bit 15",
        "4480": "Force process termination",
        "4481": "Create new thread in process",
        "4482": "Set process session ID",
        "4483": "Perform virtual memory operation",
        "4484": "Read from process memory",
        "4485": "Write to process memory",
        "4486": "Duplicate handle into or out of process",
        "4487": "Create a subprocess of process",
        "4488": "Set process quotas",
        "4489": "Set process information",
        "4490": "Query process information",
        "4491": "Set process termination port",
        "4492": "Undefined Access (no effect) Bit 12",
        "4493": "Undefined Access (no effect) Bit 13",
        "4494": "Undefined Access (no effect) Bit 14",
        "4495": "Undefined Access (no effect) Bit 15",
        "4496": "Control profile",
        "4497": "Undefined Access (no effect) Bit 1",
        "4498": "Undefined Access (no effect) Bit 2",
        "4499": "Undefined Access (no effect) Bit 3",
        "4500": "Undefined Access (no effect) Bit 4",
        "4501": "Undefined Access (no effect) Bit 5",
        "4502": "Undefined Access (no effect) Bit 6",
        "4503": "Undefined Access (no effect) Bit 7",
        "4504": "Undefined Access (no effect) Bit 8",
        "4505": "Undefined Access (no effect) Bit 9",
        "4506": "Undefined Access (no effect) Bit 10",
        "4507": "Undefined Access (no effect) Bit 11",
        "4508": "Undefined Access (no effect) Bit 12",
        "4509": "Undefined Access (no effect) Bit 13",
        "4510": "Undefined Access (no effect) Bit 14",
        "4511": "Undefined Access (no effect) Bit 15",
        "4512": "Query section state",
        "4513": "Map section for write",
        "4514": "Map section for read",
        "4515": "Map section for execute",
        "4516": "Extend size",
        "4517": "Undefined Access (no effect) Bit 5",
        "4518": "Undefined Access (no effect) Bit 6",
        "4519": "Undefined Access (no effect) Bit 7",
        "4520": "Undefined Access (no effect) Bit 8",
        "4521": "Undefined Access (no effect) Bit 9",
        "4522": "Undefined Access (no effect) Bit 10",
        "4523": "Undefined Access (no effect) Bit 11",
        "4524": "Undefined Access (no effect) Bit 12",
        "4525": "Undefined Access (no effect) Bit 13",
        "4526": "Undefined Access (no effect) Bit 14",
        "4527": "Undefined Access (no effect) Bit 15",
        "4528": "Query semaphore state",
        "4529": "Modify semaphore state",
        "4530": "Undefined Access (no effect) Bit 2",
        "4531": "Undefined Access (no effect) Bit 3",
        "4532": "Undefined Access (no effect) Bit 4",
        "4533": "Undefined Access (no effect) Bit 5",
        "4534": "Undefined Access (no effect) Bit 6",
        "4535": "Undefined Access (no effect) Bit 7",
        "4536": "Undefined Access (no effect) Bit 8",
        "4537": "Undefined Access (no effect) Bit 9",
        "4538": "Undefined Access (no effect) Bit 10",
        "4539": "Undefined Access (no effect) Bit 11",
        "4540": "Undefined Access (no effect) Bit 12",
        "4541": "Undefined Access (no effect) Bit 13",
        "4542": "Undefined Access (no effect) Bit 14",
        "4543": "Undefined Access (no effect) Bit 15",
        "4544": "Use symbolic link",
        "4545": "Undefined Access (no effect) Bit 1",
        "4546": "Undefined Access (no effect) Bit 2",
        "4547": "Undefined Access (no effect) Bit 3",
        "4548": "Undefined Access (no effect) Bit 4",
        "4549": "Undefined Access (no effect) Bit 5",
        "4550": "Undefined Access (no effect) Bit 6",
        "4551": "Undefined Access (no effect) Bit 7",
        "4552": "Undefined Access (no effect) Bit 8",
        "4553": "Undefined Access (no effect) Bit 9",
        "4554": "Undefined Access (no effect) Bit 10",
        "4555": "Undefined Access (no effect) Bit 11",
        "4556": "Undefined Access (no effect) Bit 12",
        "4557": "Undefined Access (no effect) Bit 13",
        "4558": "Undefined Access (no effect) Bit 14",
        "4559": "Undefined Access (no effect) Bit 15",
        "4560": "Force thread termination",
        "4561": "Suspend or resume thread",
        "4562": "Send an alert to thread",
        "4563": "Get thread context",
        "4564": "Set thread context",
        "4565": "Set thread information",
        "4566": "Query thread information",
        "4567": "Assign a token to the thread",
        "4568": "Cause thread to directly impersonate another thread",
        "4569": "Directly impersonate this thread",
        "4570": "Undefined Access (no effect) Bit 10",
        "4571": "Undefined Access (no effect) Bit 11",
        "4572": "Undefined Access (no effect) Bit 12",
        "4573": "Undefined Access (no effect) Bit 13",
        "4574": "Undefined Access (no effect) Bit 14",
        "4575": "Undefined Access (no effect) Bit 15",
        "4576": "Query timer state",
        "4577": "Modify timer state",
        "4578": "Undefined Access (no effect) Bit 2",
        "4579": "Undefined Access (no effect) Bit 3",
        "4580": "Undefined Access (no effect) Bit 4",
        "4581": "Undefined Access (no effect) Bit 5",
        "4582": "Undefined Access (no effect) Bit 6",
        "4584": "Undefined Access (no effect) Bit 8",
        "4585": "Undefined Access (no effect) Bit 9",
        "4586": "Undefined Access (no effect) Bit 10",
        "4587": "Undefined Access (no effect) Bit 11",
        "4588": "Undefined Access (no effect) Bit 12",
        "4589": "Undefined Access (no effect) Bit 13",
        "4590": "Undefined Access (no effect) Bit 14",
        "4591": "Undefined Access (no effect) Bit 15",
        "4592": "AssignAsPrimary",
        "4593": "Duplicate",
        "4594": "Impersonate",
        "4595": "Query",
        "4596": "QuerySource",
        "4597": "AdjustPrivileges",
        "4598": "AdjustGroups",
        "4599": "AdjustDefaultDacl",
        "4600": "AdjustSessionID",
        "4601": "Undefined Access (no effect) Bit 9",
        "4602": "Undefined Access (no effect) Bit 10",
        "4603": "Undefined Access (no effect) Bit 11",
        "4604": "Undefined Access (no effect) Bit 12",
        "4605": "Undefined Access (no effect) Bit 13",
        "4606": "Undefined Access (no effect) Bit 14",
        "4607": "Undefined Access (no effect) Bit 15",
        "4608": "Create instance of object type",
        "4609": "Undefined Access (no effect) Bit 1",
        "4610": "Undefined Access (no effect) Bit 2",
        "4611": "Undefined Access (no effect) Bit 3",
        "4612": "Undefined Access (no effect) Bit 4",
        "4613": "Undefined Access (no effect) Bit 5",
        "4614": "Undefined Access (no effect) Bit 6",
        "4615": "Undefined Access (no effect) Bit 7",
        "4616": "Undefined Access (no effect) Bit 8",
        "4617": "Undefined Access (no effect) Bit 9",
        "4618": "Undefined Access (no effect) Bit 10",
        "4619": "Undefined Access (no effect) Bit 11",
        "4620": "Undefined Access (no effect) Bit 12",
        "4621": "Undefined Access (no effect) Bit 13",
        "4622": "Undefined Access (no effect) Bit 14",
        "4623": "Undefined Access (no effect) Bit 15",
        "4864": "Query State",
        "4865": "Modify State",
        "5120": "Channel read message",
        "5121": "Channel write message",
        "5122": "Channel query information",
        "5123": "Channel set information",
        "5124": "Undefined Access (no effect) Bit 4",
        "5125": "Undefined Access (no effect) Bit 5",
        "5126": "Undefined Access (no effect) Bit 6",
        "5127": "Undefined Access (no effect) Bit 7",
        "5128": "Undefined Access (no effect) Bit 8",
        "5129": "Undefined Access (no effect) Bit 9",
        "5130": "Undefined Access (no effect) Bit 10",
        "5131": "Undefined Access (no effect) Bit 11",
        "5132": "Undefined Access (no effect) Bit 12",
        "5133": "Undefined Access (no effect) Bit 13",
        "5134": "Undefined Access (no effect) Bit 14",
        "5135": "Undefined Access (no effect) Bit 15",
        "5136": "Assign process",
        "5137": "Set Attributes",
        "5138": "Query Attributes",
        "5139": "Terminate Job",
        "5140": "Set Security Attributes",
        "5141": "Undefined Access (no effect) Bit 5",
        "5142": "Undefined Access (no effect) Bit 6",
        "5143": "Undefined Access (no effect) Bit 7",
        "5144": "Undefined Access (no effect) Bit 8",
        "5145": "Undefined Access (no effect) Bit 9",
        "5146": "Undefined Access (no effect) Bit 10",
        "5147": "Undefined Access (no effect) Bit 11",
        "5148": "Undefined Access (no effect) Bit 12",
        "5149": "Undefined Access (no effect) Bit 13",
        "5150": "Undefined Access (no effect) Bit 14",
        "5151": "Undefined Access (no effect) Bit 15",
        "5376": "ConnectToServer",
        "5377": "ShutdownServer",
        "5378": "InitializeServer",
        "5379": "CreateDomain",
        "5380": "EnumerateDomains",
        "5381": "LookupDomain",
        "5382": "Undefined Access (no effect) Bit 6",
        "5383": "Undefined Access (no effect) Bit 7",
        "5384": "Undefined Access (no effect) Bit 8",
        "5385": "Undefined Access (no effect) Bit 9",
        "5386": "Undefined Access (no effect) Bit 10",
        "5387": "Undefined Access (no effect) Bit 11",
        "5388": "Undefined Access (no effect) Bit 12",
        "5389": "Undefined Access (no effect) Bit 13",
        "5390": "Undefined Access (no effect) Bit 14",
        "5391": "Undefined Access (no effect) Bit 15",
        "5392": "ReadPasswordParameters",
        "5393": "WritePasswordParameters",
        "5394": "ReadOtherParameters",
        "5395": "WriteOtherParameters",
        "5396": "CreateUser",
        "5397": "CreateGlobalGroup",
        "5398": "CreateLocalGroup",
        "5399": "GetLocalGroupMembership",
        "5400": "ListAccounts",
        "5401": "LookupIDs",
        "5402": "AdministerServer",
        "5403": "Undefined Access (no effect) Bit 11",
        "5404": "Undefined Access (no effect) Bit 12",
        "5405": "Undefined Access (no effect) Bit 13",
        "5406": "Undefined Access (no effect) Bit 14",
        "5407": "Undefined Access (no effect) Bit 15",
        "5408": "ReadInformation",
        "5409": "WriteAccount",
        "5410": "AddMember",
        "5411": "RemoveMember",
        "5412": "ListMembers",
        "5413": "Undefined Access (no effect) Bit 5",
        "5414": "Undefined Access (no effect) Bit 6",
        "5415": "Undefined Access (no effect) Bit 7",
        "5416": "Undefined Access (no effect) Bit 8",
        "5417": "Undefined Access (no effect) Bit 9",
        "5418": "Undefined Access (no effect) Bit 10",
        "5419": "Undefined Access (no effect) Bit 11",
        "5420": "Undefined Access (no effect) Bit 12",
        "5421": "Undefined Access (no effect) Bit 13",
        "5422": "Undefined Access (no effect) Bit 14",
        "5423": "Undefined Access (no effect) Bit 15",
        "5424": "AddMember",
        "5425": "RemoveMember",
        "5426": "ListMembers",
        "5427": "ReadInformation",
        "5428": "WriteAccount",
        "5429": "Undefined Access (no effect) Bit 5",
        "5430": "Undefined Access (no effect) Bit 6",
        "5431": "Undefined Access (no effect) Bit 7",
        "5432": "Undefined Access (no effect) Bit 8",
        "5433": "Undefined Access (no effect) Bit 9",
        "5434": "Undefined Access (no effect) Bit 10",
        "5435": "Undefined Access (no effect) Bit 11",
        "5436": "Undefined Access (no effect) Bit 12",
        "5437": "Undefined Access (no effect) Bit 13",
        "5438": "Undefined Access (no effect) Bit 14",
        "5439": "Undefined Access (no effect) Bit 15",
        "5440": "ReadGeneralInformation",
        "5441": "ReadPreferences",
        "5442": "WritePreferences",
        "5443": "ReadLogon",
        "5444": "ReadAccount",
        "5445": "WriteAccount",
        "5446": "ChangePassword (with knowledge of old password)",
        "5447": "SetPassword (without knowledge of old password)",
        "5448": "ListGroups",
        "5449": "ReadGroupMembership",
        "5450": "ChangeGroupMembership",
        "5451": "Undefined Access (no effect) Bit 11",
        "5452": "Undefined Access (no effect) Bit 12",
        "5453": "Undefined Access (no effect) Bit 13",
        "5454": "Undefined Access (no effect) Bit 14",
        "5455": "Undefined Access (no effect) Bit 15",
        "5632": "View non-sensitive policy information",
        "5633": "View system audit requirements",
        "5634": "Get sensitive policy information",
        "5635": "Modify domain trust relationships",
        "5636": "Create special accounts (for assignment of user rights)",
        "5637": "Create a secret object",
        "5638": "Create a privilege",
        "5639": "Set default quota limits",
        "5640": "Change system audit requirements",
        "5641": "Administer audit log attributes",
        "5642": "Enable/Disable LSA",
        "5643": "Lookup Names/SIDs",
        "5648": "Change secret value",
        "5649": "Query secret value",
        "5650": "Undefined Access (no effect) Bit 2",
        "5651": "Undefined Access (no effect) Bit 3",
        "5652": "Undefined Access (no effect) Bit 4",
        "5653": "Undefined Access (no effect) Bit 5",
        "5654": "Undefined Access (no effect) Bit 6",
        "5655": "Undefined Access (no effect) Bit 7",
        "5656": "Undefined Access (no effect) Bit 8",
        "5657": "Undefined Access (no effect) Bit 9",
        "5658": "Undefined Access (no effect) Bit 10",
        "5659": "Undefined Access (no effect) Bit 11",
        "5660": "Undefined Access (no effect) Bit 12",
        "5661": "Undefined Access (no effect) Bit 13",
        "5662": "Undefined Access (no effect) Bit 14",
        "5663": "Undefined Access (no effect) Bit 15",
        "5664": "Query trusted domain name/SID",
        "5665": "Retrieve the controllers in the trusted domain",
        "5666": "Change the controllers in the trusted domain",
        "5667": "Query the Posix ID offset assigned to the trusted domain",
        "5668": "Change the Posix ID offset assigned to the trusted domain",
        "5669": "Undefined Access (no effect) Bit 5",
        "5670": "Undefined Access (no effect) Bit 6",
        "5671": "Undefined Access (no effect) Bit 7",
        "5672": "Undefined Access (no effect) Bit 8",
        "5673": "Undefined Access (no effect) Bit 9",
        "5674": "Undefined Access (no effect) Bit 10",
        "5675": "Undefined Access (no effect) Bit 11",
        "5676": "Undefined Access (no effect) Bit 12",
        "5677": "Undefined Access (no effect) Bit 13",
        "5678": "Undefined Access (no effect) Bit 14",
        "5679": "Undefined Access (no effect) Bit 15",
        "5680": "Query account information",
        "5681": "Change privileges assigned to account",
        "5682": "Change quotas assigned to account",
        "5683": "Change logon capabilities assigned to account",
        "5684": "Change the Posix ID offset assigned to the accounted domain",
        "5685": "Undefined Access (no effect) Bit 5",
        "5686": "Undefined Access (no effect) Bit 6",
        "5687": "Undefined Access (no effect) Bit 7",
        "5688": "Undefined Access (no effect) Bit 8",
        "5689": "Undefined Access (no effect) Bit 9",
        "5690": "Undefined Access (no effect) Bit 10",
        "5691": "Undefined Access (no effect) Bit 11",
        "5692": "Undefined Access (no effect) Bit 12",
        "5693": "Undefined Access (no effect) Bit 13",
        "5694": "Undefined Access (no effect) Bit 14",
        "5695": "Undefined Access (no effect) Bit 15",
        "5696": "KeyedEvent Wait",
        "5697": "KeyedEvent Wake",
        "5698": "Undefined Access (no effect) Bit 2",
        "5699": "Undefined Access (no effect) Bit 3",
        "5700": "Undefined Access (no effect) Bit 4",
        "5701": "Undefined Access (no effect) Bit 5",
        "5702": "Undefined Access (no effect) Bit 6",
        "5703": "Undefined Access (no effect) Bit 7",
        "5704": "Undefined Access (no effect) Bit 8",
        "5705": "Undefined Access (no effect) Bit 9",
        "5706": "Undefined Access (no effect) Bit 10",
        "5707": "Undefined Access (no effect) Bit 11",
        "5708": "Undefined Access (no effect) Bit 12",
        "5709": "Undefined Access (no effect) Bit 13",
        "5710": "Undefined Access (no effect) Bit 14",
        "5711": "Undefined Access (no effect) Bit 15",
        "6656": "Enumerate desktops",
        "6657": "Read attributes",
        "6658": "Access Clipboard",
        "6659": "Create desktop",
        "6660": "Write attributes",
        "6661": "Access global atoms",
        "6662": "Exit windows",
        "6663": "Unused Access Flag",
        "6664": "Include this windowstation in enumerations",
        "6665": "Read screen",
        "6672": "Read Objects",
        "6673": "Create window",
        "6674": "Create menu",
        "6675": "Hook control",
        "6676": "Journal (record)",
        "6677": "Journal (playback)",
        "6678": "Include this desktop in enumerations",
        "6679": "Write objects",
        "6680": "Switch to this desktop",
        "6912": "Administer print server",
        "6913": "Enumerate printers",
        "6930": "Full Control",
        "6931": "Print",
        "6948": "Administer Document",
        "7168": "Connect to service controller",
        "7169": "Create a new service",
        "7170": "Enumerate services",
        "7171": "Lock service database for exclusive access",
        "7172": "Query service database lock state",
        "7173": "Set last-known-good state of service database",
        "7184": "Query service configuration information",
        "7185": "Set service configuration information",
        "7186": "Query status of service",
        "7187": "Enumerate dependencies of service",
        "7188": "Start the service",
        "7189": "Stop the service",
        "7190": "Pause or continue the service",
        "7191": "Query information from service",
        "7192": "Issue service-specific control commands",
        "7424": "DDE Share Read",
        "7425": "DDE Share Write",
        "7426": "DDE Share Initiate Static",
        "7427": "DDE Share Initiate Link",
        "7428": "DDE Share Request",
        "7429": "DDE Share Advise",
        "7430": "DDE Share Poke",
        "7431": "DDE Share Execute",
        "7432": "DDE Share Add Items",
        "7433": "DDE Share List Items",
        "7680": "Create Child",
        "7681": "Delete Child",
        "7682": "List Contents",
        "7683": "Write Self",
        "7684": "Read Property",
        "7685": "Write Property",
        "7686": "Delete Tree",
        "7687": "List Object",
        "7688": "Control Access",
        "7689": "Undefined Access (no effect) Bit 9",
        "7690": "Undefined Access (no effect) Bit 10",
        "7691": "Undefined Access (no effect) Bit 11",
        "7692": "Undefined Access (no effect) Bit 12",
        "7693": "Undefined Access (no effect) Bit 13",
        "7694": "Undefined Access (no effect) Bit 14",
        "7695": "Undefined Access (no effect) Bit 15",
        "7936": "Audit Set System Policy",
        "7937": "Audit Query System Policy",
        "7938": "Audit Set Per User Policy",
        "7939": "Audit Query Per User Policy",
        "7940": "Audit Enumerate Users",
        "7941": "Audit Set Options",
        "7942": "Audit Query Options",
        "8064": "Port sharing (read)",
        "8065": "Port sharing (write)",
        "8096": "Default credentials",
        "8097": "Credentials manager",
        "8098": "Fresh credentials",
        "8192": "Kerberos",
        "8193": "Preshared key",
        "8194": "Unknown authentication",
        "8195": "DES",
        "8196": "3DES",
        "8197": "MD5",
        "8198": "SHA1",
        "8199": "Local computer",
        "8200": "Remote computer",
        "8201": "No state",
        "8202": "Sent first (SA) payload",
        "8203": "Sent second (KE) payload",
        "8204": "Sent third (ID) payload",
        "8205": "Initiator",
        "8206": "Responder",
        "8207": "No state",
        "8208": "Sent first (SA) payload",
        "8209": "Sent final payload",
        "8210": "Complete",
        "8211": "Unknown",
        "8212": "Transport",
        "8213": "Tunnel",
        "8214": "IKE/AuthIP DoS prevention mode started",
        "8215": "IKE/AuthIP DoS prevention mode stopped",
        "8216": "Enabled",
        "8217": "Not enabled",
        "8218": "No state",
        "8219": "Sent first (EM attributes) payload",
        "8220": "Sent second (SSPI) payload",
        "8221": "Sent third (hash) payload",
        "8222": "IKEv1",
        "8223": "AuthIP",
        "8224": "Anonymous",
        "8225": "NTLM V2",
        "8226": "CGA",
        "8227": "Certificate",
        "8228": "SSL",
        "8229": "None",
        "8230": "DH group 1",
        "8231": "DH group 2",
        "8232": "DH group 14",
        "8233": "DH group ECP 256",
        "8234": "DH group ECP 384",
        "8235": "AES-128",
        "8236": "AES-192",
        "8237": "AES-256",
        "8238": "Certificate ECDSA P256",
        "8239": "Certificate ECDSA P384",
        "8240": "SSL ECDSA P256",
        "8241": "SSL ECDSA P384",
        "8242": "SHA 256",
        "8243": "SHA 384",
        "8244": "IKEv2",
        "8245": "EAP payload sent",
        "8246": "Authentication payload sent",
        "8247": "EAP",
        "8248": "DH group 24",
        "8272": "System",
        "8273": "Logon/Logoff",
        "8274": "Object Access",
        "8275": "Privilege Use",
        "8276": "Detailed Tracking",
        "8277": "Policy Change",
        "8278": "Account Management",
        "8279": "DS Access",
        "8280": "Account Logon",
        "8448": "Success removed",
        "8449": "Success Added",
        "8450": "Failure removed",
        "8451": "Failure added",
        "8452": "Success include removed",
        "8453": "Success include added",
        "8454": "Success exclude removed",
        "8455": "Success exclude added",
        "8456": "Failure include removed",
        "8457": "Failure include added",
        "8458": "Failure exclude removed",
        "8459": "Failure exclude added",
        "12288": "Security State Change",
        "12289": "Security System Extension",
        "12290": "System Integrity",
        "12291": "IPsec Driver",
        "12292": "Other System Events",
        "12544": "Logon",
        "12545": "Logoff",
        "12546": "Account Lockout",
        "12547": "IPsec Main Mode",
        "12548": "Special Logon",
        "12549": "IPsec Quick Mode",
        "12550": "IPsec Extended Mode",
        "12551": "Other Logon/Logoff Events",
        "12552": "Network Policy Server",
        "12553": "User / Device Claims",
        "12554": "Group Membership",
        "12800": "File System",
        "12801": "Registry",
        "12802": "Kernel Object",
        "12803": "SAM",
        "12804": "Other Object Access Events",
        "12805": "Certification Services",
        "12806": "Application Generated",
        "12807": "Handle Manipulation",
        "12808": "File Share",
        "12809": "Filtering Platform Packet Drop",
        "12810": "Filtering Platform Connection",
        "12811": "Detailed File Share",
        "12812": "Removable Storage",
        "12813": "Central Policy Staging",
        "13056": "Sensitive Privilege Use",
        "13057": "Non Sensitive Privilege Use",
        "13058": "Other Privilege Use Events",
        "13312": "Process Creation",
        "13313": "Process Termination",
        "13314": "DPAPI Activity",
        "13315": "RPC Events",
        "13316": "Plug and Play Events",
        "13317": "Token Right Adjusted Events",
        "13568": "Audit Policy Change",
        "13569": "Authentication Policy Change",
        "13570": "Authorization Policy Change",
        "13571": "MPSSVC Rule-Level Policy Change",
        "13572": "Filtering Platform Policy Change",
        "13573": "Other Policy Change Events",
        "13824": "User Account Management",
        "13825": "Computer Account Management",
        "13826": "Security Group Management",
        "13827": "Distribution Group Management",
        "13828": "Application Group Management",
        "13829": "Other Account Management Events",
        "14080": "Directory Service Access",
        "14081": "Directory Service Changes",
        "14082": "Directory Service Replication",
        "14083": "Detailed Directory Service Replication",
        "14336": "Credential Validation",
        "14337": "Kerberos Service Ticket Operations",
        "14338": "Other Account Logon Events",
        "14339": "Kerberos Authentication Service",
        "14592": "Inbound",
        "14593": "Outbound",
        "14594": "Forward",
        "14595": "Bidirectional",
        "14596": "IP Packet",
        "14597": "Transport",
        "14598": "Forward",
        "14599": "Stream",
        "14600": "Datagram Data",
        "14601": "ICMP Error",
        "14602": "MAC 802.3",
        "14603": "MAC Native",
        "14604": "vSwitch",
        "14608": "Resource Assignment",
        "14609": "Listen",
        "14610": "Receive/Accept",
        "14611": "Connect",
        "14612": "Flow Established",
        "14614": "Resource Release",
        "14615": "Endpoint Closure",
        "14616": "Connect Redirect",
        "14617": "Bind Redirect",
        "14624": "Stream Packet",
        "14640": "ICMP Echo-Request",
        "14641": "vSwitch Ingress",
        "14642": "vSwitch Egress",
        "14672": "<Binary>",
        "14673": "[NULL]",
        "14674": "Value Added",
        "14675": "Value Deleted",
        "14676": "Active Directory Domain Services",
        "14677": "Active Directory Lightweight Directory Services",
        "14678": "Yes",
        "14679": "No",
        "14680": "Value Added With Expiration Time",
        "14681": "Value Deleted With Expiration Time",
        "14688": "Value Auto Deleted With Expiration Time",
        "16384": "Add",
        "16385": "Delete",
        "16386": "Boot-time",
        "16387": "Persistent",
        "16388": "Not persistent",
        "16389": "Block",
        "16390": "Permit",
        "16391": "Callout",
        "16392": "MD5",
        "16393": "SHA-1",
        "16394": "SHA-256",
        "16395": "AES-GCM 128",
        "16396": "AES-GCM 192",
        "16397": "AES-GCM 256",
        "16398": "DES",
        "16399": "3DES",
        "16400": "AES-128",
        "16401": "AES-192",
        "16402": "AES-256",
        "16403": "Transport",
        "16404": "Tunnel",
        "16405": "Responder",
        "16406": "Initiator",
        "16407": "AES-GMAC 128",
        "16408": "AES-GMAC 192",
        "16409": "AES-GMAC 256",
        "16416": "AuthNoEncap Transport",
        "16896": "Enable WMI Account",
        "16897": "Execute Method",
        "16898": "Full Write",
        "16899": "Partial Write",
        "16900": "Provider Write",
        "16901": "Remote Access",
        "16902": "Subscribe",
        "16903": "Publish",
    };

    // lookupMessageCode returns the string associated with the code. key should
    // be the name of the field in evt containing the code (e.g. %%2313).
    var lookupMessageCode = function (evt, key) {
        var code = evt.Get(key);
        if (!code) {
            return;
        }
        code = code.replace("%%", "");
        return msobjsMessageTable[code];
    };

    var addEventFields = function(evt){
        var code = evt.Get("event.code");
        if (!code) {
            return;
        }
        var eventActionDescription = eventActionTypes[code][2];
        if (eventActionDescription) {
            evt.AppendTo("event.category", eventActionTypes[code][0]);
            evt.AppendTo("event.type", eventActionTypes[code][1]);
            evt.Put("event.action", eventActionTypes[code][2]);
        }
    };

    var addEventOutcome = function(evt) {
        var auditResult = evt.Get("winlog.keywords");
        if (!auditResult) {
            return;
        }
        var  eventOutcome = eventOutcomes[auditResult];
        if (eventOutcome === undefined) {
            return;
        }
        evt.Put("event.outcome", eventOutcome);
    };

    var addLogonType = function(evt) {
        var code = evt.Get("winlog.event_data.LogonType");
        if (!code) {
            return;
        }
        var descriptiveLogonType = logonTypes[code];
        if (descriptiveLogonType === undefined) {
            return;
        }
        evt.Put("winlog.logon.type", descriptiveLogonType);
    };

    var addFailureCode = function(evt) {
        var msg = lookupMessageCode(evt, "winlog.event_data.FailureReason");
        if (!msg) {
            return;
        }
        evt.Put("winlog.logon.failure.reason", msg);
    };

    var addFailureStatus = function(evt) {
        var code = evt.Get("winlog.event_data.Status");
        if (!code) {
            return;
        }
        var descriptiveFailureStatus = logonFailureStatus[code];
        if (descriptiveFailureStatus === undefined) {
            return;
        }
        evt.Put("winlog.logon.failure.status", descriptiveFailureStatus);
    };

    var addFailureSubStatus = function(evt) {
        var code = evt.Get("winlog.event_data.SubStatus");
        if (!code) {
            return;
        }
        var descriptiveFailureStatus = logonFailureStatus[code];
        if (descriptiveFailureStatus === undefined) {
            return;
        }
        evt.Put("winlog.logon.failure.sub_status", descriptiveFailureStatus);
    };

    var addUACDescription = function(evt) {
        var code = evt.Get("winlog.event_data.NewUacValue");
        if (!code) {
            return;
        }
        var uacCode = parseInt(code);
        var uacResult = [];
        for (var i = 0; i < uacFlags.length; i++) {
            if ((uacCode | uacFlags[i][0]) === uacCode) {
                uacResult.push(uacFlags[i][1]);
            }
        }
        if (uacResult) {
            evt.Put("winlog.event_data.NewUACList", uacResult);
        }
        var uacList = evt.Get("winlog.event_data.UserAccountControl").replace(/\s/g, '').split("%%").filter(String);
        if (!uacList) {
            return;
        }
        evt.Put("winlog.event_data.UserAccountControl", uacList);
      };

    var addAuditInfo = function(evt) {
        var subcategoryGuid = evt.Get("winlog.event_data.SubcategoryGuid").replace("{", '').replace("}", '').toUpperCase();
        if (!subcategoryGuid) {
            return;
        }
        if (!auditDescription[subcategoryGuid]) {
            return;
        }
        evt.Put("winlog.event_data.Category", auditDescription[subcategoryGuid][1]);
        evt.Put("winlog.event_data.SubCategory", auditDescription[subcategoryGuid][0]);
        var codedActions = evt.Get("winlog.event_data.AuditPolicyChanges").split(",");
        var actionResults = [];
        for (var j = 0; j < codedActions.length; j++) {
            var actionCode = codedActions[j].replace("%%", '').replace(' ', '');
            actionResults.push(auditActions[actionCode]);
        }
        evt.Put("winlog.event_data.AuditPolicyChangesDescription", actionResults);
    };

    var addTicketOptionsDescription = function(evt) {
        var code = evt.Get("winlog.event_data.TicketOptions");
        if (!code) {
            return;
        }
        var tktCode = parseInt(code, 16).toString(2);
        var tktResult = [];
        var tktCodeLen = tktCode.length;
        for (var i = tktCodeLen; i >= 0; i--) {
            if (tktCode[i] == 1) {
                tktResult.push(ticketOptions[(32-tktCodeLen)+i]);
            }
        }
        if (tktResult) {
            evt.Put("winlog.event_data.TicketOptionsDescription", tktResult);
        }
    };

    var addTicketEncryptionType = function(evt) {
        var code = evt.Get("winlog.event_data.TicketEncryptionType");
        if (!code) {
            return;
        }
        var encTypeCode = code.toLowerCase();
        evt.Put("winlog.event_data.TicketEncryptionTypeDescription", ticketEncryptionTypes[encTypeCode]);
    };

    var addTicketStatus = function(evt) {
        var code = evt.Get("winlog.event_data.Status");
        if (!code) {
            return;
        }
        evt.Put("winlog.event_data.StatusDescription", kerberosTktStatusCodes[code]);
    };

    var addSessionData = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.AccountName", to: "user.name"},
                {from: "winlog.event_data.AccountDomain", to: "user.domain"},
                {from: "winlog.event_data.ClientAddress", to: "source.ip"},
                {from: "winlog.event_data.ClientName", to: "source.domain"},
                {from: "winlog.event_data.LogonID", to: "winlog.logon.id"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.AccountName");
            evt.AppendTo('related.user', user);
        })
        .Build();

    var addServiceFields = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.ServiceName", to: "service.name"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            var code = evt.Get("winlog.event_data.ServiceType");
            if (!code) {
                return;
            }
            evt.Put("service.type", serviceTypes[code]);
        })
        .Build();

    var copyTargetUser = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.TargetUserSid", to: "user.id"},
                {from: "winlog.event_data.TargetUserName", to: "user.name"},
                {from: "winlog.event_data.TargetDomainName", to: "user.domain"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.TargetUserName");
            if (/.@*/.test(user)) {
                user = user.split('@')[0];
                evt.Put('user.name', user);
            }
            evt.AppendTo('related.user', user);
        })
        .Build();

    var copyTargetUserToGroup = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.TargetUserSid", to: "group.id"},
                {from: "winlog.event_data.TargetUserName", to: "group.name"},
                {from: "winlog.event_data.TargetDomainName", to: "group.domain"},
            ],
            ignore_missing: true,
        })
        .Build();

    var copyTargetUserToComputerObject = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.TargetSid", to: "winlog.computerObject.id"},
                {from: "winlog.event_data.TargetUserName", to: "winlog.computerObject.name"},
                {from: "winlog.event_data.TargetDomainName", to: "winlog.computerObject.domain"},
            ],
            ignore_missing: true,
        })
        .Build();

    var copyTargetUserLogonId  = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.TargetLogonId", to: "winlog.logon.id"},
            ],
            ignore_missing: true,
        })
        .Build();

    var copySubjectUser  = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.SubjectUserSid", to: "user.id"},
                {from: "winlog.event_data.SubjectUserName", to: "user.name"},
                {from: "winlog.event_data.SubjectDomainName", to: "user.domain"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.SubjectUserName");
            evt.AppendTo('related.user', user);
        })
        .Build();

    var copySubjectUserFromUserData  = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.user_data.SubjectUserSid", to: "user.id"},
                {from: "winlog.user_data.SubjectUserName", to: "user.name"},
                {from: "winlog.user_data.SubjectDomainName", to: "user.domain"},
            ],
            ignore_missing: true,
        })
        .Add(function(evt) {
            var user = evt.Get("winlog.user_data.SubjectUserName");
            evt.AppendTo('related.user', user);
        })
        .Build();

    var copySubjectUserLogonId  = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.SubjectLogonId", to: "winlog.logon.id"},
            ],
            ignore_missing: true,
        })
        .Build();

    var copySubjectUserLogonIdFromUserData  = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.user_data.SubjectLogonId", to: "winlog.logon.id"},
            ],
            ignore_missing: true,
        })
        .Build();

    var renameCommonAuthFields = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.ProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.ProcessName", to: "process.executable"},
                {from: "winlog.event_data.IpAddress", to: "source.ip", type: "ip"},
                {from: "winlog.event_data.IpPort", to: "source.port", type: "long"},
                {from: "winlog.event_data.WorkstationName", to: "source.domain"},
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(function(evt) {
            var name = evt.Get("process.name");
            if (name) {
                return;
            }
            var exe = evt.Get("process.executable");
            if (!exe) {
                return;
            }
            evt.Put("process.name", path.basename(exe));
        })
        .Build();

    var renameNewProcessFields = new processor.Chain()
        .Convert({
            fields: [
                {from: "winlog.event_data.NewProcessId", to: "process.pid", type: "long"},
                {from: "winlog.event_data.NewProcessName", to: "process.executable"},
                {from: "winlog.event_data.ParentProcessName", to: "process.parent.executable"}
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(function(evt) {
            var name = evt.Get("process.name");
            if (name) {
                return;
            }
            var exe = evt.Get("process.executable");
            if (!exe) {
                return;
            }
            evt.Put("process.name", path.basename(exe));
        })
        .Add(function(evt) {
            var name = evt.Get("process.parent.name");
            if (name) {
                return;
            }
            var exe = evt.Get("process.parent.executable");
            if (!exe) {
                return;
            }
            evt.Put("process.parent.name", path.basename(exe));
        })
        .Add(function(evt) {
            var cl = evt.Get("winlog.event_data.CommandLine");
            if (!cl) {
                return;
            }
            evt.Put("process.args", winlogbeat.splitCommandLine(cl));
            evt.Put("process.command_line", cl);
        })
        .Build();

    // Handles 4634 and 4647.
    var logoff = new processor.Chain()
        .Add(copyTargetUser)
        .Add(copyTargetUserLogonId)
        .Add(addLogonType)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    // Handles both 4624
    var logonSuccess = new processor.Chain()
        .Add(copyTargetUser)
        .Add(copyTargetUserLogonId)
        .Add(addLogonType)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.SubjectUserName");
            if (user) {
                var res = /^-$/.test(user);
                if (!res) {
                    evt.AppendTo('related.user', user);
                }
            }
         })
        .Build();

    // Handles both 4648
    var event4648 = new processor.Chain()
        .Add(copyTargetUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.SubjectUserName");
            if (user) {
                var res = /^-$/.test(user);
                if (!res) {
                    evt.AppendTo('related.user', user);
                }
            }
         })
        .Build();

    var event4625 = new processor.Chain()
        .Add(copyTargetUser)
        .Add(copySubjectUserLogonId)
        .Add(addLogonType)
        .Add(addFailureCode)
        .Add(addFailureStatus)
        .Add(addFailureSubStatus)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    var event4672 = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(function(evt) {
            var privs = evt.Get("winlog.event_data.PrivilegeList");
            if (!privs) {
                return;
            }
            evt.Put("winlog.event_data.PrivilegeList", privs.split(/\s+/));
        })
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    var event4688 = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameNewProcessFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.TargetUserName");
            var res = /^-$/.test(user);
                if (!res) {
                    evt.AppendTo('related.user', user);
                }
        })
        .Build();

    var event4689 = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    var event4697 = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addServiceFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "change");
        })
        .Build();

    var userMgmtEvts = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addUACDescription)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var user = evt.Get("winlog.event_data.TargetUserName");
            evt.AppendTo('related.user', user);
            evt.AppendTo("event.type", "user");
        })
        .Build();

    var userRenamed = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var userNew = evt.Get("winlog.event_data.NewTargetUserName");
            evt.AppendTo('related.user', userNew);
            var userOld = evt.Get("winlog.event_data.OldTargetUserName");
            evt.AppendTo('related.user', userOld);
            evt.AppendTo("event.type", "user");
        })
        .Build();

    var groupMgmtEvts = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(copyTargetUserToGroup)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "group");
            var member = evt.Get("winlog.event_data.MemberName");
            if (!member) {
                return;
            }
            evt.AppendTo("related.user", member.split(',')[0].replace('CN=', '').replace('cn=', ''));
        })

        .Build();

    var auditLogCleared = new processor.Chain()
        .Add(copySubjectUserFromUserData)
        .Add(copySubjectUserLogonIdFromUserData)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "change");
        })
        .Build();

    var auditChanged = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addAuditInfo)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "change");
        })
        .Build();

    var auditLogMgmt = new processor.Chain()
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    var computerMgmtEvts = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(copyTargetUserToComputerObject)
        .Add(renameCommonAuthFields)
        .Add(addUACDescription)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var privs = evt.Get("winlog.event_data.PrivilegeList");
            if (!privs) {
                return;
            }
            evt.Put("winlog.event_data.PrivilegeList", privs.split(/\s+/));
            evt.AppendTo("event.type", "admin");
        })
        .Build();

    var sessionEvts = new processor.Chain()
        .Add(addSessionData)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

     var event4964 = new processor.Chain()
        .Add(copyTargetUser)
        .Add(copyTargetUserLogonId)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "group");
        })
        .Build();

    var kerberosTktEvts = new processor.Chain()
        .Add(copyTargetUser)
        .Add(renameCommonAuthFields)
        .Add(addTicketOptionsDescription)
        .Add(addTicketEncryptionType)
        .Add(addTicketStatus)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var ip = evt.Get("source.ip");
            if (/::ffff:/.test(ip)) {
                evt.Put("source.ip", ip.replace("::ffff:", ""));
            }
        })
        .Build();

     var event4776 = new processor.Chain()
        .Add(copyTargetUser)
        .Add(addFailureStatus)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Build();

    var  scheduledTask = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            evt.AppendTo("event.type", "admin");
        })
        .Build();

    var sensitivePrivilege = new processor.Chain()
        .Add(copySubjectUser)
        .Add(copySubjectUserLogonId)
        .Add(renameCommonAuthFields)
        .Add(addEventFields)
        .Add(addEventOutcome)
        .Add(function(evt) {
            var privs = evt.Get("winlog.event_data.PrivilegeList");
            if (!privs) {
                return;
            }
            evt.Put("winlog.event_data.PrivilegeList", privs.split(/\s+/));
        })
        .Add(function(evt){
            var maskCodes = evt.Get("winlog.event_data.AccessMask");
            if (!maskCodes) {
                return;
            }
            var maskList = maskCodes.replace(/\s+/g, '').split("%%").filter(String);
            evt.Put("winlog.event_data.AccessMask", maskList);
            var maskResults = [];
            for (var j = 0; j < maskList.length; j++) {
                var description = msobjsMessageTable[maskList[j]];
                if (description === undefined) {
                    return;
                }
                maskResults.push(description);
            }
            evt.Put("winlog.event_data.AccessMaskDescription", maskResults);
        })
        .Build();

    return {

        // 1100 - The event logging service has shut down.
        1100: auditLogMgmt.Run,

        // 1102 - The audit log was cleared.
        1102: auditLogCleared.Run,

        // 1104 - The security log is now full.
        1104: auditLogMgmt.Run,

        // 1105 - Event log automatic backup.
        1105: auditLogMgmt.Run,

        // 1108 - The event logging service encountered an error while processing an incoming event published from %1
        1108: auditLogMgmt.Run,

        // 4624 - An account was successfully logged on.
        4624: logonSuccess.Run,

        // 4625 - An account failed to log on.
        4625: event4625.Run,

        // 4634 - An account was logged off.
        4634: logoff.Run,

        // 4647 - User initiated logoff.
        4647: logoff.Run,

        // 4648 - A logon was attempted using explicit credentials.
        4648: event4648.Run,

        // 4672 - Special privileges assigned to new logon.
        4672: event4672.Run,

        // 4673 - A privileged service was called.
        4673: sensitivePrivilege.Run,

        // 4674 - An operation was attempted on a privileged object.
        4674: sensitivePrivilege.Run,

        // 4688 - A new process has been created.
        4688: event4688.Run,

        // 4689 - A process has exited.
        4689: event4689.Run,

        // 4697 - A service was installed in the system.
        4697: event4697.Run,

        // 4698 - A scheduled task was created.
        4698: scheduledTask.Run,

        // 4699 - A scheduled task was deleted.
        4699: scheduledTask.Run,

        // 4700 - A scheduled task was enabled.
        4700: scheduledTask.Run,

        // 4701 - A scheduled task was disabled.
        4701: scheduledTask.Run,

        // 4702 - A scheduled task was updated.
        4702: scheduledTask.Run,

        // 4719 -  System audit policy was changed.
        4719: auditChanged.Run,

        // 4720 - A user account was created
        4720: userMgmtEvts.Run,

        // 4722 - A user account was enabled
        4722: userMgmtEvts.Run,

        // 4723 - An attempt was made to change an account's password
        4723: userMgmtEvts.Run,

        // 4724 - An attempt was made to reset an account's password
        4724: userMgmtEvts.Run,

        // 4725 - A user account was disabled.
        4725: userMgmtEvts.Run,

        // 4726 - An user account was deleted.
        4726: userMgmtEvts.Run,

        // 4727 - A security-enabled global group was created.
        4727: groupMgmtEvts.Run,

        // 4728 - A member was added to a security-enabled global group.
        4728: groupMgmtEvts.Run,

        // 4729 - A member was removed from a security-enabled global group.
        4729: groupMgmtEvts.Run,

        // 4730 - A security-enabled global group was deleted.
        4730: groupMgmtEvts.Run,

        // 4731 - A security-enabled local group was created.
        4731: groupMgmtEvts.Run,

        // 4732 - A member was added to a security-enabled local group.
        4732: groupMgmtEvts.Run,

        // 4733 - A member was removed from a security-enabled local group.
        4733: groupMgmtEvts.Run,

        // 4734 - A security-enabled local group was deleted.
        4734: groupMgmtEvts.Run,

        // 4735 - A security-enabled local group was changed.
        4735: groupMgmtEvts.Run,

        // 4737 - A security-enabled global group was changed.
        4737: groupMgmtEvts.Run,

        // 4738 - An user account was changed.
        4738: userMgmtEvts.Run,

        // 4740 - An account was locked out
        4740: userMgmtEvts.Run,

        // 4741 - A computer account was created.
        4741: computerMgmtEvts.Run,

        // 4742 -  A computer account was changed.
        4742: computerMgmtEvts.Run,

        // 4743 -  A computer account was deleted.
        4743: computerMgmtEvts.Run,

        // 4744 -  A security-disabled local group was created.
        4744: groupMgmtEvts.Run,

        // 4745 -  A security-disabled local group was changed.
        4745: groupMgmtEvts.Run,

        // 4746 -  A member was added to a security-disabled local group.
        4746: groupMgmtEvts.Run,

        // 4747 -  A member was removed from a security-disabled local group.
        4747: groupMgmtEvts.Run,

        // 4748 -  A security-disabled local group was deleted.
        4748: groupMgmtEvts.Run,

        // 4749 - A security-disabled global group was created.
        4749: groupMgmtEvts.Run,

        // 4750 - A security-disabled global group was changed.
        4750: groupMgmtEvts.Run,

        // 4751 - A member was added to a security-disabled global group.
        4751: groupMgmtEvts.Run,

        // 4752 - A member was removed from a security-disabled global group.
        4752: groupMgmtEvts.Run,

        // 4753 - A security-disabled global group was deleted.
        4753: groupMgmtEvts.Run,

        // 4754 -  A security-enabled universal group was created.
        4754: groupMgmtEvts.Run,

        // 4755 - A security-enabled universal group was changed.
        4755: groupMgmtEvts.Run,

        // 4756 - A member was added to a security-enabled universal group.
        4756: groupMgmtEvts.Run,

        // 4757 - A member was removed from a security-enabled universal group.
        4757: groupMgmtEvts.Run,

        // 4758 - A security-enabled universal group was deleted.
        4758: groupMgmtEvts.Run,

        // 4759 - A security-disabled universal group was created.
        4759: groupMgmtEvts.Run,

        // 4760 - A security-disabled universal group was changed.
        4760: groupMgmtEvts.Run,

        // 4761 - A member was added to a security-disabled universal group.
        4761: groupMgmtEvts.Run,

        // 4762 - A member was removed from a security-disabled universal group.
        4762: groupMgmtEvts.Run,

        // 4763 - A security-disabled global group was deleted.
        4763: groupMgmtEvts.Run,

        // 4764 - A group\'s type was changed.
        4764: groupMgmtEvts.Run,

        // 4767 - A user account was unlocked.
        4767: userMgmtEvts.Run,

        // 4768 - A Kerberos authentication ticket TGT was requested.
        4768: kerberosTktEvts.Run,

        // 4769 - A Kerberos service ticket was requested.
        4769: kerberosTktEvts.Run,

        // 4770 - A Kerberos service ticket was renewed.
        4770: kerberosTktEvts.Run,

        // 4771 - Kerberos pre-authentication failed.
        4771: kerberosTktEvts.Run,

        // 4776 - The computer attempted to validate the credentials for an account.
        4776: event4776.Run,

        // 4778 - A session was reconnected to a Window Station.
        4778: sessionEvts.Run,

        // 4779 - A session was disconnected from a Window Station.
        4779: sessionEvts.Run,

        // 4781 - The name of an account was changed.
        4781: userRenamed.Run,

        // 4798 - A user's local group membership was enumerated.
        4798: userMgmtEvts.Run,

        // 4799 - A security-enabled local group membership was enumerated.
        4799: groupMgmtEvts.Run,

        // 4964 - Special groups have been assigned to a new logon.
        4964: event4964.Run,

        process: function(evt) {
            var eventId = evt.Get("winlog.event_id");
            var processor = this[eventId];
            if (processor === undefined) {
                return;
            }
            evt.Put("event.module", "security");
            processor(evt);
        },
    };
})();

function process(evt) {
    return security.process(evt);
}
