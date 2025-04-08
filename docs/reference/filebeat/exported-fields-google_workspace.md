---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-google_workspace.html
---

# google_workspace fields [exported-fields-google_workspace]

Google Workspace Module


## google_workspace [_google_workspace]

Google Workspace specific fields. More information about specific fields can be found at [https://developers.google.com/admin-sdk/reports/v1/reference/activities/list](https://developers.google.com/admin-sdk/reports/v1/reference/activities/list)

**`google_workspace.actor.type`**
:   The type of actor. Values can be: **USER**: Another user in the same domain. **EXTERNAL_USER**: A user outside the domain. **KEY**: A non-human actor.

type: keyword


**`google_workspace.actor.key`**
:   Only present when `actor.type` is `KEY`. Can be the `consumer_key` of the requestor for OAuth 2LO API requests or an identifier for robot accounts.

type: keyword


**`google_workspace.event.type`**
:   The type of Google Workspace event, mapped from `items[].events[].type` in the original payload. Each fileset can have a different set of values for it, more details can be found at [https://developers.google.com/admin-sdk/reports/v1/reference/activities/list](https://developers.google.com/admin-sdk/reports/v1/reference/activities/list)

type: keyword

example: audit#activity


**`google_workspace.kind`**
:   The type of API resource, mapped from `kind` in the original payload. More details can be found at [https://developers.google.com/admin-sdk/reports/v1/reference/activities/list](https://developers.google.com/admin-sdk/reports/v1/reference/activities/list)

type: keyword

example: audit#activity


**`google_workspace.organization.domain`**
:   The domain that is affected by the report’s event.

type: keyword


**`google_workspace.admin.application.edition`**
:   The Google Workspace edition.

type: keyword


**`google_workspace.admin.application.name`**
:   The application’s name.

type: keyword


**`google_workspace.admin.application.enabled`**
:   The enabled application.

type: keyword


**`google_workspace.admin.application.licences_order_number`**
:   Order number used to redeem licenses.

type: keyword


**`google_workspace.admin.application.licences_purchased`**
:   Number of licences purchased.

type: keyword


**`google_workspace.admin.application.id`**
:   The application ID.

type: keyword


**`google_workspace.admin.application.asp_id`**
:   The application specific password ID.

type: keyword


**`google_workspace.admin.application.package_id`**
:   The mobile application package ID.

type: keyword


**`google_workspace.admin.group.email`**
:   The group’s primary email address.

type: keyword


**`google_workspace.admin.new_value`**
:   The new value for the setting.

type: keyword


**`google_workspace.admin.old_value`**
:   The old value for the setting.

type: keyword


**`google_workspace.admin.org_unit.name`**
:   The organizational unit name.

type: keyword


**`google_workspace.admin.org_unit.full`**
:   The org unit full path including the root org unit name.

type: keyword


**`google_workspace.admin.setting.name`**
:   The setting name.

type: keyword


**`google_workspace.admin.user_defined_setting.name`**
:   The name of the user-defined setting.

type: keyword


**`google_workspace.admin.setting.description`**
:   The setting name.

type: keyword


**`google_workspace.admin.group.priorities`**
:   Group priorities.

type: keyword


**`google_workspace.admin.domain.alias`**
:   The domain alias.

type: keyword


**`google_workspace.admin.domain.name`**
:   The primary domain name.

type: keyword


**`google_workspace.admin.domain.secondary_name`**
:   The secondary domain name.

type: keyword


**`google_workspace.admin.managed_configuration`**
:   The name of the managed configuration.

type: keyword


**`google_workspace.admin.non_featured_services_selection`**
:   Non-featured services selection. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-application-settings#FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTED](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-application-settings#FLASHLIGHT_EDU_NON_FEATURED_SERVICES_SELECTED)

type: keyword


**`google_workspace.admin.field`**
:   The name of the field.

type: keyword


**`google_workspace.admin.resource.id`**
:   The name of the resource identifier.

type: keyword


**`google_workspace.admin.user.email`**
:   The user’s primary email address.

type: keyword


**`google_workspace.admin.user.nickname`**
:   The user’s nickname.

type: keyword


**`google_workspace.admin.user.birthdate`**
:   The user’s birth date.

type: date


**`google_workspace.admin.gateway.name`**
:   Gateway name. Present on some chat settings.

type: keyword


**`google_workspace.admin.chrome_os.session_type`**
:   Chrome OS session type.

type: keyword


**`google_workspace.admin.device.serial_number`**
:   Device serial number.

type: keyword


**`google_workspace.admin.device.id`**
:   type: keyword


**`google_workspace.admin.device.type`**
:   Device type.

type: keyword


**`google_workspace.admin.print_server.name`**
:   The name of the print server.

type: keyword


**`google_workspace.admin.printer.name`**
:   The name of the printer.

type: keyword


**`google_workspace.admin.device.command_details`**
:   Command details.

type: keyword


**`google_workspace.admin.role.id`**
:   Unique identifier for this role privilege.

type: keyword


**`google_workspace.admin.role.name`**
:   The role name. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-delegated-admin-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-delegated-admin-settings)

type: keyword


**`google_workspace.admin.privilege.name`**
:   Privilege name.

type: keyword


**`google_workspace.admin.service.name`**
:   The service name.

type: keyword


**`google_workspace.admin.url.name`**
:   The website name.

type: keyword


**`google_workspace.admin.product.name`**
:   The product name.

type: keyword


**`google_workspace.admin.product.sku`**
:   The product SKU.

type: keyword


**`google_workspace.admin.bulk_upload.failed`**
:   Number of failed records in bulk upload operation.

type: long


**`google_workspace.admin.bulk_upload.total`**
:   Number of total records in bulk upload operation.

type: long


**`google_workspace.admin.group.allowed_list`**
:   Names of allow-listed groups.

type: keyword


**`google_workspace.admin.email.quarantine_name`**
:   The name of the quarantine.

type: keyword


**`google_workspace.admin.email.log_search_filter.message_id`**
:   The log search filter’s email message ID.

type: keyword


**`google_workspace.admin.email.log_search_filter.start_date`**
:   The log search filter’s start date.

type: date


**`google_workspace.admin.email.log_search_filter.end_date`**
:   The log search filter’s ending date.

type: date


**`google_workspace.admin.email.log_search_filter.recipient.value`**
:   The log search filter’s email recipient.

type: keyword


**`google_workspace.admin.email.log_search_filter.sender.value`**
:   The log search filter’s email sender.

type: keyword


**`google_workspace.admin.email.log_search_filter.recipient.ip`**
:   The log search filter’s email recipient’s IP address.

type: ip


**`google_workspace.admin.email.log_search_filter.sender.ip`**
:   The log search filter’s email sender’s IP address.

type: ip


**`google_workspace.admin.chrome_licenses.enabled`**
:   Licences enabled. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-org-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-org-settings)

type: keyword


**`google_workspace.admin.chrome_licenses.allowed`**
:   Licences enabled. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-org-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-org-settings)

type: keyword


**`google_workspace.admin.oauth2.service.name`**
:   OAuth2 service name. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings)

type: keyword


**`google_workspace.admin.oauth2.application.id`**
:   OAuth2 application ID.

type: keyword


**`google_workspace.admin.oauth2.application.name`**
:   OAuth2 application name.

type: keyword


**`google_workspace.admin.oauth2.application.type`**
:   OAuth2 application type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings)

type: keyword


**`google_workspace.admin.verification_method`**
:   Related verification method. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-security-settings) and [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-domain-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-domain-settings)

type: keyword


**`google_workspace.admin.alert.name`**
:   The alert name.

type: keyword


**`google_workspace.admin.rule.name`**
:   The rule name.

type: keyword


**`google_workspace.admin.api.client.name`**
:   The API client name.

type: keyword


**`google_workspace.admin.api.scopes`**
:   The API scopes.

type: keyword


**`google_workspace.admin.mdm.token`**
:   The MDM vendor enrollment token.

type: keyword


**`google_workspace.admin.mdm.vendor`**
:   The MDM vendor’s name.

type: keyword


**`google_workspace.admin.info_type`**
:   This will be used to state what kind of information was changed. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-domain-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-domain-settings)

type: keyword


**`google_workspace.admin.email_monitor.dest_email`**
:   The destination address of the email monitor.

type: keyword


**`google_workspace.admin.email_monitor.level.chat`**
:   The chat email monitor level.

type: keyword


**`google_workspace.admin.email_monitor.level.draft`**
:   The draft email monitor level.

type: keyword


**`google_workspace.admin.email_monitor.level.incoming`**
:   The incoming email monitor level.

type: keyword


**`google_workspace.admin.email_monitor.level.outgoing`**
:   The outgoing email monitor level.

type: keyword


**`google_workspace.admin.email_dump.include_deleted`**
:   Indicates if deleted emails are included in the export.

type: boolean


**`google_workspace.admin.email_dump.package_content`**
:   The contents of the mailbox package.

type: keyword


**`google_workspace.admin.email_dump.query`**
:   The search query used for the dump.

type: keyword


**`google_workspace.admin.request.id`**
:   The request ID.

type: keyword


**`google_workspace.admin.mobile.action.id`**
:   The mobile device action’s ID.

type: keyword


**`google_workspace.admin.mobile.action.type`**
:   The mobile device action’s type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings)

type: keyword


**`google_workspace.admin.mobile.certificate.name`**
:   The mobile certificate common name.

type: keyword


**`google_workspace.admin.mobile.company_owned_devices`**
:   The number of devices a company owns.

type: long


**`google_workspace.admin.distribution.entity.name`**
:   The distribution entity value, which can be a group name or an org-unit name. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings)

type: keyword


**`google_workspace.admin.distribution.entity.type`**
:   The distribution entity type, which can be a group or an org-unit. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/admin-mobile-settings)

type: keyword


**`google_workspace.drive.billable`**
:   Whether this activity is billable.

type: boolean


**`google_workspace.drive.source_folder_id`**
:   type: keyword


**`google_workspace.drive.source_folder_title`**
:   type: keyword


**`google_workspace.drive.destination_folder_id`**
:   type: keyword


**`google_workspace.drive.destination_folder_title`**
:   type: keyword


**`google_workspace.drive.file.id`**
:   type: keyword


**`google_workspace.drive.file.type`**
:   Document Drive type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.originating_app_id`**
:   The Google Cloud Project ID of the application that performed the action.

type: keyword


**`google_workspace.drive.file.owner.email`**
:   type: keyword


**`google_workspace.drive.file.owner.is_shared_drive`**
:   Boolean flag denoting whether owner is a shared drive.

type: boolean


**`google_workspace.drive.primary_event`**
:   Whether this is a primary event. A single user action in Drive may generate several events.

type: boolean


**`google_workspace.drive.shared_drive_id`**
:   The unique identifier of the Team Drive. Only populated for for events relating to a Team Drive or item contained inside a Team Drive.

type: keyword


**`google_workspace.drive.visibility`**
:   Visibility of target file. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.new_value`**
:   When a setting or property of the file changes, the new value for it will appear here.

type: keyword


**`google_workspace.drive.old_value`**
:   When a setting or property of the file changes, the old value for it will appear here.

type: keyword


**`google_workspace.drive.sheets_import_range_recipient_doc`**
:   Doc ID of the recipient of a sheets import range.

type: keyword


**`google_workspace.drive.old_visibility`**
:   When visibility changes, this holds the old value.

type: keyword


**`google_workspace.drive.visibility_change`**
:   When visibility changes, this holds the new overall visibility of the file.

type: keyword


**`google_workspace.drive.target_domain`**
:   The domain for which the acccess scope was changed. This can also be the alias all to indicate the access scope was changed for all domains that have visibility for this document.

type: keyword


**`google_workspace.drive.added_role`**
:   Added membership role of a user/group in a Team Drive. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.membership_change_type`**
:   Type of change in Team Drive membership of a user/group. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.shared_drive_settings_change_type`**
:   Type of change in Team Drive settings. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.removed_role`**
:   Removed membership role of a user/group in a Team Drive. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/drive)

type: keyword


**`google_workspace.drive.target`**
:   Target user or group.

type: keyword


**`google_workspace.groups.acl_permission`**
:   Group permission setting updated. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.email`**
:   Group email.

type: keyword


**`google_workspace.groups.member.email`**
:   Member email.

type: keyword


**`google_workspace.groups.member.role`**
:   Member role. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.setting`**
:   Group setting updated. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.new_value`**
:   New value(s) of the group setting. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.old_value`**
:   Old value(s) of the group setting. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.value`**
:   Value of the group setting. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/groups)

type: keyword


**`google_workspace.groups.message.id`**
:   SMTP message Id of an email message. Present for moderation events.

type: keyword


**`google_workspace.groups.message.moderation_action`**
:   Message moderation action. Possible values are `approved` and `rejected`.

type: keyword


**`google_workspace.groups.status`**
:   A status describing the output of an operation. Possible values are `failed` and `succeeded`.

type: keyword


**`google_workspace.login.affected_email_address`**
:   type: keyword


**`google_workspace.login.challenge_method`**
:   Login challenge method. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login).

type: keyword


**`google_workspace.login.failure_type`**
:   Login failure type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login).

type: keyword


**`google_workspace.login.type`**
:   Login credentials type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/login).

type: keyword


**`google_workspace.login.is_second_factor`**
:   type: boolean


**`google_workspace.login.is_suspicious`**
:   type: boolean


**`google_workspace.saml.application_name`**
:   Saml SP application name.

type: keyword


**`google_workspace.saml.failure_type`**
:   Login failure type. For a list of possible values refer to [https://developers.google.com/admin-sdk/reports/v1/appendix/activity/saml](https://developers.google.com/admin-sdk/reports/v1/appendix/activity/saml).

type: keyword


**`google_workspace.saml.initiated_by`**
:   Requester of SAML authentication.

type: keyword


**`google_workspace.saml.orgunit_path`**
:   User orgunit.

type: keyword


**`google_workspace.saml.status_code`**
:   SAML status code.

type: keyword


**`google_workspace.saml.second_level_status_code`**
:   SAML second level status code.

type: keyword


