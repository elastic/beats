---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-zoom.html
---

# Zoom fields [exported-fields-zoom]

Module for handling incoming Zoom webhook requests


## zoom [_zoom]

Module for parsing Zoom API Webhooks.

**`zoom.master_account_id`**
:   Master Account related to a specific Sub Account

type: keyword


**`zoom.sub_account_id`**
:   Related Sub Account

type: keyword


**`zoom.operator_id`**
:   UserID that triggered the event

type: keyword


**`zoom.operator`**
:   Username/Email related to the user that triggered the event

type: keyword


**`zoom.account_id`**
:   Related accountID to the event

type: keyword


**`zoom.timestamp`**
:   Timestamp related to the event

type: date


**`zoom.creation_type`**
:   Creation type

type: keyword


**`zoom.account.owner_id`**
:   UserID of the user whose sub account was created/disassociated

type: keyword


**`zoom.account.email`**
:   Email related to the user the action was performed on

type: keyword


**`zoom.account.owner_email`**
:   Email of the user whose sub account was created/disassociated

type: keyword


**`zoom.account.account_name`**
:   When an account name is updated, this is the new value set

type: keyword


**`zoom.account.account_alias`**
:   When an account alias is updated, this is the new value set

type: keyword


**`zoom.account.account_support_name`**
:   When an account support_name is updated, this is the new value set

type: keyword


**`zoom.account.account_support_email`**
:   When an account support_email is updated, this is the new value set

type: keyword


**`zoom.chat_channel.name`**
:   The name of the channel that has been added/modified/deleted

type: keyword


**`zoom.chat_channel.id`**
:   The ID of the channel that has been added/modified/deleted

type: keyword


**`zoom.chat_channel.type`**
:   Type of channel related to the event. Can be 1(Invite-Only), 2(Private) or 3(Public)

type: keyword


**`zoom.chat_message.id`**
:   Unique ID of the related chat message

type: keyword


**`zoom.chat_message.type`**
:   Type of message, can be either "to_contact" or "to_channel"

type: keyword


**`zoom.chat_message.session_id`**
:   SessionID for the channel related to the message

type: keyword


**`zoom.chat_message.contact_email`**
:   Email address related to the user sending the message

type: keyword


**`zoom.chat_message.contact_id`**
:   UserID belonging to the user receiving a message

type: keyword


**`zoom.chat_message.channel_id`**
:   ChannelID related to the message

type: keyword


**`zoom.chat_message.channel_name`**
:   Channel name related to the message

type: keyword


**`zoom.chat_message.message`**
:   A string containing the full message that was sent

type: keyword


**`zoom.meeting.id`**
:   Unique ID of the related meeting

type: keyword


**`zoom.meeting.uuid`**
:   The UUID of the related meeting

type: keyword


**`zoom.meeting.host_id`**
:   The UserID of the configured meeting host

type: keyword


**`zoom.meeting.topic`**
:   Topic of the related meeting

type: keyword


**`zoom.meeting.type`**
:   Type of meeting created

type: keyword


**`zoom.meeting.start_time`**
:   Date and time the meeting started

type: date


**`zoom.meeting.timezone`**
:   Which timezone is used for the meeting timestamps

type: keyword


**`zoom.meeting.duration`**
:   The duration of a meeting in minutes

type: long


**`zoom.meeting.issues`**
:   When a user reports an issue with the meeting, for example: "Unstable audio quality"

type: keyword


**`zoom.meeting.password`**
:   Password related to the meeting

type: keyword


**`zoom.phone.id`**
:   Unique ID for the phone or conversation

type: keyword


**`zoom.phone.user_id`**
:   UserID for the phone owner related to a Call Log being completed

type: keyword


**`zoom.phone.download_url`**
:   Download URL for the voicemail

type: keyword


**`zoom.phone.ringing_start_time`**
:   The timestamp when a ringtone was established to the callee

type: date


**`zoom.phone.connected_start_time`**
:   The date and time when a ringtone was established to the callee

type: date


**`zoom.phone.answer_start_time`**
:   The date and time when the call was answered

type: date


**`zoom.phone.call_end_time`**
:   The date and time when the call ended

type: date


**`zoom.phone.call_id`**
:   Unique ID of the related call

type: keyword


**`zoom.phone.duration`**
:   Duration of a voicemail in minutes

type: long


**`zoom.phone.caller.id`**
:   UserID of the caller related to the voicemail/call

type: keyword


**`zoom.phone.caller.user_id`**
:   UserID of the person which initiated the call

type: keyword


**`zoom.phone.caller.number_type`**
:   The type of number, can be 1(Internal) or 2(External)

type: keyword


**`zoom.phone.caller.name`**
:   The name of the related callee

type: keyword


**`zoom.phone.caller.phone_number`**
:   Phone Number of the caller related to the call

type: keyword


**`zoom.phone.caller.extension_type`**
:   Extension type of the caller number, can be user, callQueue, autoReceptionist or shareLineGroup

type: keyword


**`zoom.phone.caller.extension_number`**
:   Extension number of the caller

type: keyword


**`zoom.phone.caller.timezone`**
:   Timezone of the caller

type: keyword


**`zoom.phone.caller.device_type`**
:   Device type used by the caller

type: keyword


**`zoom.phone.callee.id`**
:   UserID of the callee related to the voicemail/call

type: keyword


**`zoom.phone.callee.user_id`**
:   UserID of the related callee of a voicemail/call

type: keyword


**`zoom.phone.callee.name`**
:   The name of the related callee

type: keyword


**`zoom.phone.callee.number_type`**
:   The type of number, can be 1(Internal) or 2(External)

type: keyword


**`zoom.phone.callee.phone_number`**
:   Phone Number of the callee related to the call

type: keyword


**`zoom.phone.callee.extension_type`**
:   Extension type of the callee number, can be user, callQueue, autoReceptionist or shareLineGroup

type: keyword


**`zoom.phone.callee.extension_number`**
:   Extension number of the callee related to the call

type: keyword


**`zoom.phone.callee.timezone`**
:   Timezone of the callee related to the call

type: keyword


**`zoom.phone.callee.device_type`**
:   Device type used by the callee related to the call

type: keyword


**`zoom.phone.date_time`**
:   Date and time of the related phone event

type: date


**`zoom.recording.id`**
:   Unique ID of the related recording

type: keyword


**`zoom.recording.uuid`**
:   UUID of the related recording

type: keyword


**`zoom.recording.host_id`**
:   UserID of the host of the meeting that was recorded

type: keyword


**`zoom.recording.topic`**
:   Topic of the meeting related to the recording

type: keyword


**`zoom.recording.type`**
:   Type of recording, can be multiple type of values, please check Zoom documentation

type: keyword


**`zoom.recording.start_time`**
:   The date and time when the recording started

type: date


**`zoom.recording.timezone`**
:   The timezone used for the recording date

type: keyword


**`zoom.recording.duration`**
:   Duration of the recording in minutes

type: long


**`zoom.recording.share_url`**
:   The URL to access the recording

type: keyword


**`zoom.recording.total_size`**
:   Total size of the recording in bytes

type: long


**`zoom.recording.recording_count`**
:   Number of recording files related to the recording

type: long


**`zoom.recording.recording_file.recording_start`**
:   The date and time the recording started

type: date


**`zoom.recording.recording_file.recording_end`**
:   The date and time the recording finished

type: date


**`zoom.recording.host_email`**
:   Email address of the host related to the meeting that was recorded

type: keyword


**`zoom.user.id`**
:   UserID related to the user event

type: keyword


**`zoom.user.first_name`**
:   User first name related to the user event

type: keyword


**`zoom.user.last_name`**
:   User last name related to the user event

type: keyword


**`zoom.user.email`**
:   User email related to the user event

type: keyword


**`zoom.user.type`**
:   User type related to the user event

type: keyword


**`zoom.user.phone_number`**
:   User phone number related to the user event

type: keyword


**`zoom.user.phone_country`**
:   User country code related to the user event

type: keyword


**`zoom.user.company`**
:   User company related to the user event

type: keyword


**`zoom.user.pmi`**
:   User personal meeting ID related to the user event

type: keyword


**`zoom.user.use_pmi`**
:   If a user has PMI enabled

type: boolean


**`zoom.user.pic_url`**
:   Full URL to the profile picture used by the user

type: keyword


**`zoom.user.vanity_name`**
:   Name of the personal meeting room related to the user event

type: keyword


**`zoom.user.timezone`**
:   Timezone configured for the user

type: keyword


**`zoom.user.language`**
:   Language configured for the user

type: keyword


**`zoom.user.host_key`**
:   Host key set for the user

type: keyword


**`zoom.user.role`**
:   The configured role for the user

type: keyword


**`zoom.user.dept`**
:   The configured departement for the user

type: keyword


**`zoom.user.presence_status`**
:   Current presence status of user

type: keyword


**`zoom.user.personal_notes`**
:   Personal notes for the User

type: keyword


**`zoom.user.client_type`**
:   Type of client used by the user. Can be browser, mac, win, iphone or android

type: keyword


**`zoom.user.version`**
:   Version of the client used by the user

type: keyword


**`zoom.webinar.id`**
:   Unique ID for the related webinar

type: keyword


**`zoom.webinar.join_url`**
:   The URL configured to join the webinar

type: keyword


**`zoom.webinar.uuid`**
:   UUID for the related webinar

type: keyword


**`zoom.webinar.host_id`**
:   UserID for the configured host of the webinar

type: keyword


**`zoom.webinar.topic`**
:   Meeting topic of the related webinar

type: keyword


**`zoom.webinar.type`**
:   Type of webinar created. Can be either 5(Webinar), 6(Recurring webinar without fixed time) or 9(Recurring webinar with fixed time)

type: keyword


**`zoom.webinar.start_time`**
:   The date and time when the webinar started

type: date


**`zoom.webinar.timezone`**
:   Timezone used for the dates related to the webinar

type: keyword


**`zoom.webinar.duration`**
:   Duration of the webinar in minutes

type: long


**`zoom.webinar.agenda`**
:   The configured agenda of the webinar

type: keyword


**`zoom.webinar.password`**
:   Password configured to access the webinar

type: keyword


**`zoom.webinar.issues`**
:   Any reported issues about a webinar is reported in this field

type: keyword


**`zoom.zoomroom.id`**
:   Unique ID of the Zoom room

type: keyword


**`zoom.zoomroom.room_name`**
:   The configured name of the Zoom room

type: keyword


**`zoom.zoomroom.calendar_name`**
:   Calendar name of the Zoom room

type: keyword


**`zoom.zoomroom.calendar_id`**
:   Unique ID of the calendar used by the Zoom room

type: keyword


**`zoom.zoomroom.event_id`**
:   Unique ID of the calendar event associated with the Zoom Room

type: keyword


**`zoom.zoomroom.change_key`**
:   Key used by Microsoft products integration that represents a specific version of a calendar

type: keyword


**`zoom.zoomroom.resource_email`**
:   Email address associated with the calendar in use by the Zoom room

type: keyword


**`zoom.zoomroom.email`**
:   Email address associated with the Zoom room itself

type: keyword


**`zoom.zoomroom.issue`**
:   Any reported alerts or issues related to the Zoom room or its equipment

type: keyword


**`zoom.zoomroom.alert_type`**
:   An integer value representing the type of alert. The list of alert types can be found in the Zoom documentation

type: keyword


**`zoom.zoomroom.component`**
:   An integer value representing the type of equipment or component, The list of component types can be found in the Zoom documentation

type: keyword


**`zoom.zoomroom.alert_kind`**
:   An integer value showing if the Zoom room alert has been either 1(Triggered) or 2(Cleared)

type: keyword


**`zoom.registrant.id`**
:   Unique ID of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.status`**
:   Status of the specific user registration

type: keyword


**`zoom.registrant.email`**
:   Email of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.first_name`**
:   First name of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.last_name`**
:   Last name of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.address`**
:   Address of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.city`**
:   City of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.country`**
:   Country of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.zip`**
:   Zip code of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.state`**
:   State of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.phone`**
:   Phone number of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.industry`**
:   Related industry of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.org`**
:   Organization related to the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.job_title`**
:   Job title of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.purchasing_time_frame`**
:   Choosen purchase timeframe of the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.role_in_purchase_process`**
:   Choosen role in a purchase process related to the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.no_of_employees`**
:   Number of employees choosen by the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.comments`**
:   Comments left by the user registering to a meeting or webinar

type: keyword


**`zoom.registrant.join_url`**
:   The URL that the registrant can use to join the webinar

type: keyword


**`zoom.participant.id`**
:   Unique ID of the participant related to a meeting

type: keyword


**`zoom.participant.user_id`**
:   UserID of the participant related to a meeting

type: keyword


**`zoom.participant.user_name`**
:   Username of the participant related to a meeting

type: keyword


**`zoom.participant.join_time`**
:   The date and time a participant joined a meeting

type: date


**`zoom.participant.leave_time`**
:   The date and time a participant left a meeting

type: date


**`zoom.participant.sharing_details.link_source`**
:   Method of sharing with dropbox integration

type: keyword


**`zoom.participant.sharing_details.content`**
:   Type of content that was shared

type: keyword


**`zoom.participant.sharing_details.file_link`**
:   The file link that was shared

type: keyword


**`zoom.participant.sharing_details.date_time`**
:   Timestamp the sharing started

type: keyword


**`zoom.participant.sharing_details.source`**
:   The file source that was share

type: keyword


**`zoom.old_values`**
:   Includes the old values when updating a object like user, meeting, account or webinar

type: flattened


**`zoom.settings`**
:   The current active settings related to a object like user, meeting, account or webinar

type: flattened


