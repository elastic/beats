// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var login = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        evt.Put("event.category", ["iam"]);
        switch (evt.Get("event.action")) {
            case "CHANGE_APPLICATION_SETTING":
            case "UPDATE_MANAGED_CONFIGURATION":
            case "GPLUS_PREMIUM_FEATURES":
            case "FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTED":
            case "UPDATE_BUILDING":
            case "UPDATE_CALENDAR_RESOURCE_FEATURE":
            case "RENAME_CALENDAR_RESOURCE":
            case "UPDATE_CALENDAR_RESOURCE":
            case "CHANGE_CALENDAR_SETTING":
            case "CANCEL_CALENDAR_EVENTS":
            case "RELEASE_CALENDAR_RESOURCES":
            case "MEET_INTEROP_MODIFY_GATEWAY":
            case "CHANGE_CHAT_SETTING":
                evt.Put("event.type", ["change"]);
                break;
            case "CREATE_APPLICATION_SETTING":
            case "CREATE_MANAGED_CONFIGURATION":
            case "CREATE_BUILDING":
            case "CREATE_CALENDAR_RESOURCE":
            case "CREATE_CALENDAR_RESOURCE_FEATURE":
            case "MEET_INTEROP_CREATE_GATEWAY":
                evt.Put("event.type", ["creation"]);
                break;
            case "DELETE_APPLICATION_SETTING":
            case "DELETE_MANAGED_CONFIGURATION":
            case "DELETE_BUILDING":
            case "DELETE_CALENDAR_RESOURCE":
            case "DELETE_CALENDAR_RESOURCE_FEATURE":
            case "MEET_INTEROP_DELETE_GATEWAY":
                evt.Put("event.type", ["deletion"]);
                break;
            case "REORDER_GROUP_BASED_POLICIES_EVENT":
                evt.Put("event.type", ["group", "change"]);
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
            evt.Put("gsuite.admin."+p.name, getParamValue(p));
        });

        evt.Delete("json.events.parameters");
    };

    var setGroupInfo = function(evt) {
        var email = evt.Get("gsuite.admin.group.email");
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

    var setRelatedUserInfo = function(evt) {
        var email = evt.Get("gsuite.admin.user.email");
        if (!email) {
            return;
        }

        var data = email.split("@");
        if (data.length !== 2) {
            return;
        }

        evt.AppendTo("related.user", data[0]);
    };


    var pipeline = new processor.Chain()
        .Add(categorizeEvent)
        .Add(flattenParams)
        .Convert({
            fields: [
                {
                    from: "gsuite.admin.APPLICATION_EDITION",
                    to: "gsuite.admin.application.edition",
                },
                {
                    from: "gsuite.admin.APPLICATION_NAME",
                    to: "gsuite.admin.application.name",
                },
                {
                    from: "gsuite.admin.GROUP_EMAIL",
                    to: "gsuite.admin.group.email",
                },
                {
                    from: "gsuite.admin.NEW_VALUE",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.OLD_VALUE",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.ORG_UNIT_NAME",
                    to: "gsuite.admin.org_unit.name",
                },
                {
                    from: "gsuite.admin.SETTING_NAME",
                    to: "gsuite.admin.setting",
                },
                {
                    from: "gsuite.admin.GROUP_PRIORITIES",
                    to: "gsuite.admin.group.priorities",
                },
                {
                    from: "gsuite.admin.DOMAIN_NAME",
                    to: "gsuite.admin.domain",
                },
                {
                    from: "gsuite.admin.MANAGED_CONFIGURATION_NAME",
                    to: "gsuite.admin.managed_configuration",
                },
                {
                    from: "gsuite.admin.MOBILE_APP_PACKAGE_ID",
                    to: "gsuite.admin.mobile_app.package_id",
                },
                {
                    from: "gsuite.admin.FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTION",
                    to: "gsuite.admin.non_featured_services_selection",
                },
                {
                    from: "gsuite.admin.FIELD_NAME",
                    to: "gsuite.admin.field",
                },
                {
                    from: "gsuite.admin.RESOURCE_IDENTIFIER",
                    to: "gsuite.admin.resource.id",
                },
                {
                    from: "gsuite.admin.USER_EMAIL",
                    to: "gsuite.admin.user.email",
                },
                {
                    from: "gsuite.admin.GATEWAY_NAME",
                    to: "gsuite.admin.gateway.name",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setGroupInfo)
        .Add(setRelatedUserInfo)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return login.process(evt);
}
