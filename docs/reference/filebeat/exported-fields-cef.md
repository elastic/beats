---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-cef.html
---

# Decode CEF processor fields fields [exported-fields-cef]

Common Event Format (CEF) data.


## cef [_cef]

By default the `decode_cef` processor writes all data from the CEF message to this `cef` object. It contains the CEF header fields and the extension data.

**`cef.version`**
:   Version of the CEF specification used by the message.

type: keyword


**`cef.device.vendor`**
:   Vendor of the device that produced the message.

type: keyword


**`cef.device.product`**
:   Product of the device that produced the message.

type: keyword


**`cef.device.version`**
:   Version of the product that produced the message.

type: keyword


**`cef.device.event_class_id`**
:   Unique identifier of the event type.

type: keyword


**`cef.severity`**
:   Importance of the event. The valid string values are Unknown, Low, Medium, High, and Very-High. The valid integer values are 0-3=Low, 4-6=Medium, 7- 8=High, and 9-10=Very-High.

type: keyword

example: Very-High


**`cef.name`**
:   Short description of the event.

type: keyword



## extensions [_extensions]

Collection of key-value pairs carried in the CEF extension field.

**`cef.extensions.agentAddress`**
:   The IP address of the ArcSight connector that processed the event.

type: ip


**`cef.extensions.agentDnsDomain`**
:   The DNS domain name of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentHostName`**
:   The hostname of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentId`**
:   The agent ID of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentMacAddress`**
:   The MAC address of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentNtDomain`**
:   None

type: keyword


**`cef.extensions.agentReceiptTime`**
:   The time at which information about the event was received by the ArcSight connector.

type: date


**`cef.extensions.agentTimeZone`**
:   The agent time zone of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentTranslatedAddress`**
:   None

type: ip


**`cef.extensions.agentTranslatedZoneExternalID`**
:   None

type: keyword


**`cef.extensions.agentTranslatedZoneURI`**
:   None

type: keyword


**`cef.extensions.agentType`**
:   The agent type of the ArcSight connector that processed the event

type: keyword


**`cef.extensions.agentVersion`**
:   The version of the ArcSight connector that processed the event.

type: keyword


**`cef.extensions.agentZoneExternalID`**
:   None

type: keyword


**`cef.extensions.agentZoneURI`**
:   None

type: keyword


**`cef.extensions.applicationProtocol`**
:   Application level protocol, example values are HTTP, HTTPS, SSHv2, Telnet, POP, IMPA, IMAPS, and so on.

type: keyword


**`cef.extensions.baseEventCount`**
:   A count associated with this event. How many times was this same event observed? Count can be omitted if it is 1.

type: long


**`cef.extensions.bytesIn`**
:   Number of bytes transferred inbound, relative to the source to destination relationship, meaning that data was flowing from source to destination.

type: long


**`cef.extensions.bytesOut`**
:   Number of bytes transferred outbound relative to the source to destination relationship. For example, the byte number of data flowing from the destination to the source.

type: long


**`cef.extensions.customerExternalID`**
:   None

type: keyword


**`cef.extensions.customerURI`**
:   None

type: keyword


**`cef.extensions.destinationAddress`**
:   Identifies the destination address that the event refers to in an IP network. The format is an IPv4 address.

type: ip


**`cef.extensions.destinationDnsDomain`**
:   The DNS domain part of the complete fully qualified domain name (FQDN).

type: keyword


**`cef.extensions.destinationGeoLatitude`**
:   The latitudinal value from which the destination’s IP address belongs.

type: double


**`cef.extensions.destinationGeoLongitude`**
:   The longitudinal value from which the destination’s IP address belongs.

type: double


**`cef.extensions.destinationHostName`**
:   Identifies the destination that an event refers to in an IP network. The format should be a fully qualified domain name (FQDN) associated with the destination node, when a node is available.

type: keyword


**`cef.extensions.destinationMacAddress`**
:   Six colon-seperated hexadecimal numbers.

type: keyword


**`cef.extensions.destinationNtDomain`**
:   The Windows domain name of the destination address.

type: keyword


**`cef.extensions.destinationPort`**
:   The valid port numbers are between 0 and 65535.

type: long


**`cef.extensions.destinationProcessId`**
:   Provides the ID of the destination process associated with the event. For example, if an event contains process ID 105, "105" is the process ID.

type: long


**`cef.extensions.destinationProcessName`**
:   The name of the event’s destination process.

type: keyword


**`cef.extensions.destinationServiceName`**
:   The service targeted by this event.

type: keyword


**`cef.extensions.destinationTranslatedAddress`**
:   Identifies the translated destination that the event refers to in an IP network.

type: ip


**`cef.extensions.destinationTranslatedPort`**
:   Port after it was translated; for example, a firewall. Valid port numbers are 0 to 65535.

type: long


**`cef.extensions.destinationTranslatedZoneExternalID`**
:   None

type: keyword


**`cef.extensions.destinationTranslatedZoneURI`**
:   The URI for the Translated Zone that the destination asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.destinationUserId`**
:   Identifies the destination user by ID. For example, in UNIX, the root user is generally associated with user ID 0.

type: keyword


**`cef.extensions.destinationUserName`**
:   Identifies the destination user by name. This is the user associated with the event’s destination. Email addresses are often mapped into the UserName fields. The recipient is a candidate to put into this field.

type: keyword


**`cef.extensions.destinationUserPrivileges`**
:   The typical values are "Administrator", "User", and "Guest". This identifies the destination user’s privileges. In UNIX, for example, activity executed on the root user would be identified with destinationUser Privileges of "Administrator".

type: keyword


**`cef.extensions.destinationZoneExternalID`**
:   None

type: keyword


**`cef.extensions.destinationZoneURI`**
:   The URI for the Zone that the destination asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.deviceAction`**
:   Action taken by the device.

type: keyword


**`cef.extensions.deviceAddress`**
:   Identifies the device address that an event refers to in an IP network.

type: ip


**`cef.extensions.deviceCustomFloatingPoint1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomFloatingPoint3Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomFloatingPoint4Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomDate1`**
:   One of two timestamp fields available to map fields that do not apply to any other in this dictionary.

type: date


**`cef.extensions.deviceCustomDate1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomDate2`**
:   One of two timestamp fields available to map fields that do not apply to any other in this dictionary.

type: date


**`cef.extensions.deviceCustomDate2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomFloatingPoint1`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary.

type: double


**`cef.extensions.deviceCustomFloatingPoint2`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary.

type: double


**`cef.extensions.deviceCustomFloatingPoint2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomFloatingPoint3`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary.

type: double


**`cef.extensions.deviceCustomFloatingPoint4`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary.

type: double


**`cef.extensions.deviceCustomIPv6Address1`**
:   One of four IPv6 address fields available to map fields that do not apply to any other in this dictionary.

type: ip


**`cef.extensions.deviceCustomIPv6Address1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomIPv6Address2`**
:   One of four IPv6 address fields available to map fields that do not apply to any other in this dictionary.

type: ip


**`cef.extensions.deviceCustomIPv6Address2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomIPv6Address3`**
:   One of four IPv6 address fields available to map fields that do not apply to any other in this dictionary.

type: ip


**`cef.extensions.deviceCustomIPv6Address3Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomIPv6Address4`**
:   One of four IPv6 address fields available to map fields that do not apply to any other in this dictionary.

type: ip


**`cef.extensions.deviceCustomIPv6Address4Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomNumber1`**
:   One of three number fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: long


**`cef.extensions.deviceCustomNumber1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomNumber2`**
:   One of three number fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: long


**`cef.extensions.deviceCustomNumber2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomNumber3`**
:   One of three number fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: long


**`cef.extensions.deviceCustomNumber3Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString1`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString2`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString3`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString3Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString4`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString4Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString5`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString5Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceCustomString6`**
:   One of six strings available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: keyword


**`cef.extensions.deviceCustomString6Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceDirection`**
:   Any information about what direction the observed communication has taken. The following values are supported - "0" for inbound or "1" for outbound.

type: long


**`cef.extensions.deviceDnsDomain`**
:   The DNS domain part of the complete fully qualified domain name (FQDN).

type: keyword


**`cef.extensions.deviceEventCategory`**
:   Represents the category assigned by the originating device. Devices often use their own categorization schema to classify event. Example "/Monitor/Disk/Read".

type: keyword


**`cef.extensions.deviceExternalId`**
:   A name that uniquely identifies the device generating this event.

type: keyword


**`cef.extensions.deviceFacility`**
:   The facility generating this event. For example, Syslog has an explicit facility associated with every event.

type: keyword


**`cef.extensions.deviceFlexNumber1`**
:   One of two alternative number fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: long


**`cef.extensions.deviceFlexNumber1Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceFlexNumber2`**
:   One of two alternative number fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible.

type: long


**`cef.extensions.deviceFlexNumber2Label`**
:   All custom fields have a corresponding label field. Each of these fields is a string and describes the purpose of the custom field.

type: keyword


**`cef.extensions.deviceHostName`**
:   The format should be a fully qualified domain name (FQDN) associated with the device node, when a node is available.

type: keyword


**`cef.extensions.deviceInboundInterface`**
:   Interface on which the packet or data entered the device.

type: keyword


**`cef.extensions.deviceMacAddress`**
:   Six colon-separated hexadecimal numbers.

type: keyword


**`cef.extensions.deviceNtDomain`**
:   The Windows domain name of the device address.

type: keyword


**`cef.extensions.deviceOutboundInterface`**
:   Interface on which the packet or data left the device.

type: keyword


**`cef.extensions.devicePayloadId`**
:   Unique identifier for the payload associated with the event.

type: keyword


**`cef.extensions.deviceProcessId`**
:   Provides the ID of the process on the device generating the event.

type: long


**`cef.extensions.deviceProcessName`**
:   Process name associated with the event. An example might be the process generating the syslog entry in UNIX.

type: keyword


**`cef.extensions.deviceReceiptTime`**
:   The time at which the event related to the activity was received. The format is MMM dd yyyy HH:mm:ss or milliseconds since epoch (Jan 1st 1970)

type: date


**`cef.extensions.deviceTimeZone`**
:   The time zone for the device generating the event.

type: keyword


**`cef.extensions.deviceTranslatedAddress`**
:   Identifies the translated device address that the event refers to in an IP network.

type: ip


**`cef.extensions.deviceTranslatedZoneExternalID`**
:   None

type: keyword


**`cef.extensions.deviceTranslatedZoneURI`**
:   The URI for the Translated Zone that the device asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.deviceZoneExternalID`**
:   None

type: keyword


**`cef.extensions.deviceZoneURI`**
:   Thee URI for the Zone that the device asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.endTime`**
:   The time at which the activity related to the event ended. The format is MMM dd yyyy HH:mm:ss or milliseconds since epoch (Jan 1st1970). An example would be reporting the end of a session.

type: date


**`cef.extensions.eventId`**
:   This is a unique ID that ArcSight assigns to each event.

type: long


**`cef.extensions.eventOutcome`**
:   Displays the outcome, usually as *success* or *failure*.

type: keyword


**`cef.extensions.externalId`**
:   The ID used by an originating device. They are usually increasing numbers, associated with events.

type: keyword


**`cef.extensions.fileCreateTime`**
:   Time when the file was created.

type: date


**`cef.extensions.fileHash`**
:   Hash of a file.

type: keyword


**`cef.extensions.fileId`**
:   An ID associated with a file could be the inode.

type: keyword


**`cef.extensions.fileModificationTime`**
:   Time when the file was last modified.

type: date


**`cef.extensions.filename`**
:   Name of the file only (without its path).

type: keyword


**`cef.extensions.filePath`**
:   Full path to the file, including file name itself.

type: keyword


**`cef.extensions.filePermission`**
:   Permissions of the file.

type: keyword


**`cef.extensions.fileSize`**
:   Size of the file.

type: long


**`cef.extensions.fileType`**
:   Type of file (pipe, socket, etc.)

type: keyword


**`cef.extensions.flexDate1`**
:   A timestamp field available to map a timestamp that does not apply to any other defined timestamp field in this dictionary. Use all flex fields sparingly and seek a more specific, dictionary supplied field when possible. These fields are typically reserved for customer use and should not be set by vendors unless necessary.

type: date


**`cef.extensions.flexDate1Label`**
:   The label field is a string and describes the purpose of the flex field.

type: keyword


**`cef.extensions.flexString1`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible. These fields are typically reserved for customer use and should not be set by vendors unless necessary.

type: keyword


**`cef.extensions.flexString2`**
:   One of four floating point fields available to map fields that do not apply to any other in this dictionary. Use sparingly and seek a more specific, dictionary supplied field when possible. These fields are typically reserved for customer use and should not be set by vendors unless necessary.

type: keyword


**`cef.extensions.flexString1Label`**
:   The label field is a string and describes the purpose of the flex field.

type: keyword


**`cef.extensions.flexString2Label`**
:   The label field is a string and describes the purpose of the flex field.

type: keyword


**`cef.extensions.message`**
:   An arbitrary message giving more details about the event. Multi-line entries can be produced by using \n as the new line separator.

type: keyword


**`cef.extensions.oldFileCreateTime`**
:   Time when old file was created.

type: date


**`cef.extensions.oldFileHash`**
:   Hash of the old file.

type: keyword


**`cef.extensions.oldFileId`**
:   An ID associated with the old file could be the inode.

type: keyword


**`cef.extensions.oldFileModificationTime`**
:   Time when old file was last modified.

type: date


**`cef.extensions.oldFileName`**
:   Name of the old file.

type: keyword


**`cef.extensions.oldFilePath`**
:   Full path to the old file, including the file name itself.

type: keyword


**`cef.extensions.oldFilePermission`**
:   Permissions of the old file.

type: keyword


**`cef.extensions.oldFileSize`**
:   Size of the old file.

type: long


**`cef.extensions.oldFileType`**
:   Type of the old file (pipe, socket, etc.)

type: keyword


**`cef.extensions.rawEvent`**
:   None

type: keyword


**`cef.extensions.Reason`**
:   The reason an audit event was generated. For example "bad password" or "unknown user". This could also be an error or return code. Example "0x1234".

type: keyword


**`cef.extensions.requestClientApplication`**
:   The User-Agent associated with the request.

type: keyword


**`cef.extensions.requestContext`**
:   Description of the content from which the request originated (for example, HTTP Referrer)

type: keyword


**`cef.extensions.requestCookies`**
:   Cookies associated with the request.

type: keyword


**`cef.extensions.requestMethod`**
:   The HTTP method used to access a URL.

type: keyword


**`cef.extensions.requestUrl`**
:   In the case of an HTTP request, this field contains the URL accessed. The URL should contain the protocol as well.

type: keyword


**`cef.extensions.sourceAddress`**
:   Identifies the source that an event refers to in an IP network.

type: ip


**`cef.extensions.sourceDnsDomain`**
:   The DNS domain part of the complete fully qualified domain name (FQDN).

type: keyword


**`cef.extensions.sourceGeoLatitude`**
:   None

type: double


**`cef.extensions.sourceGeoLongitude`**
:   None

type: double


**`cef.extensions.sourceHostName`**
:   Identifies the source that an event refers to in an IP network. The format should be a fully qualified domain name (FQDN) associated with the source node, when a mode is available. Examples: *host* or *host.domain.com*.

type: keyword


**`cef.extensions.sourceMacAddress`**
:   Six colon-separated hexadecimal numbers.

type: keyword

example: 00:0d:60:af:1b:61


**`cef.extensions.sourceNtDomain`**
:   The Windows domain name for the source address.

type: keyword


**`cef.extensions.sourcePort`**
:   The valid port numbers are 0 to 65535.

type: long


**`cef.extensions.sourceProcessId`**
:   The ID of the source process associated with the event.

type: long


**`cef.extensions.sourceProcessName`**
:   The name of the event’s source process.

type: keyword


**`cef.extensions.sourceServiceName`**
:   The service that is responsible for generating this event.

type: keyword


**`cef.extensions.sourceTranslatedAddress`**
:   Identifies the translated source that the event refers to in an IP network.

type: ip


**`cef.extensions.sourceTranslatedPort`**
:   A port number after being translated by, for example, a firewall. Valid port numbers are 0 to 65535.

type: long


**`cef.extensions.sourceTranslatedZoneExternalID`**
:   None

type: keyword


**`cef.extensions.sourceTranslatedZoneURI`**
:   The URI for the Translated Zone that the destination asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.sourceUserId`**
:   Identifies the source user by ID. This is the user associated with the source of the event. For example, in UNIX, the root user is generally associated with user ID 0.

type: keyword


**`cef.extensions.sourceUserName`**
:   Identifies the source user by name. Email addresses are also mapped into the UserName fields. The sender is a candidate to put into this field.

type: keyword


**`cef.extensions.sourceUserPrivileges`**
:   The typical values are "Administrator", "User", and "Guest". It identifies the source user’s privileges. In UNIX, for example, activity executed by the root user would be identified with "Administrator".

type: keyword


**`cef.extensions.sourceZoneExternalID`**
:   None

type: keyword


**`cef.extensions.sourceZoneURI`**
:   The URI for the Zone that the source asset has been assigned to in ArcSight.

type: keyword


**`cef.extensions.startTime`**
:   The time when the activity the event referred to started. The format is MMM dd yyyy HH:mm:ss or milliseconds since epoch (Jan 1st 1970)

type: date


**`cef.extensions.transportProtocol`**
:   Identifies the Layer-4 protocol used. The possible values are protocols such as TCP or UDP.

type: keyword


**`cef.extensions.type`**
:   0 means base event, 1 means aggregated, 2 means correlation, and 3 means action. This field can be omitted for base events (type 0).

type: long


**`cef.extensions.categoryDeviceType`**
:   Device type. Examples - Proxy, IDS, Web Server

type: keyword


**`cef.extensions.categoryObject`**
:   Object that the event is about. For example it can be an operating sytem, database, file, etc.

type: keyword


**`cef.extensions.categoryBehavior`**
:   Action or a behavior associated with an event. It’s what is being done to the object.

type: keyword


**`cef.extensions.categoryTechnique`**
:   Technique being used (e.g. /DoS).

type: keyword


**`cef.extensions.categoryDeviceGroup`**
:   General device group like Firewall.

type: keyword


**`cef.extensions.categorySignificance`**
:   Characterization of the importance of the event.

type: keyword


**`cef.extensions.categoryOutcome`**
:   Outcome of the event (e.g. sucess, failure, or attempt).

type: keyword


**`cef.extensions.managerReceiptTime`**
:   When the Arcsight ESM received the event.

type: date


**`source.service.name`**
:   Service that is the source of the event.

type: keyword


**`destination.service.name`**
:   Service that is the target of the event.

type: keyword


