// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var processor = require("processor");
var console   = require("console");

// PipelineBuilder to aid debugging of pipelines during development.
function PipelineBuilder(pipelineName, debug) {
    this.pipeline = new processor.Chain();
    this.add = function (processor) {
        this.pipeline = this.pipeline.Add(processor);
    };
    this.Add = function (name, processor) {
        this.add(processor);
        if (debug) {
            this.add(makeLogEvent("after " + pipelineName + "/" + name));
        }
    };
    this.Build = function () {
        if (debug) {
            this.add(makeLogEvent(pipelineName + "processing done"));
        }
        return this.pipeline.Build();
    };
    if (debug) {
        this.add(makeLogEvent(pipelineName + ": begin processing event"));
    }
}

function appendFields(options) {
    return function(evt) {
        options.fields.forEach(function (key) {
            var value = evt.Get(key);
            if (value != null) evt.AppendTo(options.to, value);
        });
    }
}

// logEvent(msg)
//
// Processor that logs the current value of evt to console.debug.
function makeLogEvent(msg) {
    return function (evt) {
        console.debug(msg + " :" +  JSON.stringify(evt, null, 4));
    };
}

// makeConditional({condition:expr, result1:processor|expr, [...]})
//
// Processor that selects which processor to run depending on the result of
// evaluating a _condition_. Result can be boolean (if-else equivalent) or any
// other value (switch equivalent). Unspecified values are a no-op.
function makeConditional(options) {
    return function (evt) {
        var branch = options[options.condition(evt)] || function(evt){};
        return (typeof branch === "function" ? branch : branch.Run)(evt);
    };
}

// makeMapper({from:field, to:field, default:value mappings:{orig: new, [...]}})
//
// Processor that sets the `to` field by mapping of `from` field's value.
function makeMapper(options) {
    return function (evt) {
        var key = evt.Get(options.from);
        if (key == null && options.skip_missing) return;
        if (options.lowercase && typeof key == "string") {
            key = key.toLowerCase();
        }
        var value = options.default;
        if (key in options.mappings) {
            value = options.mappings[key];
        } else if (typeof value === "function") {
            value = value(key);
        }
        if (value != null) {
            evt.Put(options.to, value);
        }
    };
}

// Makes sure a name can be used as a field in the output document.
function validFieldName(s) {
    return s.replace(/[\ \.]/g, '_')
}

/* Turns a `common.NameValuePair` array into an object. Multiple-value fields
   are stored as arrays.
 input (a NameValuePair array):
     from_field: [
        {Name: name1, Value: value1},
        {Name: name2, Value: value2},
        {Name: name2, Value: value2b},
        [...]
        {Name: nameN, Value: valueN}
     ]

 output (an object):
     to_field: {
        name1: value1,
        name2: [value2, value2b],
        [...]
        nameN: valueN
     }
*/
function makeObjFromNameValuePairArray(options) {
    return function(evt) {
        var src = evt.Get(options.from);
        var dict = {};
        if (src == null) return;
        if (!(src instanceof Array)) {
            evt.Put(options.to, {"_raw": src} );
            return;
        }
        for (var i=0; i < src.length; i++) {
            var name, value;
            if (src[i] == null
                || (name=src[i].Name) == null
                || (value=src[i].Value) == null) continue;
            name = validFieldName(name);
            if (name in dict) {
                if (dict[name] instanceof Array) {
                    dict[name].push(value);
                } else {
                    dict[name] = [value];
                }
            } else {
                dict[name] = value;
            }
        }
        evt.Put(options.to, dict);
    }
}

/* Converts a Common.ModifiedProperty array into an object.
   input:
    from_field: [
        {Name: name1, OldValue: old1, NewValue: new1},
        {Name: name2, OldValue: old2, NewValue: new2},
        {Name: name2, OldValue: old2b, NewValue: new2b},
        [...]
        {Name: nameN, OldValue: oldN, NewValue: newN},
    ],

    output:
    to_field: {
        name1: { OldValue: old1, NewValue: new1 },
        name2: { OldValue: [old2, old2b], NewValue: [new2, new2b] },
        [...]
        nameN: { OldValue: oldN, NewValue: newN }
    }
 */
function makeDictFromModifiedPropertyArray(options) {
    return function(evt) {
        var src = evt.Get(options.from);
        var dict = {};
        if (src == null || !(src instanceof Array)) return;
        for (var i=0; i < src.length; i++) {
            var name, newValue, oldValue;
            if (src[i] == null
                || (name=src[i].Name) == null
                || (newValue=src[i].NewValue) == null
                || (oldValue=src[i].OldValue) == null) continue;
            name = validFieldName(name);
            if (name in dict) {
                if (dict[name].NewValue instanceof Array) {
                    dict[name].NewValue.push(newValue);
                    dict[name].OldValue.push(oldValue);
                } else {
                    dict[name].NewValue = [newValue];
                    dict[name].OldValue = [oldValue];
                }
            } else {
                dict[name] = {
                    NewValue: newValue,
                    OldValue: oldValue,
                };
            }
        }
        evt.Put(options.to, dict);
    }
}

function exchangeAdminSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.ExchangeAdmin", debug);
    builder.Add("saveFields", new processor.Convert({
        fields: [
            {from: 'o365audit.OrganizationName', to: 'organization.name'},
            {from: 'o365audit.OriginatingServer', to: 'server.address'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));
    return builder.Build();
}

function typeMapEnrich(conversions) {
    return function (evt) {
        var action = evt.Get("event.action");
        if (action != null && conversions.hasOwnProperty(action)) {
            var conv = conversions[action];
            if (conv.action !== undefined) evt.Put("event.action", conv.action);
            if (conv.category !== undefined) evt.Put("event.category", conv.category);
            if (conv.type !== undefined) evt.Put("event.type", conv.type);
            var n = conv.copy !== undefined? conv.copy.length : 0;
            for (var i=0; i<n; i++) {
                var value = evt.Get(conv.copy[i].from);
                if (value != null)
                    evt.Put(conv.copy[i].to, value);
            }
        }
    }
}

function azureADSchema(debug) {
    var azureADConversion = {
        'Add user.': {
            action: "added-user-account",
            category: 'iam',
            type: ['user', 'creation'],
            copy: [
                {
                    from: 'o365audit.ObjectId',
                    to: 'user.target.id',
                }
            ],
        },
        'Update user.': {
            action: "modified-user-account",
            category: 'iam',
            type: ['user', 'change'],
            copy: [
                {
                    from: 'o365audit.ObjectId',
                    to: 'user.target.id',
                }
            ],
        },
        'Delete user.': {
            action: "deleted-user-account",
            category: 'iam',
            type: ['user', 'deletion'],
            copy: [
                {
                    from: 'o365audit.ObjectId',
                    to: 'user.target.id',
                }
            ],
        },
    };

    var builder = new PipelineBuilder("o365.audit.AzureActiveDirectory", debug);
    builder.Add("setIAMFields", typeMapEnrich(azureADConversion));
    return builder.Build();
}

function teamsSchema(debug) {
    var teamsConversion = {
        'TeamCreated': {
            action: "added-group-account-to",
            category: 'iam',
            type: ['group', 'creation'],
            copy: [
                {
                    from: 'o365audit.TeamName',
                    to: 'group.name',
                }
            ],
        },
        'MemberAdded': {
            action: "added-users-to-group",
            category: 'iam',
            type: ['group', 'change'],
        },

        'Delete user.': {
            action: "deleted-user-account",
            category: 'iam',
            type: ['user', 'deletion'],
            copy: [
                {
                    from: 'o365audit.ObjectId',
                    to: 'user.target.id',
                }
            ],
        },
    };

    var builder = new PipelineBuilder("o365.audit.MicrosoftTeams", debug);
    builder.Add("setIAMFields", typeMapEnrich(teamsConversion));
    builder.Add("groupMembersToRelatedUser", function (evt) {
        var m = evt.Get("o365audit.Members");
        if (m == null || m.forEach == null) return;
        m.forEach(function (obj) {
            if (obj != null && obj.hasOwnProperty('UPN'))
                evt.AppendTo('related.user', obj.UPN);
        })
    })
    return builder.Build();
}

function azureADLogonSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.AzureActiveDirectoryLogon", debug);
    builder.Add("setEventAuthFields", function(evt){
       evt.Put("event.category", "authentication");
       var outcome = evt.Get("event.outcome");
       // As event.type is an array, this sets both the traditional
       // "authentication_success"/"authentication_failure"
       // and the ECS standard "start".
       var types = ["start"];
       if (outcome != null && outcome !== "unknown") {
           types.push("authentication_" + outcome);
       }
       evt.Put("event.type", types);
    });
    return builder.Build();
}

function sharePointFileOperationSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.SharePointFileOperation", debug);
    builder.Add("saveFields", new processor.Convert({
        fields: [
            {from: 'o365audit.ObjectId', to: 'url.original'},
            {from: 'o365audit.SourceRelativeUrl', to: 'file.directory'},
            {from: 'o365audit.SourceFileName', to: 'file.name'},
            {from: 'o365audit.SourceFileExtension', to: 'file.extension'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));

    var actionToCategoryType = {
        ComplianceSettingChanged: ['configuration', 'change'],
        FileAccessed: ['file', 'access'],
        FileDeleted: ['file', 'deletion'],
        FileDownloaded: ['file', 'access'],
        FileModified: ['file', 'change'],
        FileMoved:  ['file', 'change'],
        FileRenamed: ['file', 'change'],
        FileRestored: ['file', 'change'],
        FileUploaded: ['file', 'creation'],
        FolderCopied: ['file', 'creation'],
        FolderCreated: ['file', 'creation'],
        FolderDeleted: ['file', 'deletion'],
        FolderModified: ['file', 'change'],
        FolderMoved: ['file', 'change'],
        FolderRenamed: ['file', 'change'],
        FolderRestored: ['file', 'change'],
    };

    builder.Add("setEventFields", function(evt) {
        var action = evt.Get("o365audit.Operation");
        if (action == null) return;
        var fields = actionToCategoryType[action];
        if (fields == null) return;
        evt.Put("event.category", fields[0]);
        evt.Put("event.type", fields[1]);
    });
    return builder.Build();
}

function exchangeMailboxSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.SharePointFileOperation", debug);
    builder.Add("saveFields", new processor.Convert({
        fields: [
            {from: 'o365audit.MailboxOwnerUPN', to: 'user.email'},
            {from: 'o365audit.LogonUserSid', to: 'user.id', type: 'string'},
            {from: 'o365audit.LogonUserDisplayName', to: 'user.full_name'},
            {from: 'o365audit.OrganizationName', to: 'organization.name'},
            {from: 'o365audit.OriginatingServer', to: 'server.address'},
            {from: 'o365audit.ClientIPAddress', to: 'client.address'},
            {from: 'o365audit.ClientProcessName', to: 'process.name'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));
    return builder.Build();
}

function dataLossPreventionSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.DLP", debug);
    builder.Add("setEventFields", new processor.AddFields({
        target: 'event',
        fields: {
            kind: 'alert',
            category: 'file',
            type: 'access',
        },
    }));

    builder.Add("saveFields", new processor.Convert({
        fields: [
            // SharePoint metadata
            {from: 'o365audit.SharePointMetaData.From', to: 'user.id'},
            {from: 'o365audit.SharePointMetaData.FileName', to: 'file.name'},
            {from: 'o365audit.SharePointMetaData.FilePathUrl', to: 'url.original'},
            {from: 'o365audit.SharePointMetaData.UniqueId', to: 'file.inode'},
            {from: 'o365audit.SharePointMetaData.UniqueID', to: 'file.inode'},
            {from: 'o365audit.SharePointMetaData.FileOwner', to: 'file.owner'},

            // Exchange metadata
            {from: 'o365audit.ExchangeMetaData.From', to: 'source.user.email'},
            {from: 'o365audit.ExchangeMetaData.Subject', to: 'message'},

            // Policy details
            {from: 'o365audit.PolicyId', to: 'rule.id'},
            {from: 'o365audit.PolicyName', to: 'rule.name'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));

    builder.Add("setMTime", new processor.Timestamp({
        field: "o365audit.SharePointMetaData.LastModifiedTime",
        target_field: "file.mtime",
        layouts: [
            "2006-01-02T15:04:05",
            "2006-01-02T15:04:05Z",
        ],
        ignore_missing: true,
        ignore_failure: true,
    }));

    builder.Add("appendDestinationEmails", function(evt) {
       var list = [];
       var fields = [
           'o365audit.ExchangeMetaData.To',
           'o365audit.ExchangeMetaData.CC',
           'o365audit.ExchangeMetaData.BCC',
       ];
       for (var i=0; i<fields.length; i++) {
           var value = evt.Get(fields[i]);
           if (value == null) continue;
           if (value instanceof Array) {
               list = list.concat(value);
           } else {
               list.push(value);
           }
       }
       if (list.length == 1) {
           evt.Put("destination.user.email", list[0]);
       } else if (list.length > 1) {
           evt.Put("destination.user.email", list);
       }
    });

    // ExceptionInfo is documented as string but has been observed to be an object.
    builder.Add("fixExceptionInfo", function(evt) {
        var key = "o365audit.ExceptionInfo";
        var eInfo = evt.Get(key);
        if (eInfo == null) return;
        if (typeof eInfo === "string") {
            if (eInfo === "") {
                evt.Delete(key);
            } else {
                evt.Put(key, {
                    Reason: eInfo,
                });
            }
        }
    });

    builder.Add("extractRules", function(evt) {
        var policies = evt.Get("o365audit.PolicyDetails");
        if (policies == null) return;
        // rule.id will be an array of all rules' IDs.
        var ruleIds = [];
        // rule.name will be an array of all rules' names.
        var ruleNames = [];
        // event.severity will be the higher severity seen.
        var maxSeverity = -1;
        // event.outcome will determine if access to sensitive data was allowed.
        // Either because the rules were configured to only alert or because
        // the alert was overridden by the user.
        var allowed = true;
        for (var i = 0; i < policies.length; i++) {
            var rules = policies[i].Rules;
            if (rules == null) continue;
            for (var j = 0; j < rules.length; j++) {
                var rule = rules[j];
                var id = rule.RuleId;
                var name = rule.RuleName;
                var sev = severityToCode(rule.Severity);
                if (id != null && name != null) {
                    ruleIds.push(id);
                    ruleNames.push(name);
                }
                if (sev > maxSeverity) maxSeverity = sev;
                if (allowed) {
                    if (rule.Actions != null && rule.Actions.indexOf("BlockAccess") > -1) {
                        allowed = false;
                    }
                }
            }
        }
        if (ruleIds.length === 1) {
            evt.Put("rule.id", ruleIds[0]);
            evt.Put("rule.name", ruleNames[0]);
        } else if (ruleIds.length > 0) {
            evt.Put("rule.id", ruleIds);
            evt.Put("rule.name", ruleNames);
        }
        if (maxSeverity > -1) {
            evt.Put("event.severity", maxSeverity);
        }
        evt.Put("event.outcome", (allowed || isBlockOverride(evt))? "success" : "failure");
    });
    return builder.Build();
}

// Numeric mapping for o365 mgmt API severities.
function severityToCode(str) {
    if (str == null) return -1;
    switch (str.toLowerCase()) {
        case 'informational': return 1; // undocumented severity.
        case 'low': return 2;
        case 'medium': return 3;
        case 'high': return 4;
        default: return -1;
    }
}

// Was a DLP alert overridden with an exception?
function isBlockOverride(evt) {
    switch (evt.Get("o365audit.Operation").toLowerCase()) {
        // Undo means the block was undone via change of policy or override.
        case "dlpruleundo": return true;
        // Info means it was detected as a false positive but no action taken.
        case "dlpinfo": return false;
    }
    // It's not clear to me the format of ExceptionInfo. It could be an object
    // or a string containing a JSON object. Assume that if present, an exception
    // is made.
    var exInfo = evt.Get('o365audit.ExceptionInfo');
    return exInfo != null && exInfo !== "";
}

function yammerSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.Yammer", debug);
    builder.Add("saveFields", new processor.Convert({
        fields: [
            {from: 'o365audit.ActorUserId', to: 'user.email'},
            {from: 'o365audit.ActorYammerUserId', to: 'user.id', type: 'string'},
            {from: 'o365audit.FileId', to:'file.inode'},
            {from: 'o365audit.FileName', to: 'file.name'},
            {from: 'o365audit.GroupName', to: 'group.name'},
            {from: 'o365audit.TargetUserId', to: 'destination.user.email'},
            {from: 'o365audit.TargetYammerUserId', to: 'destination.user.id'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));

    var yammerConversion = {
        // Network or verified admin changes the Yammer network's configuration.
        // This includes setting the interval for exporting data and enabling chat.
        NetworkConfigurationUpdated: {
            category: "configuration",
            type: "change",
        },
        // Verified admin updates the Yammer network's security configuration.
        // This includes setting password expiration policies and restrictions
        // on IP addresses.
        NetworkSecurityConfigurationUpdated: {
            category: ["iam", "configuration"],
            type: ["admin", "change"],
        },
        // Verified admin updates the setting for the network data retention
        // policy to either Hard Delete or Soft Delete. Only verified admins
        // can perform this operation.
        SoftDeleteSettingsUpdated: {
            category: "configuration",
            type: "change",
        },
        // Network or verified admin changes the information that appears on
        // member profiles for network users network.
        ProcessProfileFields: {
            category: "configuration",
            type: "change"
        },
        // Verified admin turns Private Content Mode on or off. This mode
        // lets an admin view the posts in private groups and view private
        // messages between individual users (or groups of users). Only verified
        // admins only can perform this operation.
        SupervisorAdminToggled: {
            category: "configuration",
            type: "change"
        },
        // User uploads a file.
        FileCreated: {
            category: "file",
            type: "creation"
        },
        // User creates a group.
        GroupCreation: {
            category: "iam",
            type: ["group", "creation"],
        },
        // A group is deleted from Yammer.
        GroupDeletion: {
            category: "iam",
            type: ["group", "deletion"]
        },
        // User downloads a file.
        FileDownloaded: {
            category: "file",
            type: "access"
        },
        // User shares a file with another user.
        FileShared: {
            category: "file",
            type: "access"
        },
        // Network or verified admin suspends (deactivates) a user from Yammer.
        NetworkUserSuspended: {
            category: "iam",
            type: "user"
        },
        // User account is suspended (deactivated).
        UserSuspension: {
            category: "iam",
            type: "user"
        },
        // User changes the description of a file.
        FileUpdateDescription: {
            category: "file",
            type: "access"
        },
        // User changes the name of a file.
        FileUpdateName: {
            category: "file",
            type: "creation",
        },
        // User views a file.
        FileVisited: {
            category: "file",
            type: "access",
        },
    };

    builder.Add("setEventFields", typeMapEnrich(yammerConversion));
    return builder.Build();
}

function securityComplianceAlertsSchema(debug) {
    var builder = new PipelineBuilder("o365.audit.SecurityComplianceAlerts", debug);
    builder.Add("saveFields", new processor.Convert({
        fields: [
            {from: 'o365audit.Comments', to: 'message'},
            {from: 'o365audit.Name', to: 'rule.name'},
            {from: 'o365audit.PolicyId', to: 'rule.id'},
            {from: 'o365audit.Category', to: 'rule.category'},
            {from: 'o365audit.EntityType', to: 'rule.ruleset'},
            // This contains the entity that triggered the alert.
            // Name of a malware or email address.
            // Need to find a better ECS field for it.
            {from: 'o365audit.AlertEntityId', to: 'rule.description'},
            {from: 'o365audit.AlertLinks', to: 'rule.reference'},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));
    builder.Add("setEventFields", new processor.AddFields({
        target: 'event',
        fields: {
            kind: 'alert',
            category: 'web',
            type: 'info',
        },
    }));
    // event.severity is numeric.
    builder.Add("mapSeverity", function(evt) {
       var sev = severityToCode(evt.Get("o365audit.Severity"));
       if (sev >= 0) {
           evt.Put("event.severity", sev);
       }
    });
    builder.Add("mapCategory", makeMapper({
        from: 'o365audit.Category',
        to: 'event.category',
        default: 'authentication',
        lowercase: true,
        mappings: {
            'accessgovernance': 'authentication',
            'datagovernance': 'file',
            'datalossprevention': 'file',
            'threatmanagement': 'malware',
        },
    }));
    builder.Add("saveEntity", makeConditional({
        condition: function(evt) {
            return evt.Get("o365audit.EntityType");
        },
        'User': new processor.Convert({
            fields: [
                {from: "o365audit.AlertEntityId", to: "user.id", type: 'string'},
            ],
            ignore_missing: true,
            fail_on_error: false
        }),
        'Recipients': new processor.Convert({
            fields: [
                {from: "o365audit.AlertEntityId", to: "user.email"},
            ],
            ignore_missing: true,
            fail_on_error: false
        }),
        'Sender': new processor.Convert({
            fields: [
                {from: "o365audit.AlertEntityId", to: "user.email"},
            ],
            ignore_missing: true,
            fail_on_error: false
        }),
        'MalwareFamily': new processor.Convert({
            fields: [
                {from: "o365audit.AlertEntityId", to: "threat.technique.id"},
            ],
            ignore_missing: true,
            fail_on_error: false
        }),
    }));
    return builder.Build();
}

function splitEmailUserID(prefix) {
    var idField = prefix + ".id",
        nameField = prefix + ".name",
        domainField = prefix + ".domain",
        emailField = prefix + ".email";
    return function(evt) {
        var email = evt.Get(idField);
        if (email == null) return;
        var pos = email.indexOf('@');
        if (pos === -1) return;
        evt.Put(emailField, email);
        evt.Put(nameField, email.substr(0, pos));
        evt.Put(domainField, email.substr(pos+1));
    }
}

function AuditProcessor(tenant_names, debug) {
    var builder = new PipelineBuilder("o365.audit", debug);

    var unsetIPValues = {"null": true, "<null>": true, "": true};
    builder.Add("cleanupNulls", function(event) {
        [
            "o365audit.ClientIP",
            "o365audit.ClientIPAddress",
            "o365audit.ActorIpAddress",
            "o365audit.OriginatingServer"
        ].forEach(function(field) {
            if (event.Get(field) in unsetIPValues) event.Delete(field);
        });
    });
    builder.Add("convertCommonAuditRecordFields", new processor.Convert({
        fields: [
            {from: "o365audit.Id", to: "event.id"},
            {from: "o365audit.ClientIP", to: "client.address"},
            {from: "o365audit.ClientIPAddress", to: "client.address"},
            {from: "o365audit.ActorIpAddress", to: "client.address"},
            {from: "o365audit.UserId", to: "user.id", type: "string"},
            {from: "o365audit.Workload", to: "event.provider"},
            {from: "o365audit.Operation", to: "event.action"},
            {from: "o365audit.OrganizationId", to: "organization.id"},
            // Extra common fields:
            {from: "o365audit.UserAgent", to: "user_agent.original"},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));
    builder.Add("mapEventType", makeMapper({
        from: 'o365audit.RecordType',
        to: 'event.code',
        // Keep original RecordType for unknown mappings.
        default: function(recordType) {
            return recordType;
        },
        mappings: {
            1: 'ExchangeAdmin', // Events from the Exchange admin audit log.
            2: 'ExchangeItem', // Events from an Exchange mailbox audit log for actions that are performed on a single item, such as creating or receiving an email message.
            3: 'ExchangeItemGroup', // Events from an Exchange mailbox audit log for actions that can be performed on multiple items, such as moving or deleted one or more email messages.
            4: 'SharePoint', // SharePoint events.
            6: 'SharePointFileOperation', // SharePoint file operation events.
            8: 'AzureActiveDirectory', // Azure Active Directory events.
            9: 'AzureActiveDirectoryAccountLogon', // Azure Active Directory OrgId logon events (deprecating).
            10: 'DataCenterSecurityCmdlet', // Data Center security cmdlet events.
            11: 'ComplianceDLPSharePoint', // Data loss protection (DLP) events in SharePoint and OneDrive for Business.
            12: 'Sway', // Events from the Sway service and clients.
            13: 'ComplianceDLPExchange', // Data loss protection (DLP) events in Exchange, when configured via Unified DLP Policy. DLP events based on Exchange Transport Rules are not supported.
            14: 'SharePointSharingOperation', // SharePoint sharing events.
            15: 'AzureActiveDirectoryStsLogon', // Secure Token Service (STS) logon events in Azure Active Directory.
            18: 'SecurityComplianceCenterEOPCmdlet', // Admin actions from the Security & Compliance Center.
            20: 'PowerBIAudit', // Power BI events.
            21: 'CRM', // Microsoft CRM events.
            22: 'Yammer', // Yammer events.
            23: 'SkypeForBusinessCmdlets', // Skype for Business events.
            24: 'Discovery', // Events for eDiscovery activities performed by running content searches and managing eDiscovery cases in the Security & Compliance Center.
            25: 'MicrosoftTeams', // Events from Microsoft Teams.
            28: 'ThreatIntelligence', // Phishing and malware events from Exchange Online Protection and Office 365 Advanced Threat Protection.
            30: 'MicrosoftFlow', // Microsoft Power Automate (formerly called Microsoft Flow) events.
            31: 'AeD', // Advanced eDiscovery events.
            32: 'MicrosoftStream', // Microsoft Stream events.
            33: 'ComplianceDLPSharePointClassification', // Events related to DLP classification in SharePoint.
            35: 'Project', // Microsoft Project events.
            36: 'SharePointListOperation', // SharePoint List events.
            38: 'DataGovernance', // Events related to retention policies and retention labels in the Security & Compliance Center
            40: 'SecurityComplianceAlerts', // Security and compliance alert signals.
            41: 'ThreatIntelligenceUrl', // Safe links time-of-block and block override events from Office 365 Advanced Threat Protection.
            42: 'SecurityComplianceInsights', // Events related to insights and reports in the Office 365 security and compliance center.
            44: 'WorkplaceAnalytics', // Workplace Analytics events.
            45: 'PowerAppsApp', // Power Apps events.
            47: 'ThreatIntelligenceAtpContent', // Phishing and malware events for files in SharePoint, OneDrive for Business, and Microsoft Teams from Office 365 Advanced Threat Protection.
            49: 'TeamsHealthcare', // Events related to the Patients application in Microsoft Teams for Healthcare.
            52: 'DataInsightsRestApiAudit', // Data Insights REST API events.
            54: 'SharePointListItemOperation', // SharePoint list item events.
            55: 'SharePointContentTypeOperation', // SharePoint list content type events.
            56: 'SharePointFieldOperation', // SharePoint list field events.
            64: 'AirInvestigation', // Automated incident response (AIR) events.
            66: 'MicrosoftForms', // Microsoft Forms events.
        },
    }));

    builder.Add("setEventFields", new processor.AddFields({
        target: 'event',
        fields: {
            kind: 'event',
            type: 'info',
            // Not so sure about web as a default category:
            category: 'web',
        },
    }));

    builder.Add("mapEventOutcome", makeMapper({
        from: 'o365audit.ResultStatus',
        to: 'event.outcome',
        lowercase: true,
        default: 'success',
        mappings: {
            'success': 'success', // This one is necessary to map Success
            'succeeded': 'success',
            'partiallysucceeded': 'success',
            'true': 'success',
            'failed': 'failure',
            'false': 'failure',
        },
    }));

    builder.Add("makeParametersDict", makeObjFromNameValuePairArray({
        from: 'o365audit.Parameters',
        to: 'o365audit.Parameters',
    }));

    builder.Add("makeExtendedPropertiesDict", makeObjFromNameValuePairArray({
        from: 'o365audit.ExtendedProperties',
        to: 'o365audit.ExtendedProperties',
    }));

    builder.Add("makeModifiedPropertyDict", makeDictFromModifiedPropertyArray({
        from: 'o365audit.ModifiedProperties',
        to: 'o365audit.ModifiedProperties',
    }));

    // Turn AlertLinks into an array of keyword instead of array of objects.
    builder.Add("alertLinks", function (evt) {
        var list = evt.Get("o365audit.AlertLinks");
        if (list == null || !(list instanceof Array)) return;
        var links = [];
        for (var i=0; i<list.length; i++) {
            var link = list[i].AlertLinkHref;
            if (link != null && typeof link === "string" && link.length > 0) {
                links.push(link);
            }
        }
        switch (links.length) {
            case 0:
                evt.Delete('o365audit.AlertLinks');
                break;
            case 1:
                evt.Put("o365audit.AlertLinks", links[0]);
                break;
            default:
                evt.Put("o365audit.AlertLinks", links);
        }
    });

    // Populate event specific fields.
    var dlp = dataLossPreventionSchema(debug);
    builder.Add("productSpecific", makeConditional({
        condition: function(event) {
            return event.Get("event.code");
        },
        'ExchangeAdmin': exchangeAdminSchema(debug).Run,
        'ExchangeItem': exchangeMailboxSchema(debug).Run,
        'AzureActiveDirectory': azureADSchema(debug).Run,
        'AzureActiveDirectoryStsLogon': azureADLogonSchema(debug).Run,
        'SharePointFileOperation': sharePointFileOperationSchema(debug).Run,
        'SecurityComplianceAlerts': securityComplianceAlertsSchema(debug).Run,
        'ComplianceDLPSharePoint': dlp.Run,
        'ComplianceDLPExchange': dlp.Run,
        'Yammer': yammerSchema(debug).Run,
        'MicrosoftTeams': teamsSchema(debug).Run,
    }));

    builder.Add("extractClientIPPortBrackets", new processor.Dissect({
        tokenizer: '[%{_ip}]:%{port}',
        field: 'client.address',
        target_prefix: 'client',
        'when.and': [
            {'not.has_fields': ['client._ip', 'client.port']},
            {'contains.client.address': ']:'},
        ],
    }));
    builder.Add("extractClientIPv4Port", new processor.Dissect({
        tokenizer: '%{_ip}:%{port}',
        field: 'client.address',
        target_prefix: 'client',
        'when.and': [
            {'not.has_fields': ['client._ip', 'client.port']},
            {'contains.client.address': '.'},
            {'contains.client.address': ':'},
            // Best effort to avoid parsing IPv6-mapped IPv4 as ip:port.
            // Won't succeed if IPv6 address is not shortened.
            {'not.contains.client.address': '::'},
        ],
    }));

    // Copy the client/server.address to .ip fields if they are valid IPs.
    builder.Add("convertIPs", new processor.Convert({
        fields: [
            {from: "client.address", to: "client.ip", type: "ip"},
            {from: "server.address", to: "server.ip", type: "ip"},
            {from: "client._ip",     to: "client.ip", type: "ip"},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));
    builder.Add("removeTempIP", function (evt) {
        evt.Delete("client._ip");
    });
    builder.Add("setSrcDstFields", new processor.Convert({
        fields: [
            {from: "client.ip", to: "source.ip"},
            {from: "client.port", to: "source.port"},
            {from: "server.ip", to: "destination.ip"},
        ],
        ignore_missing: true,
        fail_on_error: false
    }));

    [
      'user',
      'user.target',
      'source.user',
      'destination.user',
    ].forEach(function (prefix) {
        builder.Add('setFromID' + prefix, splitEmailUserID(prefix));
    })

    builder.Add("setNetworkType", function(event) {
        var ip = event.Get("client.ip");
        if (ip == null) return;
        event.Put("network.type", ip.indexOf(".") !== -1? "ipv4" : "ipv6");
    });

    builder.Add("setRelatedIP", appendFields({
        fields: [
            "client.ip",
            "server.ip",
        ],
        to: 'related.ip'
    }));

    builder.Add("setRelatedUser", appendFields({
        fields: [
            "user.name",
            "user.target.name",
            "file.owner",
        ],
        to: 'related.user'
    }));

    // Set user-agent from an alternative location.
    builder.Add("altUserAgent", function(evt) {
        var ext = evt.Get("o365audit.ExtendedProperties.UserAgent");
        if (ext != null) evt.Put("user_agent.original", ext);
    });

    // Set host.name to the O365 tenant. This is necessary to aggregate events
    // in SIEM app based on the tenant instead of the host where Filebeat is
    // running.
    builder.Add("setHostName", function(evt) {
        var value;
        if ((value=evt.Get("organization.id"))!=null) {
            value = value.toLowerCase();
            evt.Put("host.id", value);
            // Use tenant name provided in the configuration.
            if (value in tenant_names && value !== "") {
                evt.Put("organization.name", value);
                evt.Put("host.name", tenant_names[value]);
                return;
            }
        }
        if ((value=evt.Get("organization.name"))!=null ||
            (value=evt.Get("user.domain")) != null ) {
            evt.Put("host.name", value);
        }
    });

    builder.Add("saveRaw", new processor.Convert({
        fields: [
            {from: "o365audit", to: "o365.audit"},
        ],
        mode: "rename"
    }));

    var chain = builder.Build();
    return {
        process: chain.Run
    };
}


var audit;

// Register params from configuration.
function register(params) {
    var tenant_names = {};
    if (params.tenants != null) {
        for (var i = 0; i < params.tenants.length; i++) {
            tenant_names[params.tenants[i].id] = params.tenants[i].name.toLowerCase();
        }
    }
    audit = new AuditProcessor(tenant_names, params.debug);
}

function process(evt) {
    return audit.process(evt);
}
