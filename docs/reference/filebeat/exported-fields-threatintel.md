---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-threatintel.html
---

# threatintel fields [exported-fields-threatintel]

Threat intelligence Filebeat Module.

**`threat.indicator.file.hash.tlsh`**
:   The file’s import tlsh, if available.

type: keyword


**`threat.indicator.file.hash.sha384`**
:   The file’s sha384 hash, if available.

type: keyword


**`threat.feed.name`**
:   type: keyword


**`threat.feed.dashboard_id`**
:   type: keyword



## abusech.malware [_abusech_malware]

Fields for AbuseCH Malware Threat Intel

**`abusech.malware.file_type`**
:   File type guessed by URLhaus.

type: keyword


**`abusech.malware.signature`**
:   Malware familiy.

type: keyword


**`abusech.malware.urlhaus_download`**
:   Location (URL) where you can download a copy of this file.

type: keyword


**`abusech.malware.virustotal.result`**
:   AV detection ration.

type: keyword


**`abusech.malware.virustotal.percent`**
:   AV detection in percent.

type: float


**`abusech.malware.virustotal.link`**
:   Link to the Virustotal report.

type: keyword



## abusech.url [_abusech_url]

Fields for AbuseCH Malware Threat Intel

**`abusech.url.id`**
:   The ID of the url.

type: keyword


**`abusech.url.urlhaus_reference`**
:   Link to URLhaus entry.

type: keyword


**`abusech.url.url_status`**
:   The current status of the URL. Possible values are: online, offline and unknown.

type: keyword


**`abusech.url.threat`**
:   The threat corresponding to this malware URL.

type: keyword


**`abusech.url.blacklists.surbl`**
:   SURBL blacklist status. Possible values are: listed and not_listed

type: keyword


**`abusech.url.blacklists.spamhaus_dbl`**
:   Spamhaus DBL blacklist status.

type: keyword


**`abusech.url.reporter`**
:   The Twitter handle of the reporter that has reported this malware URL (or anonymous).

type: keyword


**`abusech.url.larted`**
:   Indicates whether the malware URL has been reported to the hosting provider (true or false)

type: boolean


**`abusech.url.tags`**
:   A list of tags associated with the queried malware URL

type: keyword



## anomali.limo [_anomali_limo]

Fields for Anomali Threat Intel

**`anomali.limo.id`**
:   The ID of the indicator.

type: keyword


**`anomali.limo.name`**
:   The name of the indicator.

type: keyword


**`anomali.limo.pattern`**
:   The pattern ID of the indicator.

type: keyword


**`anomali.limo.valid_from`**
:   When the indicator was first found or is considered valid.

type: date


**`anomali.limo.modified`**
:   When the indicator was last modified

type: date


**`anomali.limo.labels`**
:   The labels related to the indicator

type: keyword


**`anomali.limo.indicator`**
:   The value of the indicator, for example if the type is domain, this would be the value.

type: keyword


**`anomali.limo.description`**
:   A description of the indicator.

type: keyword


**`anomali.limo.title`**
:   Title describing the indicator.

type: keyword


**`anomali.limo.content`**
:   Extra text or descriptive content related to the indicator.

type: keyword


**`anomali.limo.type`**
:   The indicator type, can for example be "domain, email, FileHash-SHA256".

type: keyword


**`anomali.limo.object_marking_refs`**
:   The STIX reference object.

type: keyword



## anomali.threatstream [_anomali_threatstream]

Fields for Anomali ThreatStream

**`anomali.threatstream.classification`**
:   Indicates whether an indicator is private or from a public feed and available publicly. Possible values: private, public.

type: keyword

example: private


**`anomali.threatstream.confidence`**
:   The measure of the accuracy (from 0 to 100) assigned by ThreatStream’s predictive analytics technology to indicators.

type: short


**`anomali.threatstream.detail2`**
:   Detail text for indicator.

type: text

example: Imported by user 42.


**`anomali.threatstream.id`**
:   The ID of the indicator.

type: keyword


**`anomali.threatstream.import_session_id`**
:   ID of the import session that created the indicator on ThreatStream.

type: keyword


**`anomali.threatstream.itype`**
:   Indicator type. Possible values: "apt_domain", "apt_email", "apt_ip", "apt_url", "bot_ip", "c2_domain", "c2_ip", "c2_url", "i2p_ip", "mal_domain", "mal_email", "mal_ip", "mal_md5", "mal_url", "parked_ip", "phish_email", "phish_ip", "phish_url", "scan_ip", "spam_domain", "ssh_ip", "suspicious_domain", "tor_ip" and "torrent_tracker_url".

type: keyword


**`anomali.threatstream.maltype`**
:   Information regarding a malware family, a CVE ID, or another attack or threat, associated with the indicator.

type: wildcard


**`anomali.threatstream.md5`**
:   Hash for the indicator.

type: keyword


**`anomali.threatstream.resource_uri`**
:   Relative URI for the indicator details.

type: keyword


**`anomali.threatstream.severity`**
:   Criticality associated with the threat feed that supplied the indicator. Possible values: low, medium, high, very-high.

type: keyword


**`anomali.threatstream.source`**
:   Source for the indicator.

type: keyword

example: Analyst


**`anomali.threatstream.source_feed_id`**
:   ID for the integrator source.

type: keyword


**`anomali.threatstream.state`**
:   State for this indicator.

type: keyword

example: active


**`anomali.threatstream.trusted_circle_ids`**
:   ID of the trusted circle that imported the indicator.

type: keyword


**`anomali.threatstream.update_id`**
:   Update ID.

type: keyword


**`anomali.threatstream.url`**
:   URL for the indicator.

type: keyword


**`anomali.threatstream.value_type`**
:   Data type of the indicator. Possible values: ip, domain, url, email, md5.

type: keyword



## abusech.malwarebazaar [_abusech_malwarebazaar]

Fields for Malware Bazaar Threat Intel

**`abusech.malwarebazaar.file_type`**
:   File type guessed by Malware Bazaar.

type: keyword


**`abusech.malwarebazaar.signature`**
:   Malware familiy.

type: keyword


**`abusech.malwarebazaar.tags`**
:   A list of tags associated with the queried malware sample.

type: keyword


**`abusech.malwarebazaar.intelligence.downloads`**
:   Number of downloads from MalwareBazaar.

type: long


**`abusech.malwarebazaar.intelligence.uploads`**
:   Number of uploads from MalwareBazaar.

type: long


**`abusech.malwarebazaar.intelligence.mail.Generic`**
:   Malware seen in generic spam traffic.

type: keyword


**`abusech.malwarebazaar.intelligence.mail.IT`**
:   Malware seen in IT spam traffic.

type: keyword


**`abusech.malwarebazaar.anonymous`**
:   Identifies if the sample was submitted anonymously.

type: long


**`abusech.malwarebazaar.code_sign`**
:   Code signing information for the sample.

type: nested



## misp [_misp_2]

Fields for MISP Threat Intel

**`misp.id`**
:   Attribute ID.

type: keyword


**`misp.orgc_id`**
:   Organization Community ID of the event.

type: keyword


**`misp.org_id`**
:   Organization ID of the event.

type: keyword


**`misp.threat_level_id`**
:   Threat level from 5 to 1, where 1 is the most critical.

type: long


**`misp.info`**
:   Additional text or information related to the event.

type: keyword


**`misp.published`**
:   When the event was published.

type: boolean


**`misp.uuid`**
:   The UUID of the event object.

type: keyword


**`misp.date`**
:   The date of when the event object was created.

type: date


**`misp.attribute_count`**
:   How many attributes are included in a single event object.

type: long


**`misp.timestamp`**
:   The timestamp of when the event object was created.

type: date


**`misp.distribution`**
:   Distribution type related to MISP.

type: keyword


**`misp.proposal_email_lock`**
:   Settings configured on MISP for email lock on this event object.

type: boolean


**`misp.locked`**
:   If the current MISP event object is locked or not.

type: boolean


**`misp.publish_timestamp`**
:   At what time the event object was published

type: date


**`misp.sharing_group_id`**
:   The ID of the grouped events or sources of the event.

type: keyword


**`misp.disable_correlation`**
:   If correlation is disabled on the MISP event object.

type: boolean


**`misp.extends_uuid`**
:   The UUID of the event object it might extend.

type: keyword


**`misp.org.id`**
:   The organization ID related to the event object.

type: keyword


**`misp.org.name`**
:   The organization name related to the event object.

type: keyword


**`misp.org.uuid`**
:   The UUID of the organization related to the event object.

type: keyword


**`misp.org.local`**
:   If the event object is local or from a remote source.

type: boolean


**`misp.orgc.id`**
:   The Organization Community ID in which the event object was reported from.

type: keyword


**`misp.orgc.name`**
:   The Organization Community name in which the event object was reported from.

type: keyword


**`misp.orgc.uuid`**
:   The Organization Community UUID in which the event object was reported from.

type: keyword


**`misp.orgc.local`**
:   If the Organization Community was local or synced from a remote source.

type: boolean


**`misp.attribute.id`**
:   The ID of the attribute related to the event object.

type: keyword


**`misp.attribute.type`**
:   The type of the attribute related to the event object. For example email, ipv4, sha1 and such.

type: keyword


**`misp.attribute.category`**
:   The category of the attribute related to the event object. For example "Network Activity".

type: keyword


**`misp.attribute.to_ids`**
:   If the attribute should be automatically synced with an IDS.

type: boolean


**`misp.attribute.uuid`**
:   The UUID of the attribute related to the event.

type: keyword


**`misp.attribute.event_id`**
:   The local event ID of the attribute related to the event.

type: keyword


**`misp.attribute.distribution`**
:   How the attribute has been distributed, represented by integer numbers.

type: long


**`misp.attribute.timestamp`**
:   The timestamp in which the attribute was attached to the event object.

type: date


**`misp.attribute.comment`**
:   Comments made to the attribute itself.

type: keyword


**`misp.attribute.sharing_group_id`**
:   The group ID of the sharing group related to the specific attribute.

type: keyword


**`misp.attribute.deleted`**
:   If the attribute has been removed from the event object.

type: boolean


**`misp.attribute.disable_correlation`**
:   If correlation has been enabled on the attribute related to the event object.

type: boolean


**`misp.attribute.object_id`**
:   The ID of the Object in which the attribute is attached.

type: keyword


**`misp.attribute.object_relation`**
:   The type of relation the attribute has with the event object itself.

type: keyword


**`misp.attribute.value`**
:   The value of the attribute, depending on the type like "url, sha1, email-src".

type: keyword


**`misp.context.attribute.id`**
:   The ID of the secondary attribute related to the event object.

type: keyword


**`misp.context.attribute.type`**
:   The type of the secondary attribute related to the event object. For example email, ipv4, sha1 and such.

type: keyword


**`misp.context.attribute.category`**
:   The category of the secondary attribute related to the event object. For example "Network Activity".

type: keyword


**`misp.context.attribute.to_ids`**
:   If the secondary attribute should be automatically synced with an IDS.

type: boolean


**`misp.context.attribute.uuid`**
:   The UUID of the secondary attribute related to the event.

type: keyword


**`misp.context.attribute.event_id`**
:   The local event ID of the secondary attribute related to the event.

type: keyword


**`misp.context.attribute.distribution`**
:   How the secondary attribute has been distributed, represented by integer numbers.

type: long


**`misp.context.attribute.timestamp`**
:   The timestamp in which the secondary attribute was attached to the event object.

type: date


**`misp.context.attribute.comment`**
:   Comments made to the secondary attribute itself.

type: keyword


**`misp.context.attribute.sharing_group_id`**
:   The group ID of the sharing group related to the specific secondary attribute.

type: keyword


**`misp.context.attribute.deleted`**
:   If the secondary attribute has been removed from the event object.

type: boolean


**`misp.context.attribute.disable_correlation`**
:   If correlation has been enabled on the secondary attribute related to the event object.

type: boolean


**`misp.context.attribute.object_id`**
:   The ID of the Object in which the secondary attribute is attached.

type: keyword


**`misp.context.attribute.object_relation`**
:   The type of relation the secondary attribute has with the event object itself.

type: keyword


**`misp.context.attribute.value`**
:   The value of the attribute, depending on the type like "url, sha1, email-src".

type: keyword



## otx [_otx]

Fields for OTX Threat Intel

**`otx.id`**
:   The ID of the indicator.

type: keyword


**`otx.indicator`**
:   The value of the indicator, for example if the type is domain, this would be the value.

type: keyword


**`otx.description`**
:   A description of the indicator.

type: keyword


**`otx.title`**
:   Title describing the indicator.

type: keyword


**`otx.content`**
:   Extra text or descriptive content related to the indicator.

type: keyword


**`otx.type`**
:   The indicator type, can for example be "domain, email, FileHash-SHA256".

type: keyword



## threatq [_threatq]

Fields for ThreatQ Threat Library

**`threatq.updated_at`**
:   Last modification time

type: date


**`threatq.created_at`**
:   Object creation time

type: date


**`threatq.expires_at`**
:   Expiration time

type: date


**`threatq.expires_calculated_at`**
:   Expiration calculation time

type: date


**`threatq.published_at`**
:   Object publication time

type: date


**`threatq.status`**
:   Object status within the Threat Library

type: keyword


**`threatq.indicator_value`**
:   Original indicator value

type: keyword


**`threatq.adversaries`**
:   Adversaries that are linked to the object

type: keyword


**`threatq.attributes`**
:   These provide additional context about an object

type: flattened


