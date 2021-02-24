// Copyright Elasticsearch B.V. and/or licensed to Elasticsearch B.V. under one
// or more contributor license agreements. Licensed under the Elastic License;
// you may not use this file except in compliance with the Elastic License.

var login = (function () {
    var processor = require("processor");

    var categorizeEvent = function(evt) {
        // not convinced that these should be iam
        evt.Put("event.category", ["iam"]);
        switch (evt.Get("event.action")) {
            case "CHANGE_APPLICATION_SETTING":
            case "UPDATE_MANAGED_CONFIGURATION":
            case "CHANGE_CALENDAR_SETTING":
            case "CHANGE_CHAT_SETTING":
            case "CHANGE_CHROME_OS_ANDROID_APPLICATION_SETTING":
            case "GPLUS_PREMIUM_FEATURES":
            case "UPDATE_CALENDAR_RESOURCE_FEATURE":
            case "FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTED":
            case "MEET_INTEROP_MODIFY_GATEWAY":
            case "CHANGE_CHROME_OS_APPLICATION_SETTING":
            case "CHANGE_CHROME_OS_DEVICE_SETTING":
            case "CHANGE_CHROME_OS_PUBLIC_SESSION_SETTING":
            case "CHANGE_CHROME_OS_SETTING":
            case "CHANGE_CHROME_OS_USER_SETTING":
            case "CHANGE_CONTACTS_SETTING":
            case "CHANGE_DOCS_SETTING":
            case "CHANGE_SITES_SETTING":
            case "CHANGE_EMAIL_SETTING":
            case "CHANGE_GMAIL_SETTING":
            case "ALLOW_STRONG_AUTHENTICATION":
            case "ALLOW_SERVICE_FOR_OAUTH2_ACCESS":
            case "DISALLOW_SERVICE_FOR_OAUTH2_ACCESS":
            case "CHANGE_APP_ACCESS_SETTINGS_COLLECTION_ID":
            case "CHANGE_TWO_STEP_VERIFICATION_ENROLLMENT_PERIOD_DURATION":
            case "CHANGE_TWO_STEP_VERIFICATION_FREQUENCY":
            case "CHANGE_TWO_STEP_VERIFICATION_GRACE_PERIOD_DURATION":
            case "CHANGE_TWO_STEP_VERIFICATION_START_DATE":
            case "CHANGE_ALLOWED_TWO_STEP_VERIFICATION_METHODS":
            case "CHANGE_SITES_WEB_ADDRESS_MAPPING_UPDATES":
            case "ENABLE_NON_ADMIN_USER_PASSWORD_RECOVERY":
            case "ENFORCE_STRONG_AUTHENTICATION":
            case "UPDATE_ERROR_MSG_FOR_RESTRICTED_OAUTH2_APPS":
            case "WEAK_PROGRAMMATIC_LOGIN_SETTINGS_CHANGED":
            case "SESSION_CONTROL_SETTINGS_CHANGE":
            case "CHANGE_SESSION_LENGTH":
            case "TOGGLE_OAUTH_ACCESS_TO_ALL_APIS":
            case "TOGGLE_ALLOW_ADMIN_PASSWORD_RESET":
            case "ENABLE_API_ACCESS":
            case "CHANGE_WHITELIST_SETTING":
            case "COMMUNICATION_PREFERENCES_SETTING_CHANGE":
            case "ENABLE_FEEDBACK_SOLICITATION":
            case "TOGGLE_CONTACT_SHARING":
            case "TOGGLE_USE_CUSTOM_LOGO":
            case "CHANGE_DATA_LOCALIZATION_SETTING":
            case "TOGGLE_ENABLE_OAUTH_CONSUMER_KEY":
            case "TOGGLE_SSO_ENABLED":
            case "TOGGLE_SSL":
            case "TOGGLE_NEW_APP_FEATURES":
            case "TOGGLE_USE_NEXT_GEN_CONTROL_PANEL":
            case "TOGGLE_OPEN_ID_ENABLED":
            case "TOGGLE_OUTBOUND_RELAY":
            case "CHANGE_SSO_SETTINGS":
            case "ENABLE_SERVICE_OR_FEATURE_NOTIFICATIONS":
            case "CHANGE_MOBILE_APPLICATION_SETTINGS":
            case "CHANGE_MOBILE_SETTING":
                evt.AppendTo("event.category", "configuration")
                evt.Put("event.type", ["change"]);
                break;
            case "UPDATE_BUILDING":
            case "RENAME_CALENDAR_RESOURCE":
            case "UPDATE_CALENDAR_RESOURCE":
            case "CANCEL_CALENDAR_EVENTS":
            case "RELEASE_CALENDAR_RESOURCES":
            case "CHANGE_DEVICE_STATE":
            case "CHANGE_CHROME_OS_DEVICE_ANNOTATION":
            case "CHANGE_CHROME_OS_DEVICE_STATE":
            case "UPDATE_CHROME_OS_PRINT_SERVER":
            case "UPDATE_CHROME_OS_PRINTER":
            case "MOVE_DEVICE_TO_ORG_UNIT_DETAILED":
            case "UPDATE_DEVICE":
            case "SEND_CHROME_OS_DEVICE_COMMAND":
            case "ASSIGN_ROLE":
            case "ADD_PRIVILEGE":
            case "REMOVE_PRIVILEGE":
            case "RENAME_ROLE":
            case "UPDATE_ROLE":
            case "UNASSIGN_ROLE":
            case "TRANSFER_DOCUMENT_OWNERSHIP":
            case "ORG_USERS_LICENSE_ASSIGNMENT":
            case "ORG_ALL_USERS_LICENSE_ASSIGNMENT":
            case "USER_LICENSE_ASSIGNMENT":
            case "CHANGE_LICENSE_AUTO_ASSIGN":
            case "USER_LICENSE_REASSIGNMENT":
            case "ORG_LICENSE_REVOKE":
            case "USER_LICENSE_REVOKE":
            case "UPDATE_DYNAMIC_LICENSE":
            case "DROP_FROM_QUARANTINE":
            case "REJECT_FROM_QUARANTINE":
            case "RELEASE_FROM_QUARANTINE":
            case "CHROME_LICENSES_ENABLED":
            case "CHROME_APPLICATION_LICENSE_RESERVATION_UPDATED":
            case "ASSIGN_CUSTOM_LOGO":
            case "UNASSIGN_CUSTOM_LOGO":
            case "REVOKE_ENROLLMENT_TOKEN":
            case "CHROME_LICENSES_ALLOWED":
            case "EDIT_ORG_UNIT_DESCRIPTION":
            case "MOVE_ORG_UNIT":
            case "EDIT_ORG_UNIT_NAME":
            case "REVOKE_DEVICE_ENROLLMENT_TOKEN":
            case "TOGGLE_SERVICE_ENABLED":
            case "ADD_TO_TRUSTED_OAUTH2_APPS":
            case "REMOVE_FROM_TRUSTED_OAUTH2_APPS":
            case "BLOCK_ON_DEVICE_ACCESS":
            case "TOGGLE_CAA_ENABLEMENT":
            case "CHANGE_CAA_ERROR_MESSAGE":
            case "CHANGE_CAA_APP_ASSIGNMENTS":
            case "UNTRUST_DOMAIN_OWNED_OAUTH2_APPS":
            case "TRUST_DOMAIN_OWNED_OAUTH2_APPS":
            case "UNBLOCK_ON_DEVICE_ACCESS":
            case "CHANGE_ACCOUNT_AUTO_RENEWAL":
            case "ADD_APPLICATION":
            case "ADD_APPLICATION_TO_WHITELIST":
            case "CHANGE_ADVERTISEMENT_OPTION":
            case "CHANGE_ALERT_CRITERIA":
            case "ALERT_RECEIVERS_CHANGED":
            case "RENAME_ALERT":
            case "ALERT_STATUS_CHANGED":
            case "ADD_DOMAIN_ALIAS":
            case "REMOVE_DOMAIN_ALIAS":
            case "AUTHORIZE_API_CLIENT_ACCESS":
            case "REMOVE_API_CLIENT_ACCESS":
            case "CHROME_LICENSES_REDEEMED":
            case "TOGGLE_AUTO_ADD_NEW_SERVICE":
            case "CHANGE_PRIMARY_DOMAIN":
            case "CHANGE_CONFLICT_ACCOUNT_ACTION":
            case "CHANGE_CUSTOM_LOGO":
            case "CHANGE_DATA_LOCALIZATION_FOR_RUSSIA":
            case "CHANGE_DATA_PROTECTION_OFFICER_CONTACT_INFO":
            case "CHANGE_DOMAIN_DEFAULT_LOCALE":
            case "CHANGE_DOMAIN_DEFAULT_TIMEZONE":
            case "CHANGE_DOMAIN_NAME":
            case "TOGGLE_ENABLE_PRE_RELEASE_FEATURES":
            case "CHANGE_DOMAIN_SUPPORT_MESSAGE":
            case "ADD_TRUSTED_DOMAINS":
            case "REMOVE_TRUSTED_DOMAINS":
            case "CHANGE_EDU_TYPE":
            case "CHANGE_EU_REPRESENTATIVE_CONTACT_INFO":
            case "CHANGE_LOGIN_BACKGROUND_COLOR":
            case "CHANGE_LOGIN_BORDER_COLOR":
            case "CHANGE_LOGIN_ACTIVITY_TRACE":
            case "PLAY_FOR_WORK_ENROLL":
            case "PLAY_FOR_WORK_UNENROLL":
            case "UPDATE_DOMAIN_PRIMARY_ADMIN_EMAIL":
            case "CHANGE_ORGANIZATION_NAME":
            case "CHANGE_PASSWORD_MAX_LENGTH":
            case "CHANGE_PASSWORD_MIN_LENGTH":
            case "REMOVE_APPLICATION":
            case "REMOVE_APPLICATION_FROM_WHITELIST":
            case "CHANGE_RENEW_DOMAIN_REGISTRATION":
            case "CHANGE_RESELLER_ACCESS":
            case "RULE_ACTIONS_CHANGED":
            case "CHANGE_RULE_CRITERIA":
            case "RENAME_RULE":
            case "RULE_STATUS_CHANGED":
            case "ADD_SECONDARY_DOMAIN":
            case "REMOVE_SECONDARY_DOMAIN":
            case "UPDATE_DOMAIN_SECONDARY_EMAIL":
            case "UPDATE_RULE":
            case "ADD_MOBILE_CERTIFICATE":
            case "COMPANY_OWNED_DEVICE_BLOCKED":
            case "COMPANY_OWNED_DEVICE_UNBLOCKED":
            case "COMPANY_OWNED_DEVICE_WIPED":
            case "CHANGE_MOBILE_APPLICATION_PERMISSION_GRANT":
            case "CHANGE_MOBILE_APPLICATION_PRIORITY_ORDER":
            case "REMOVE_MOBILE_APPLICATION_FROM_WHITELIST":
            case "ADD_MOBILE_APPLICATION_TO_WHITELIST":
            case "CHANGE_ADMIN_RESTRICTIONS_PIN":
            case "CHANGE_MOBILE_WIRELESS_NETWORK":
            case "ADD_MOBILE_WIRELESS_NETWORK":
            case "REMOVE_MOBILE_WIRELESS_NETWORK":
            case "CHANGE_MOBILE_WIRELESS_NETWORK_PASSWORD":
            case "REMOVE_MOBILE_CERTIFICATE":
                evt.Put("event.type", ["change"]);
                break;
            case "CREATE_APPLICATION_SETTING":
            case "CREATE_GMAIL_SETTING":
                evt.AppendTo("event.category", "configuration")
                evt.Put("event.type", ["creation"]);
                break;
            case "CREATE_MANAGED_CONFIGURATION":
            case "CREATE_BUILDING":
            case "CREATE_CALENDAR_RESOURCE":
            case "CREATE_CALENDAR_RESOURCE_FEATURE":
            case "MEET_INTEROP_CREATE_GATEWAY":
            case "INSERT_CHROME_OS_PRINT_SERVER":
            case "INSERT_CHROME_OS_PRINTER":
            case "CREATE_ROLE":
            case "ADD_WEB_ADDRESS":
            case "EMAIL_UNDELETE":
            case "CHROME_APPLICATION_LICENSE_RESERVATION_CREATED":
            case "CREATE_DEVICE_ENROLLMENT_TOKEN":
            case "CREATE_ENROLLMENT_TOKEN":
            case "CREATE_ORG_UNIT":
            case "CREATE_ALERT":
            case "CREATE_PLAY_FOR_WORK_TOKEN":
            case "GENERATE_TRANSFER_TOKEN":
            case "REGENERATE_OAUTH_CONSUMER_SECRET":
            case "CREATE_RULE":
            case "GENERATE_PIN":
            case "COMPANY_DEVICES_BULK_CREATION":
                evt.Put("event.type", ["creation"]);
                break;
            case "DELETE_APPLICATION_SETTING":
            case "DELETE_GMAIL_SETTING":
                evt.AppendTo("event.category", "configuration")
                evt.Put("event.type", ["deletion"]);
                break;
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
            case "CHROME_APPLICATION_LICENSE_RESERVATION_DELETED":
            case "REMOVE_ORG_UNIT":
            case "DELETE_ALERT":
            case "DELETE_PLAY_FOR_WORK_TOKEN":
            case "DELETE_RULE":
            case "COMPANY_DEVICE_DELETION":
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
            case "REVOKE_3LO_DEVICE_TOKENS":
            case "REVOKE_3LO_TOKEN":
            case "ADD_RECOVERY_EMAIL":
            case "ADD_RECOVERY_PHONE":
            case "GRANT_ADMIN_PRIVILEGE":
            case "REVOKE_ADMIN_PRIVILEGE":
            case "REVOKE_ASP":
            case "TOGGLE_AUTOMATIC_CONTACT_SHARING":
            case "CANCEL_USER_INVITE":
            case "CHANGE_USER_CUSTOM_FIELD":
            case "CHANGE_USER_EXTERNAL_ID":
            case "CHANGE_USER_GENDER":
            case "CHANGE_USER_IM":
            case "ENABLE_USER_IP_WHITELIST":
            case "CHANGE_USER_KEYWORD":
            case "CHANGE_USER_LANGUAGE":
            case "CHANGE_USER_LOCATION":
            case "CHANGE_USER_ORGANIZATION":
            case "CHANGE_USER_PHONE_NUMBER":
            case "CHANGE_RECOVERY_EMAIL":
            case "CHANGE_RECOVERY_PHONE":
            case "CHANGE_USER_RELATION":
            case "CHANGE_USER_ADDRESS":
            case "GRANT_DELEGATED_ADMIN_PRIVILEGES":
            case "CHANGE_FIRST_NAME":
            case "GMAIL_RESET_USER":
            case "CHANGE_LAST_NAME":
            case "MAIL_ROUTING_DESTINATION_ADDED":
            case "MAIL_ROUTING_DESTINATION_REMOVED":
            case "ADD_NICKNAME":
            case "REMOVE_NICKNAME":
            case "CHANGE_PASSWORD":
            case "CHANGE_PASSWORD_ON_NEXT_LOGIN":
            case "REMOVE_RECOVERY_EMAIL":
            case "REMOVE_RECOVERY_PHONE":
            case "RESET_SIGNIN_COOKIES":
            case "SECURITY_KEY_REGISTERED_FOR_USER":
            case "REVOKE_SECURITY_KEY":
            case "TURN_OFF_2_STEP_VERIFICATION":
            case "UNBLOCK_USER_SESSION":
            case "UNENROLL_USER_FROM_TITANIUM":
            case "ARCHIVE_USER":
            case "UPDATE_BIRTHDATE":
            case "DOWNGRADE_USER_FROM_GPLUS":
            case "USER_ENROLLED_IN_TWO_STEP_VERIFICATION":
            case "MOVE_USER_TO_ORG_UNIT":
            case "USER_PUT_IN_TWO_STEP_VERIFICATION_GRACE_PERIOD":
            case "RENAME_USER":
            case "UNENROLL_USER_FROM_STRONG_AUTH":
            case "SUSPEND_USER":
            case "UNARCHIVE_USER":
            case "UNSUSPEND_USER":
            case "UPGRADE_USER_TO_GPLUS":
            case "MOBILE_DEVICE_APPROVE":
            case "MOBILE_DEVICE_BLOCK":
            case "MOBILE_DEVICE_WIPE":
            case "MOBILE_ACCOUNT_WIPE":
            case "MOBILE_DEVICE_CANCEL_WIPE_THEN_APPROVE":
            case "MOBILE_DEVICE_CANCEL_WIPE_THEN_BLOCK":
                evt.Put("event.type", ["user", "change"]);
                break;
            case "DELETE_2SV_SCRATCH_CODES":
            case "DELETE_ACCOUNT_INFO_DUMP":
            case "DELETE_EMAIL_MONITOR":
            case "DELETE_MAILBOX_DUMP":
            case "DELETE_USER":
            case "MOBILE_DEVICE_DELETE":
                evt.Put("event.type", ["user", "deletion"]);
                break;
            case "GENERATE_2SV_SCRATCH_CODES":
            case "CREATE_EMAIL_MONITOR":
            case "CREATE_DATA_TRANSFER_REQUEST":
            case "CREATE_USER":
            case "UNDELETE_USER":
                evt.Put("event.type", ["user", "creation"]);
                break;
            case "ISSUE_DEVICE_COMMAND":
            case "DRIVE_DATA_RESTORE":
            case "VIEW_SITE_DETAILS":
            case "EMAIL_LOG_SEARCH":
            case "SKIP_DOMAIN_ALIAS_MX":
            case "VERIFY_DOMAIN_ALIAS_MX":
            case "VERIFY_DOMAIN_ALIAS":
            case "VIEW_DNS_LOGIN_DETAILS":
            case "MX_RECORD_VERIFICATION_CLAIM":
            case "UPLOAD_OAUTH_CERTIFICATE":
            case "SKIP_SECONDARY_DOMAIN_MX":
            case "VERIFY_SECONDARY_DOMAIN_MX":
            case "VERIFY_SECONDARY_DOMAIN":
            case "BULK_UPLOAD":
            case "DOWNLOAD_PENDING_INVITES_LIST":
            case "DOWNLOAD_USERLIST_CSV":
            case "USERS_BULK_UPLOAD":
            case "ENROLL_FOR_GOOGLE_DEVICE_MANAGEMENT":
            case "USE_GOOGLE_MOBILE_MANAGEMENT":
            case "USE_GOOGLE_MOBILE_MANAGEMENT_FOR_NON_IOS":
            case "USE_GOOGLE_MOBILE_MANAGEMENT_FOR_IOS":
                evt.Put("event.type", ["info"]);
                break;
            case "GROUP_LIST_DOWNLOAD":
            case "GROUP_MEMBERS_DOWNLOAD":
                evt.Put("event.type", ["group", "info"]);
                break;
            case "REQUEST_ACCOUNT_INFO":
            case "REQUEST_MAILBOX_DUMP":
            case "RESEND_USER_INVITE":
            case "BULK_UPLOAD_NOTIFICATION_SENT":
            case "USER_INVITE":
            case "VIEW_TEMP_PASSWORD":
            case "USERS_BULK_UPLOAD_NOTIFICATION_SENT":
            case "ACTION_CANCELLED":
            case "ACTION_REQUESTED":
                evt.Put("event.type", ["user", "info"]);
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
        if (param.intValue !== null) {
            return param.intValue;
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

    var setEventDuration = function(evt) {
        var start = evt.Get("event.start");
        var end = evt.Get("event.end");
        if (!start || !end) {
            return;
        }

        evt.Put("event.duration", end.UnixNano() - start.UnixNano());
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

    var deleteField = function(field) {
        return function(evt) {
            evt.Delete(field);
        };
    };

    var parseDate = function(field, targetField) {
        return new processor.Chain()
            .Add(new processor.Timestamp({
                field: field,
                target_field: targetField,
                timezone: "UTC",
                layouts: [
                    "2006-01-02T15:04:05Z",
                    "2006-01-02T15:04:05.999Z",
                    "2006/01/02 15:04:05 UTC",
                ],
                tests: [
                    "2020-02-05T18:19:23Z",
                    "2020-02-05T18:19:23.599Z",
                    "2020/07/28 04:59:59 UTC",
                ],
                ignore_missing: true,
            }))
            .Add(deleteField(field))
            .Build()
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
                    from: "gsuite.admin.APPLICATION_ENABLED",
                    to: "gsuite.admin.application.enabled",
                },
                {
                    from: "gsuite.admin.APP_LICENSES_ORDER_NUMBER",
                    to: "gsuite.admin.application.licences_order_number",
                },
                {
                    from: "gsuite.admin.CHROME_NUM_LICENSES_PURCHASED",
                    to: "gsuite.admin.application.licences_purchased",
                    type: "long",
                },
                {
                    from: "gsuite.admin.REAUTH_APPLICATION",
                    to: "gsuite.admin.application.name",
                },
                {
                    from: "gsuite.admin.GROUP_EMAIL",
                    to: "gsuite.admin.group.email",
                },
                {
                    from: "gsuite.admin.GROUP_NAME",
                    to: "group.name",
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
                    to: "gsuite.admin.setting.name",
                },
                {
                    from: "gsuite.admin.SETTING_DESCRIPTION",
                    to: "gsuite.admin.setting.description",
                },
                {
                    from: "gsuite.admin.USER_DEFINED_SETTING_NAME",
                    to: "gsuite.admin.user_defined_setting.name",
                },
                {
                    from: "gsuite.admin.GROUP_PRIORITIES",
                    to: "gsuite.admin.group.priorities",
                },
                {
                    from: "gsuite.admin.DOMAIN_NAME",
                    to: "gsuite.admin.domain.name",
                },
                {
                    from: "gsuite.admin.DOMAIN_ALIAS",
                    to: "gsuite.admin.domain.alias",
                },
                {
                    from: "gsuite.admin.SECONDARY_DOMAIN_NAME",
                    to: "gsuite.admin.domain.secondary_name",
                },
                {
                    from: "gsuite.admin.MANAGED_CONFIGURATION_NAME",
                    to: "gsuite.admin.managed_configuration",
                },
                {
                    from: "gsuite.admin.MOBILE_APP_PACKAGE_ID",
                    to: "gsuite.admin.application.package_id",
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
                    to: "gsuite.admin.application.id",
                },
                {
                    from: "gsuite.admin.ASP_ID",
                    to: "gsuite.admin.application.asp_id",
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
                    from: "gsuite.admin.DEVICE_ID",
                    to: "gsuite.admin.device.id",
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
                    to: "gsuite.admin.bulk_upload.failed",
                    type: "long",
                },
                {
                    from: "gsuite.admin.GROUP_MEMBER_BULK_UPLOAD_TOTAL_NUMBER",
                    to: "gsuite.admin.bulk_upload.total",
                    type: "long",
                },
                {
                    from: "gsuite.admin.BULK_UPLOAD_FAIL_USERS_NUMBER",
                    to: "gsuite.admin.bulk_upload.failed",
                    type: "long",
                },
                {
                    from: "gsuite.admin.BULK_UPLOAD_TOTAL_USERS_NUMBER",
                    to: "gsuite.admin.bulk_upload.total",
                    type: "long",
                },
                {
                    from: "gsuite.admin.EMAIL_LOG_SEARCH_MSG_ID",
                    to: "gsuite.admin.email.log_search_filter.message_id",
                },
                {
                    from: "gsuite.admin.EMAIL_LOG_SEARCH_RECIPIENT",
                    to: "gsuite.admin.email.log_search_filter.recipient.value",
                },
                {
                    from: "gsuite.admin.EMAIL_LOG_SEARCH_SENDER",
                    to: "gsuite.admin.email.log_search_filter.sender.value",
                },
                {
                    from: "gsuite.admin.EMAIL_LOG_SEARCH_SMTP_RECIPIENT_IP",
                    to: "gsuite.admin.email.log_search_filter.recipient.ip",
                    type: "ip",
                },
                {
                    from: "gsuite.admin.EMAIL_LOG_SEARCH_SMTP_SENDER_IP",
                    to: "gsuite.admin.email.log_search_filter.sender.ip",
                    type: "ip",
                },
                {
                    from: "gsuite.admin.QUARANTINE_NAME",
                    to: "gsuite.admin.email.quarantine_name",
                },
                {
                    from: "gsuite.admin.CHROME_LICENSES_ENABLED",
                    to: "gsuite.admin.chrome_licenses.enabled",
                },
                {
                    from: "gsuite.admin.CHROME_LICENSES_ALLOWED",
                    to: "gsuite.admin.chrome_licenses.allowed",
                },
                {
                    from: "gsuite.admin.FULL_ORG_UNIT_PATH",
                    to: "gsuite.admin.org_unit.full",
                },
                {
                    from: "gsuite.admin.OAUTH2_SERVICE_NAME",
                    to: "gsuite.admin.oauth2.service.name",
                },
                {
                    from: "gsuite.admin.OAUTH2_APP_ID",
                    to: "gsuite.admin.oauth2.application.id",
                },
                {
                    from: "gsuite.admin.OAUTH2_APP_NAME",
                    to: "gsuite.admin.oauth2.application.name",
                },
                {
                    from: "gsuite.admin.OAUTH2_APP_TYPE",
                    to: "gsuite.admin.oauth2.application.type",
                },
                {
                    from: "gsuite.admin.ALLOWED_TWO_STEP_VERIFICATION_METHOD",
                    to: "gsuite.admin.verification_method",
                },
                {
                    from: "gsuite.admin.DOMAIN_VERIFICATION_METHOD",
                    to: "gsuite.admin.verification_method",
                },
                {
                    from: "gsuite.admin.CAA_ASSIGNMENTS_NEW",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.CAA_ASSIGNMENTS_OLD",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.REAUTH_SETTING_NEW",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.REAUTH_SETTING_OLD",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.ALERT_NAME",
                    to: "gsuite.admin.alert.name",
                },
                {
                    from: "gsuite.admin.API_CLIENT_NAME",
                    to: "gsuite.admin.api.client.name",
                },
                {
                    from: "gsuite.admin.API_SCOPES",
                    to: "gsuite.admin.api.scopes",
                },
                {
                    from: "gsuite.admin.PLAY_FOR_WORK_TOKEN_ID",
                    to: "gsuite.admin.mdm.token",
                },
                {
                    from: "gsuite.admin.PLAY_FOR_WORK_MDM_VENDOR_NAME",
                    to: "gsuite.admin.mdm.vendor",
                },
                {
                    from: "gsuite.admin.INFO_TYPE",
                    to: "gsuite.admin.info_type",
                },
                {
                    from: "gsuite.admin.RULE_NAME",
                    to: "gsuite.admin.rule.name",
                },
                {
                    from: "gsuite.admin.USER_CUSTOM_FIELD",
                    to: "gsuite.admin.setting.name",
                },
                {
                    from: "gsuite.admin.EMAIL_MONITOR_DEST_EMAIL",
                    to: "gsuite.admin.email_monitor.dest_email",
                },
                {
                    from: "gsuite.admin.EMAIL_MONITOR_LEVEL_CHAT",
                    to: "gsuite.admin.email_monitor.level.chat",
                },
                {
                    from: "gsuite.admin.EMAIL_MONITOR_LEVEL_DRAFT_EMAIL",
                    to: "gsuite.admin.email_monitor.level.draft",
                },
                {
                    from: "gsuite.admin.EMAIL_MONITOR_LEVEL_INCOMING_EMAIL",
                    to: "gsuite.admin.email_monitor.level.incoming",
                },
                {
                    from: "gsuite.admin.EMAIL_MONITOR_LEVEL_OUTGOING_EMAIL",
                    to: "gsuite.admin.email_monitor.level.outgoing",
                },
                {
                    from: "gsuite.admin.EMAIL_EXPORT_INCLUDE_DELETED",
                    to: "gsuite.admin.email_dump.include_deleted",
                },
                {
                    from: "gsuite.admin.EMAIL_EXPORT_PACKAGE_CONTENT",
                    to: "gsuite.admin.email_dump.package_content",
                },
                {
                    from: "gsuite.admin.SEARCH_QUERY_FOR_DUMP",
                    to: "gsuite.admin.email_dump.query",
                },
                {
                    from: "gsuite.admin.DESTINATION_USER_EMAIL",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.REQUEST_ID",
                    to: "gsuite.admin.request.id",
                },
                {
                    from: "gsuite.admin.GMAIL_RESET_REASON",
                    to: "message",
                },
                {
                    from: "gsuite.admin.USER_NICKNAME",
                    to: "gsuite.admin.user.nickname",
                },
                {
                    from: "gsuite.admin.ACTION_ID",
                    to: "gsuite.admin.mobile.action.id",
                },
                {
                    from: "gsuite.admin.ACTION_TYPE",
                    to: "gsuite.admin.mobile.action.type",
                },
                {
                    from: "gsuite.admin.MOBILE_CERTIFICATE_COMMON_NAME",
                    to: "gsuite.admin.mobile.certificate.name",
                },
                {
                    from: "gsuite.admin.NUMBER_OF_COMPANY_OWNED_DEVICES",
                    to: "gsuite.admin.mobile.company_owned_devices",
                    type: "long",
                },
                {
                    from: "gsuite.admin.COMPANY_DEVICE_ID",
                    to: "gsuite.admin.device.id",
                },
                {
                    from: "gsuite.admin.DISTRIBUTION_ENTITY_NAME",
                    to: "gsuite.admin.distribution.entity.name",
                },
                {
                    from: "gsuite.admin.DISTRIBUTION_ENTITY_TYPE",
                    to: "gsuite.admin.distribution.entity.type",
                },
                {
                    from: "gsuite.admin.MOBILE_APP_PACKAGE_ID",
                    to: "gsuite.admin.application.package_id",
                },
                {
                    from: "gsuite.admin.NEW_PERMISSION_GRANT_STATE",
                    to: "gsuite.admin.new_value",
                },
                {
                    from: "gsuite.admin.OLD_PERMISSION_GRANT_STATE",
                    to: "gsuite.admin.old_value",
                },
                {
                    from: "gsuite.admin.PERMISSION_GROUP_NAME",
                    to: "gsuite.admin.setting.name",
                },
                {
                    from: "gsuite.admin.MOBILE_WIRELESS_NETWORK_NAME",
                    to: "network.name",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(parseDate(
            "gsuite.admin.EMAIL_LOG_SEARCH_END_DATE",
            "gsuite.admin.email.log_search_filter.end_date"
        ))
        .Add(parseDate(
            "gsuite.admin.EMAIL_LOG_SEARCH_START_DATE",
            "gsuite.admin.email.log_search_filter.start_date"
        ))
        .Add(parseDate(
            "gsuite.admin.BIRTHDATE",
            "gsuite.admin.user.birthdate"
        ))
        .Add(parseDate(
            "gsuite.admin.BEGIN_DATE_TIME",
            "event.start"
        ))
        .Add(parseDate(
            "gsuite.admin.START_DATE",
            "event.start"
        ))
        .Add(parseDate(
            "gsuite.admin.END_DATE",
            "event.end"
        ))
        .Add(parseDate(
            "gsuite.admin.END_DATE_TIME",
            "event.end"
        ))
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
