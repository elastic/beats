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
            case "CHANGE_CHROME_OS_ANDROID_APPLICATION_SETTING":
            case "CHANGE_DEVICE_STATE":
            case "CHANGE_CHROME_OS_APPLICATION_SETTING":
            case "CHANGE_CHROME_OS_DEVICE_ANNOTATION":
            case "CHANGE_CHROME_OS_DEVICE_SETTING":
            case "CHANGE_CHROME_OS_DEVICE_STATE":
            case "CHANGE_CHROME_OS_PUBLIC_SESSION_SETTING":
            case "UPDATE_CHROME_OS_PRINT_SERVER":
            case "UPDATE_CHROME_OS_PRINTER":
            case "CHANGE_CHROME_OS_SETTING":
            case "CHANGE_CHROME_OS_USER_SETTING":
            case "MOVE_DEVICE_TO_ORG_UNIT_DETAILED":
            case "UPDATE_DEVICE":
            case "SEND_CHROME_OS_DEVICE_COMMAND":
            case "CHANGE_CONTACTS_SETTING":
            case "ASSIGN_ROLE":
            case "ADD_PRIVILEGE":
            case "REMOVE_PRIVILEGE":
            case "RENAME_ROLE":
            case "UPDATE_ROLE":
            case "UNASSIGN_ROLE":
            case "TRANSFER_DOCUMENT_OWNERSHIP":
            case "CHANGE_DOCS_SETTING":
            case "CHANGE_SITES_SETTING":
            case "CHANGE_SITES_WEB_ADDRESS_MAPPING_UPDATES":
            case "ORG_USERS_LICENSE_ASSIGNMENT":
            case "ORG_ALL_USERS_LICENSE_ASSIGNMENT":
            case "USER_LICENSE_ASSIGNMENT":
            case "CHANGE_LICENSE_AUTO_ASSIGN":
            case "USER_LICENSE_REASSIGNMENT":
            case "ORG_LICENSE_REVOKE":
            case "USER_LICENSE_REVOKE":
            case "UPDATE_DYNAMIC_LICENSE":
                evt.Put("event.type", ["change"]);
                break;
            case "CREATE_APPLICATION_SETTING":
            case "CREATE_MANAGED_CONFIGURATION":
            case "CREATE_BUILDING":
            case "CREATE_CALENDAR_RESOURCE":
            case "CREATE_CALENDAR_RESOURCE_FEATURE":
            case "MEET_INTEROP_CREATE_GATEWAY":
            case "INSERT_CHROME_OS_PRINT_SERVER":
            case "INSERT_CHROME_OS_PRINTER":
            case "CREATE_ROLE":
            case "ADD_WEB_ADDRESS":
                evt.Put("event.type", ["creation"]);
                break;
            case "DELETE_APPLICATION_SETTING":
            case "DELETE_MANAGED_CONFIGURATION":
            case "DELETE_BUILDING":
            case "DELETE_CALENDAR_RESOURCE":
            case "DELETE_CALENDAR_RESOURCE_FEATURE":
            case "MEET_INTEROP_DELETE_GATEWAY":
            case "DELETE_CHROME_OS_PRINT_SERVER":
            case "DELETE_CHROME_OS_PRINTER":
            case "REMOVE_CHROME_OS_APPLICATION_SETTINGS":
            case "DELETE_ROLE":
            case "DELETE_WEB_ADDRESS":
                evt.Put("event.type", ["deletion"]);
                break;
            case "DELETE_GROUP":
                evt.Put("event.type", ["group", "creation"]);
                break;
            case "CREATE_GROUP":
                evt.Put("event.type", ["group", "creation"]);
                break;
            case "REORDER_GROUP_BASED_POLICIES_EVENT":
            case "CHANGE_GROUP_DESCRIPTION":
            case "ADD_GROUP_MEMBER":
            case "REMOVE_GROUP_MEMBER":
            case "UPDATE_GROUP_MEMBER":
            case "UPDATE_GROUP_MEMBER_DELIVERY_SETTINGS":
            case "UPDATE_GROUP_MEMBER_DELIVERY_SETTINGS_CAN_EMAIL_OVERRIDE":
            case "CHANGE_GROUP_NAME":
            case "CHANGE_GROUP_SETTING":
            case "GROUP_MEMBER_BULK_UPLOAD":
            case "WHITELISTED_GROUPS_UPDATED":
                evt.Put("event.type", ["group", "change"]);
                break;
            case "ISSUE_DEVICE_COMMAND":
            case "DRIVE_DATA_RESTORE":
            case "VIEW_SITE_DETAILS":
            case "GROUP_LIST_DOWNLOAD":
            case "GROUP_MEMBERS_DOWNLOAD":
                evt.Put("event.type", ["info"]);
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

    var setEventDuration = function(evt) {
        var start = evt.Get("event.start");
        var end = evt.Get("event.end");
        if (!start || !end) {
            return;
        }

        var millisToNano = 1e6;
        var tsStart = Date.parse(start) * millisToNano;
        var tsEnd = Date.parse(end) * millisToNano;

        evt.Put("event.duration", tsEnd-tsStart);
    };

    var setEventOutcome = function(evt) {
        var failed = evt.Get("gsuite.admin.group.bulk_upload.failed");
        if (failed === null) {
            return;
        }

        if (failed === 0) {
            evt.Put("event.outcome", "success");
        } else {
            evt.Put("event.outcome", "failure");
        }
    };

    var setGroupAllowedlist = function(evt) {
        var allowedList = evt.Get("gsuite.admin.WHITELISTED_GROUPS");
        if (!allowedList) {
            return;
        }

        evt.Put("gsuite.admin.group.allowed_list", allowedList.split(","));
        evt.Delete("gsuite.admin.WHITELISTED_GROUPS");
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
                    to: "gsuite.admin.app.package_id",
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
                {
                    from: "gsuite.admin.APP_ID",
                    to: "gsuite.admin.app.id",
                },
                {
                    from: "gsuite.admin.CHROME_OS_SESSION_TYPE",
                    to: "gsuite.admin.chrome_os.session_type",
                },
                {
                    from: "gsuite.admin.DEVICE_NEW_STATE",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.DEVICE_PREVIOUS_STATE",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.DEVICE_SERIAL_NUMBER",
                    to: "gsuite.admin.device.serial_number",
                },
                {
                    from: "gsuite.admin.DEVICE_TYPE",
                    to: "gsuite.admin.device.type",
                },
                {
                    from: "gsuite.admin.PRINT_SERVER_NAME",
                    to: "gsuite.admin.print_server.name",
                },
                {
                    from: "gsuite.admin.PRINTER_NAME",
                    to: "gsuite.admin.printer.name",
                },
                {
                    from: "gsuite.admin.DEVICE_COMMAND_DETAILS",
                    to: "gsuite.admin.device.command_details",
                },
                {
                    from: "gsuite.admin.DEVICE_NEW_ORG_UNIT",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.DEVICE_PREVIOUS_ORG_UNIT",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.ROLE_NAME",
                    to: "gsuite.admin.role.name",
                },
                {
                    from: "gsuite.admin.ROLE_ID",
                    to: "gsuite.admin.role.id",
                },
                {
                    from: "gsuite.admin.PRIVILEGE_NAME",
                    to: "gsuite.admin.privilege.name",
                },
                {
                    from: "gsuite.admin.BEGIN_DATE_TIME",
                    to: "event.start",
                },
                {
                    from: "gsuite.admin.END_DATE_TIME",
                    to: "event.end",
                },
                {
                    from: "gsuite.admin.SITE_LOCATION",
                    to: "url.path",
                },
                {
                    from: "gsuite.admin.WEB_ADDRESS",
                    to: "url.full",
                },
                {
                    from: "gsuite.admin.SITE_NAME",
                    to: "gsuite.admin.url.name",
                },
                {
                    from: "gsuite.admin.SERVICE_NAME",
                    to: "gsuite.admin.service.name",
                },
                {
                    from: "gsuite.admin.PRODUCT_NAME",
                    to: "gsuite.admin.product.name",
                },
                {
                    from: "gsuite.admin.SKU_NAME",
                    to: "gsuite.admin.product.sku",
                },
                {
                    from: "gsuite.admin.GROUP_MEMBER_BULK_UPLOAD_FAILED_NUMBER",
                    to: "gsuite.admin.group.bulk_upload.failed",
                    type: "long",
                },
                {
                    from: "gsuite.admin.GROUP_MEMBER_BULK_UPLOAD_TOTAL_NUMBER",
                    to: "gsuite.admin.group.bulk_upload.total",
                    type: "long",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(setGroupInfo)
        .Add(setRelatedUserInfo)
        .Add(setEventDuration)
        .Add(setEventOutcome)
        .Add(setGroupAllowedlist)
        .Build();

    return {
        process: pipeline.Run,
    };
}());

function process(evt) {
    return login.process(evt);
}
