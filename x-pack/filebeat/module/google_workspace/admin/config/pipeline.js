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
            evt.Put("google_workspace.admin."+p.name, getParamValue(p));
        });

        evt.Delete("json.events.parameters");
    };

    var setGroupInfo = function(evt) {
        var email = evt.Get("google_workspace.admin.group.email");
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
        var email = evt.Get("google_workspace.admin.user.email");
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
        var failed = evt.Get("google_workspace.admin.group.bulk_upload.failed");
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
        var allowedList = evt.Get("google_workspace.admin.WHITELISTED_GROUPS");
        if (!allowedList) {
            return;
        }

        evt.Put("google_workspace.admin.group.allowed_list", allowedList.split(","));
        evt.Delete("google_workspace.admin.WHITELISTED_GROUPS");
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
                    from: "google_workspace.admin.APPLICATION_EDITION",
                    to: "google_workspace.admin.application.edition",
                },
                {
                    from: "google_workspace.admin.APPLICATION_NAME",
                    to: "google_workspace.admin.application.name",
                },
                {
                    from: "google_workspace.admin.APPLICATION_ENABLED",
                    to: "google_workspace.admin.application.enabled",
                },
                {
                    from: "google_workspace.admin.APP_LICENSES_ORDER_NUMBER",
                    to: "google_workspace.admin.application.licences_order_number",
                },
                {
                    from: "google_workspace.admin.CHROME_NUM_LICENSES_PURCHASED",
                    to: "google_workspace.admin.application.licences_purchased",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.REAUTH_APPLICATION",
                    to: "google_workspace.admin.application.name",
                },
                {
                    from: "google_workspace.admin.GROUP_EMAIL",
                    to: "google_workspace.admin.group.email",
                },
                {
                    from: "google_workspace.admin.GROUP_NAME",
                    to: "group.name",
                },
                {
                    from: "google_workspace.admin.NEW_VALUE",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.OLD_VALUE",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.ORG_UNIT_NAME",
                    to: "google_workspace.admin.org_unit.name",
                },
                {
                    from: "google_workspace.admin.SETTING_NAME",
                    to: "google_workspace.admin.setting.name",
                },
                {
                    from: "google_workspace.admin.SETTING_DESCRIPTION",
                    to: "google_workspace.admin.setting.description",
                },
                {
                    from: "google_workspace.admin.USER_DEFINED_SETTING_NAME",
                    to: "google_workspace.admin.user_defined_setting.name",
                },
                {
                    from: "google_workspace.admin.GROUP_PRIORITIES",
                    to: "google_workspace.admin.group.priorities",
                },
                {
                    from: "google_workspace.admin.DOMAIN_NAME",
                    to: "google_workspace.admin.domain.name",
                },
                {
                    from: "google_workspace.admin.DOMAIN_ALIAS",
                    to: "google_workspace.admin.domain.alias",
                },
                {
                    from: "google_workspace.admin.SECONDARY_DOMAIN_NAME",
                    to: "google_workspace.admin.domain.secondary_name",
                },
                {
                    from: "google_workspace.admin.MANAGED_CONFIGURATION_NAME",
                    to: "google_workspace.admin.managed_configuration",
                },
                {
                    from: "google_workspace.admin.MOBILE_APP_PACKAGE_ID",
                    to: "google_workspace.admin.application.package_id",
                },
                {
                    from: "google_workspace.admin.FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTION",
                    to: "google_workspace.admin.non_featured_services_selection",
                },
                {
                    from: "google_workspace.admin.FIELD_NAME",
                    to: "google_workspace.admin.field",
                },
                {
                    from: "google_workspace.admin.RESOURCE_IDENTIFIER",
                    to: "google_workspace.admin.resource.id",
                },
                {
                    from: "google_workspace.admin.USER_EMAIL",
                    to: "google_workspace.admin.user.email",
                },
                {
                    from: "google_workspace.admin.GATEWAY_NAME",
                    to: "google_workspace.admin.gateway.name",
                },
                {
                    from: "google_workspace.admin.APP_ID",
                    to: "google_workspace.admin.application.id",
                },
                {
                    from: "google_workspace.admin.ASP_ID",
                    to: "google_workspace.admin.application.asp_id",
                },
                {
                    from: "google_workspace.admin.CHROME_OS_SESSION_TYPE",
                    to: "google_workspace.admin.chrome_os.session_type",
                },
                {
                    from: "google_workspace.admin.DEVICE_NEW_STATE",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.DEVICE_PREVIOUS_STATE",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.DEVICE_SERIAL_NUMBER",
                    to: "google_workspace.admin.device.serial_number",
                },
                {
                    from: "google_workspace.admin.DEVICE_ID",
                    to: "google_workspace.admin.device.id",
                },
                {
                    from: "google_workspace.admin.DEVICE_TYPE",
                    to: "google_workspace.admin.device.type",
                },
                {
                    from: "google_workspace.admin.PRINT_SERVER_NAME",
                    to: "google_workspace.admin.print_server.name",
                },
                {
                    from: "google_workspace.admin.PRINTER_NAME",
                    to: "google_workspace.admin.printer.name",
                },
                {
                    from: "google_workspace.admin.DEVICE_COMMAND_DETAILS",
                    to: "google_workspace.admin.device.command_details",
                },
                {
                    from: "google_workspace.admin.DEVICE_NEW_ORG_UNIT",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.DEVICE_PREVIOUS_ORG_UNIT",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.ROLE_NAME",
                    to: "google_workspace.admin.role.name",
                },
                {
                    from: "google_workspace.admin.ROLE_ID",
                    to: "google_workspace.admin.role.id",
                },
                {
                    from: "google_workspace.admin.PRIVILEGE_NAME",
                    to: "google_workspace.admin.privilege.name",
                },
                {
                    from: "google_workspace.admin.SITE_LOCATION",
                    to: "url.path",
                },
                {
                    from: "google_workspace.admin.WEB_ADDRESS",
                    to: "url.full",
                },
                {
                    from: "google_workspace.admin.SITE_NAME",
                    to: "google_workspace.admin.url.name",
                },
                {
                    from: "google_workspace.admin.SERVICE_NAME",
                    to: "google_workspace.admin.service.name",
                },
                {
                    from: "google_workspace.admin.PRODUCT_NAME",
                    to: "google_workspace.admin.product.name",
                },
                {
                    from: "google_workspace.admin.SKU_NAME",
                    to: "google_workspace.admin.product.sku",
                },
                {
                    from: "google_workspace.admin.GROUP_MEMBER_BULK_UPLOAD_FAILED_NUMBER",
                    to: "google_workspace.admin.bulk_upload.failed",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.GROUP_MEMBER_BULK_UPLOAD_TOTAL_NUMBER",
                    to: "google_workspace.admin.bulk_upload.total",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.BULK_UPLOAD_FAIL_USERS_NUMBER",
                    to: "google_workspace.admin.bulk_upload.failed",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.BULK_UPLOAD_TOTAL_USERS_NUMBER",
                    to: "google_workspace.admin.bulk_upload.total",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.EMAIL_LOG_SEARCH_MSG_ID",
                    to: "google_workspace.admin.email.log_search_filter.message_id",
                },
                {
                    from: "google_workspace.admin.EMAIL_LOG_SEARCH_RECIPIENT",
                    to: "google_workspace.admin.email.log_search_filter.recipient.value",
                },
                {
                    from: "google_workspace.admin.EMAIL_LOG_SEARCH_SENDER",
                    to: "google_workspace.admin.email.log_search_filter.sender.value",
                },
                {
                    from: "google_workspace.admin.EMAIL_LOG_SEARCH_SMTP_RECIPIENT_IP",
                    to: "google_workspace.admin.email.log_search_filter.recipient.ip",
                    type: "ip",
                },
                {
                    from: "google_workspace.admin.EMAIL_LOG_SEARCH_SMTP_SENDER_IP",
                    to: "google_workspace.admin.email.log_search_filter.sender.ip",
                    type: "ip",
                },
                {
                    from: "google_workspace.admin.QUARANTINE_NAME",
                    to: "google_workspace.admin.email.quarantine_name",
                },
                {
                    from: "google_workspace.admin.CHROME_LICENSES_ENABLED",
                    to: "google_workspace.admin.chrome_licenses.enabled",
                },
                {
                    from: "google_workspace.admin.CHROME_LICENSES_ALLOWED",
                    to: "google_workspace.admin.chrome_licenses.allowed",
                },
                {
                    from: "google_workspace.admin.FULL_ORG_UNIT_PATH",
                    to: "google_workspace.admin.org_unit.full",
                },
                {
                    from: "google_workspace.admin.OAUTH2_SERVICE_NAME",
                    to: "google_workspace.admin.oauth2.service.name",
                },
                {
                    from: "google_workspace.admin.OAUTH2_APP_ID",
                    to: "google_workspace.admin.oauth2.application.id",
                },
                {
                    from: "google_workspace.admin.OAUTH2_APP_NAME",
                    to: "google_workspace.admin.oauth2.application.name",
                },
                {
                    from: "google_workspace.admin.OAUTH2_APP_TYPE",
                    to: "google_workspace.admin.oauth2.application.type",
                },
                {
                    from: "google_workspace.admin.ALLOWED_TWO_STEP_VERIFICATION_METHOD",
                    to: "google_workspace.admin.verification_method",
                },
                {
                    from: "google_workspace.admin.DOMAIN_VERIFICATION_METHOD",
                    to: "google_workspace.admin.verification_method",
                },
                {
                    from: "google_workspace.admin.CAA_ASSIGNMENTS_NEW",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.CAA_ASSIGNMENTS_OLD",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.REAUTH_SETTING_NEW",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.REAUTH_SETTING_OLD",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.ALERT_NAME",
                    to: "google_workspace.admin.alert.name",
                },
                {
                    from: "google_workspace.admin.API_CLIENT_NAME",
                    to: "google_workspace.admin.api.client.name",
                },
                {
                    from: "google_workspace.admin.API_SCOPES",
                    to: "google_workspace.admin.api.scopes",
                },
                {
                    from: "google_workspace.admin.PLAY_FOR_WORK_TOKEN_ID",
                    to: "google_workspace.admin.mdm.token",
                },
                {
                    from: "google_workspace.admin.PLAY_FOR_WORK_MDM_VENDOR_NAME",
                    to: "google_workspace.admin.mdm.vendor",
                },
                {
                    from: "google_workspace.admin.INFO_TYPE",
                    to: "google_workspace.admin.info_type",
                },
                {
                    from: "google_workspace.admin.RULE_NAME",
                    to: "google_workspace.admin.rule.name",
                },
                {
                    from: "google_workspace.admin.USER_CUSTOM_FIELD",
                    to: "google_workspace.admin.setting.name",
                },
                {
                    from: "google_workspace.admin.EMAIL_MONITOR_DEST_EMAIL",
                    to: "google_workspace.admin.email_monitor.dest_email",
                },
                {
                    from: "google_workspace.admin.EMAIL_MONITOR_LEVEL_CHAT",
                    to: "google_workspace.admin.email_monitor.level.chat",
                },
                {
                    from: "google_workspace.admin.EMAIL_MONITOR_LEVEL_DRAFT_EMAIL",
                    to: "google_workspace.admin.email_monitor.level.draft",
                },
                {
                    from: "google_workspace.admin.EMAIL_MONITOR_LEVEL_INCOMING_EMAIL",
                    to: "google_workspace.admin.email_monitor.level.incoming",
                },
                {
                    from: "google_workspace.admin.EMAIL_MONITOR_LEVEL_OUTGOING_EMAIL",
                    to: "google_workspace.admin.email_monitor.level.outgoing",
                },
                {
                    from: "google_workspace.admin.EMAIL_EXPORT_INCLUDE_DELETED",
                    to: "google_workspace.admin.email_dump.include_deleted",
                },
                {
                    from: "google_workspace.admin.EMAIL_EXPORT_PACKAGE_CONTENT",
                    to: "google_workspace.admin.email_dump.package_content",
                },
                {
                    from: "google_workspace.admin.SEARCH_QUERY_FOR_DUMP",
                    to: "google_workspace.admin.email_dump.query",
                },
                {
                    from: "google_workspace.admin.DESTINATION_USER_EMAIL",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.REQUEST_ID",
                    to: "google_workspace.admin.request.id",
                },
                {
                    from: "google_workspace.admin.GMAIL_RESET_REASON",
                    to: "message",
                },
                {
                    from: "google_workspace.admin.USER_NICKNAME",
                    to: "google_workspace.admin.user.nickname",
                },
                {
                    from: "google_workspace.admin.ACTION_ID",
                    to: "google_workspace.admin.mobile.action.id",
                },
                {
                    from: "google_workspace.admin.ACTION_TYPE",
                    to: "google_workspace.admin.mobile.action.type",
                },
                {
                    from: "google_workspace.admin.MOBILE_CERTIFICATE_COMMON_NAME",
                    to: "google_workspace.admin.mobile.certificate.name",
                },
                {
                    from: "google_workspace.admin.NUMBER_OF_COMPANY_OWNED_DEVICES",
                    to: "google_workspace.admin.mobile.company_owned_devices",
                    type: "long",
                },
                {
                    from: "google_workspace.admin.COMPANY_DEVICE_ID",
                    to: "google_workspace.admin.device.id",
                },
                {
                    from: "google_workspace.admin.DISTRIBUTION_ENTITY_NAME",
                    to: "google_workspace.admin.distribution.entity.name",
                },
                {
                    from: "google_workspace.admin.DISTRIBUTION_ENTITY_TYPE",
                    to: "google_workspace.admin.distribution.entity.type",
                },
                {
                    from: "google_workspace.admin.MOBILE_APP_PACKAGE_ID",
                    to: "google_workspace.admin.application.package_id",
                },
                {
                    from: "google_workspace.admin.NEW_PERMISSION_GRANT_STATE",
                    to: "google_workspace.admin.new_value",
                },
                {
                    from: "google_workspace.admin.OLD_PERMISSION_GRANT_STATE",
                    to: "google_workspace.admin.old_value",
                },
                {
                    from: "google_workspace.admin.PERMISSION_GROUP_NAME",
                    to: "google_workspace.admin.setting.name",
                },
                {
                    from: "google_workspace.admin.MOBILE_WIRELESS_NETWORK_NAME",
                    to: "network.name",
                },
            ],
            mode: "rename",
            ignore_missing: true,
            fail_on_error: false,
        })
        .Add(parseDate(
            "google_workspace.admin.EMAIL_LOG_SEARCH_END_DATE",
            "google_workspace.admin.email.log_search_filter.end_date"
        ))
        .Add(parseDate(
            "google_workspace.admin.EMAIL_LOG_SEARCH_START_DATE",
            "google_workspace.admin.email.log_search_filter.start_date"
        ))
        .Add(parseDate(
            "google_workspace.admin.BIRTHDATE",
            "google_workspace.admin.user.birthdate"
        ))
        .Add(parseDate(
            "google_workspace.admin.BEGIN_DATE_TIME",
            "event.start"
        ))
        .Add(parseDate(
            "google_workspace.admin.START_DATE",
            "event.start"
        ))
        .Add(parseDate(
            "google_workspace.admin.END_DATE",
            "event.end"
        ))
        .Add(parseDate(
            "google_workspace.admin.END_DATE_TIME",
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
