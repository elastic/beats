// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var groups = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.category", ["iam"]);
        evt.Put("event.type", ["group"]);
        switch (evt.Get("event.action")) {
            case "change_basic_setting":
            case "change_identity_setting":
            case "change_info_setting":
            case "change_new_members_restrictions_setting":
            case "change_post_replies_setting":
            case "change_spam_moderation_setting":
            case "change_topic_setting":
                evt.AppendTo("event.category", "configuration");
                evt.AppendTo("event.type", "change");
                break;
            case "change_acl_permission":
                evt.AppendTo("event.type", "change");
                break;
            case "accept_invitation":
                evt.AppendTo("event.type", "info");
                evt.AppendTo("event.type", "user");
                break;
            case "approve_join_request":
            case "join":
                evt.AppendTo("event.type", "user");
                evt.AppendTo("event.type", "change");
                break;
            case "request_to_join":
            case "ban_user_with_moderation":
            case "revoke_invitation":
            case "invite_user":
            case "reject_join_request":
            case "reinvite_user":
                evt.AppendTo("event.type", "info");
                evt.AppendTo("event.type", "user");
                break;
            case "create_group":
                evt.AppendTo("event.type", "creation");
                break;
            case "add_info_setting":
                evt.AppendTo("event.category", "configuration");
                evt.AppendTo("event.type", "creation");
                break;
            case "delete_group":
                evt.AppendTo("event.type", "deletion");
                break;
            case "remove_info_setting":
                evt.AppendTo("event.category", "configuration");
                evt.AppendTo("event.type", "deletion");
                break;
            case "moderate_message":
            case "always_post_from_user":
                evt.AppendTo("event.type", "info");
                break;
            case "add_user":
                evt.AppendTo("event.type", "creation");
                evt.AppendTo("event.type", "user");
                break;
            case "remove_user":
                evt.AppendTo("event.type", "deletion");
                evt.AppendTo("event.type", "user");
                break;
        }
    };

    var getParamValue = function(param) {
        if (param.value) {
            return param.value;
        }
        if (param.multiValue) {
            return param.multiValue;
        }
    };

    var flattenParams = function(evt) {
        var params = evt.Get("json.events.parameters");
        if (!params || !Array.isArray(params)) {
            return;
        }

        params.forEach(function(p){
            evt.Put("gsuite.groups."+p.name, getParamValue(p));
        });

        evt.Delete("json.events.parameters");
    };

    var setOutcome = function(evt) {
        switch (evt.Get("gsuite.groups.status")) {
            case "failed":
                evt.Put("event.outcome", "failure");
                break;
            case "succeeded":
                evt.Put("event.outcome", "success");
                break;
        }
    };

    var setGroupInfo = function(evt) {
        var email = evt.Get("gsuite.groups.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.Put("group.name", data[0]);
        evt.Put("group.domain", data[1]);
    };

    var setRelatedMemberInfo = function(evt) {
        var email = evt.Get("gsuite.groups.member.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.AppendTo("related.user", data[0]);
        evt.Put("user.target.name", data[0]);
        evt.Put("user.target.domain", data[1]);
        evt.Put("user.target.email", email);
        var groupName = evt.Get("group.name");
        if (groupName) {
            evt.Put("user.target.group.name", groupName);
        }
        var groupDomain = evt.Get("group.domain");
        if (groupDomain) {
            evt.Put("user.target.group.domain", groupDomain);
        }
    };

    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(flattenParams)
        .Convert({
            fields: [
                {
                    from: "gsuite.groups.group_email",
                    to: "gsuite.groups.email",
                },
                {
                    from: "gsuite.groups.new_value_repeated",
                    to: "gsuite.groups.new_value",
                },
                {
                    from: "gsuite.groups.old_value_repeated",
                    to: "gsuite.groups.old_value",
                },
                {
                    from: "gsuite.groups.user_email",
                    to: "gsuite.groups.member.email",
                },
                {
                    from: "gsuite.groups.basic_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.identity_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.info_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.new_members_restrictions_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.post_replies_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.spam_moderation_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.topic_setting",
                    to: "gsuite.groups.setting",
                },
                {
                    from: "gsuite.groups.message_id",
                    to: "gsuite.groups.message.id",
                },
                {
                    from: "gsuite.groups.message_moderation_action",
                    to: "gsuite.groups.message.moderation_action",
                },
                {
                    from: "gsuite.groups.member_role",
                    to: "gsuite.groups.member.role",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setOutcome)
        .Add(setGroupInfo)
        .Add(setRelatedMemberInfo)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return groups.process(evt);
}
