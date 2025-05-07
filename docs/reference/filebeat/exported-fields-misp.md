---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-misp.html
---

# MISP fields [exported-fields-misp]

Module for handling threat information from MISP.


## misp [_misp]

Fields from MISP threat information.


## attack_pattern [_attack_pattern]

Fields provide support for specifying information about attack patterns.

**`misp.attack_pattern.id`**
:   Identifier of the threat indicator.

type: keyword


**`misp.attack_pattern.name`**
:   Name of the attack pattern.

type: keyword


**`misp.attack_pattern.description`**
:   Description of the attack pattern.

type: text


**`misp.attack_pattern.kill_chain_phases`**
:   The kill chain phase(s) to which this attack pattern corresponds.

type: keyword



## campaign [_campaign]

Fields provide support for specifying information about campaigns.

**`misp.campaign.id`**
:   Identifier of the campaign.

type: keyword


**`misp.campaign.name`**
:   Name of the campaign.

type: keyword


**`misp.campaign.description`**
:   Description of the campaign.

type: text


**`misp.campaign.aliases`**
:   Alternative names used to identify this campaign.

type: text


**`misp.campaign.first_seen`**
:   The time that this Campaign was first seen, in RFC3339 format.

type: date


**`misp.campaign.last_seen`**
:   The time that this Campaign was last seen, in RFC3339 format.

type: date


**`misp.campaign.objective`**
:   This field defines the Campaign’s primary goal, objective, desired outcome, or intended effect.

type: keyword



## course_of_action [_course_of_action]

A Course of Action is an action taken either to prevent an attack or to respond to an attack that is in progress.

**`misp.course_of_action.id`**
:   Identifier of the Course of Action.

type: keyword


**`misp.course_of_action.name`**
:   The name used to identify the Course of Action.

type: keyword


**`misp.course_of_action.description`**
:   Description of the Course of Action.

type: text



## identity [_identity_2]

Identity can represent actual individuals, organizations, or groups, as well as classes of individuals, organizations, or groups.

**`misp.identity.id`**
:   Identifier of the Identity.

type: keyword


**`misp.identity.name`**
:   The name used to identify the Identity.

type: keyword


**`misp.identity.description`**
:   Description of the Identity.

type: text


**`misp.identity.identity_class`**
:   The type of entity that this Identity describes, e.g., an individual or organization. Open Vocab - identity-class-ov

type: keyword


**`misp.identity.labels`**
:   The list of roles that this Identity performs.

type: keyword

example: CEO


**`misp.identity.sectors`**
:   The list of sectors that this Identity belongs to. Open Vocab - industry-sector-ov

type: keyword


**`misp.identity.contact_information`**
:   The contact information (e-mail, phone number, etc.) for this Identity.

type: text



## intrusion_set [_intrusion_set]

An Intrusion Set is a grouped set of adversary behavior and resources with common properties that is believed to be orchestrated by a single organization.

**`misp.intrusion_set.id`**
:   Identifier of the Intrusion Set.

type: keyword


**`misp.intrusion_set.name`**
:   The name used to identify the Intrusion Set.

type: keyword


**`misp.intrusion_set.description`**
:   Description of the Intrusion Set.

type: text


**`misp.intrusion_set.aliases`**
:   Alternative names used to identify the Intrusion Set.

type: text


**`misp.intrusion_set.first_seen`**
:   The time that this Intrusion Set was first seen, in RFC3339 format.

type: date


**`misp.intrusion_set.last_seen`**
:   The time that this Intrusion Set was last seen, in RFC3339 format.

type: date


**`misp.intrusion_set.goals`**
:   The high level goals of this Intrusion Set, namely, what are they trying to do.

type: text


**`misp.intrusion_set.resource_level`**
:   This defines the organizational level at which this Intrusion Set typically works. Open Vocab - attack-resource-level-ov

type: text


**`misp.intrusion_set.primary_motivation`**
:   The primary reason, motivation, or purpose behind this Intrusion Set. Open Vocab - attack-motivation-ov

type: text


**`misp.intrusion_set.secondary_motivations`**
:   The secondary reasons, motivations, or purposes behind this Intrusion Set. Open Vocab - attack-motivation-ov

type: text



## malware [_malware]

Malware is a type of TTP that is also known as malicious code and malicious software, refers to a program that is inserted into a system, usually covertly, with the intent of compromising the confidentiality, integrity, or availability of the victim’s data, applications, or operating system (OS) or of otherwise annoying or disrupting the victim.

**`misp.malware.id`**
:   Identifier of the Malware.

type: keyword


**`misp.malware.name`**
:   The name used to identify the Malware.

type: keyword


**`misp.malware.description`**
:   Description of the Malware.

type: text


**`misp.malware.labels`**
:   The type of malware being described.  Open Vocab - malware-label-ov.  adware,backdoor,bot,ddos,dropper,exploit-kit,keylogger,ransomware, remote-access-trojan,resource-exploitation,rogue-security-software,rootkit, screen-capture,spyware,trojan,virus,worm

type: keyword


**`misp.malware.kill_chain_phases`**
:   The list of kill chain phases for which this Malware instance can be used.

type: keyword

format: string



## note [_note]

A Note is a comment or note containing informative text to help explain the context of one or more STIX Objects (SDOs or SROs) or to provide additional analysis that is not contained in the original object.

**`misp.note.id`**
:   Identifier of the Note.

type: keyword


**`misp.note.summary`**
:   A brief description used as a summary of the Note.

type: keyword


**`misp.note.description`**
:   The content of the Note.

type: text


**`misp.note.authors`**
:   The name of the author(s) of this Note.

type: keyword


**`misp.note.object_refs`**
:   The STIX Objects (SDOs and SROs) that the note is being applied to.

type: keyword



## threat_indicator [_threat_indicator]

Fields provide support for specifying information about threat indicators, and related matching patterns.

**`misp.threat_indicator.labels`**
:   list of type open-vocab that specifies the type of indicator.

type: keyword

example: Domain Watchlist


**`misp.threat_indicator.id`**
:   Identifier of the threat indicator.

type: keyword


**`misp.threat_indicator.version`**
:   Version of the threat indicator.

type: keyword


**`misp.threat_indicator.type`**
:   Type of the threat indicator.

type: keyword


**`misp.threat_indicator.description`**
:   Description of the threat indicator.

type: text


**`misp.threat_indicator.feed`**
:   Name of the threat feed.

type: text


**`misp.threat_indicator.valid_from`**
:   The time from which this Indicator should be considered valuable  intelligence, in RFC3339 format.

type: date


**`misp.threat_indicator.valid_until`**
:   The time at which this Indicator should no longer be considered valuable intelligence. If the valid_until property is omitted, then there is no constraint on the latest time for which the indicator should be used, in RFC3339 format.

type: date


**`misp.threat_indicator.severity`**
:   Threat severity to which this indicator corresponds.

type: keyword

example: high

format: string


**`misp.threat_indicator.confidence`**
:   Confidence level to which this indicator corresponds.

type: keyword

example: high


**`misp.threat_indicator.kill_chain_phases`**
:   The kill chain phase(s) to which this indicator corresponds.

type: keyword

format: string


**`misp.threat_indicator.mitre_tactic`**
:   MITRE tactics to which this indicator corresponds.

type: keyword

example: Initial Access

format: string


**`misp.threat_indicator.mitre_technique`**
:   MITRE techniques to which this indicator corresponds.

type: keyword

example: Drive-by Compromise

format: string


**`misp.threat_indicator.attack_pattern`**
:   The attack_pattern for this indicator is a STIX Pattern as specified in STIX Version 2.0 Part 5 - STIX Patterning.

type: keyword

example: [destination:ip = *91.219.29.188/32*]


**`misp.threat_indicator.attack_pattern_kql`**
:   The attack_pattern for this indicator is KQL query that matches the attack_pattern specified in the STIX Pattern format.

type: keyword

example: destination.ip: "91.219.29.188/32"


**`misp.threat_indicator.negate`**
:   When set to true, it specifies the absence of the attack_pattern.

type: boolean


**`misp.threat_indicator.intrusion_set`**
:   Name of the intrusion set if known.

type: keyword


**`misp.threat_indicator.campaign`**
:   Name of the attack campaign if known.

type: keyword


**`misp.threat_indicator.threat_actor`**
:   Name of the threat actor if known.

type: keyword



## observed_data [_observed_data]

Observed data conveys information that was observed on systems and networks, such as log data or network traffic, using the Cyber Observable specification.

**`misp.observed_data.id`**
:   Identifier of the Observed Data.

type: keyword


**`misp.observed_data.first_observed`**
:   The beginning of the time window that the data was observed, in RFC3339 format.

type: date


**`misp.observed_data.last_observed`**
:   The end of the time window that the data was observed, in RFC3339 format.

type: date


**`misp.observed_data.number_observed`**
:   The number of times the data represented in the objects property was observed. This MUST be an integer between 1 and 999,999,999 inclusive.

type: integer


**`misp.observed_data.objects`**
:   A dictionary of Cyber Observable Objects that describes the single fact that was observed.

type: keyword



## report [_report]

Reports are collections of threat intelligence focused on one or more topics, such as a description of a threat actor, malware, or attack technique, including context and related details.

**`misp.report.id`**
:   Identifier of the Report.

type: keyword


**`misp.report.labels`**
:   This field is an Open Vocabulary that specifies the primary subject of this report.  Open Vocab - report-label-ov. threat-report,attack-pattern,campaign,identity,indicator,malware,observed-data,threat-actor,tool,vulnerability

type: keyword


**`misp.report.name`**
:   The name used to identify the Report.

type: keyword


**`misp.report.description`**
:   A description that provides more details and context about Report.

type: text


**`misp.report.published`**
:   The date that this report object was officially published by the creator of this report, in RFC3339 format.

type: date


**`misp.report.object_refs`**
:   Specifies the STIX Objects that are referred to by this Report.

type: text



## threat_actor [_threat_actor]

Threat Actors are actual individuals, groups, or organizations believed to be operating with malicious intent.

**`misp.threat_actor.id`**
:   Identifier of the Threat Actor.

type: keyword


**`misp.threat_actor.labels`**
:   This field specifies the type of threat actor.  Open Vocab - threat-actor-label-ov. activist,competitor,crime-syndicate,criminal,hacker,insider-accidental,insider-disgruntled,nation-state,sensationalist,spy,terrorist

type: keyword


**`misp.threat_actor.name`**
:   The name used to identify this Threat Actor or Threat Actor group.

type: keyword


**`misp.threat_actor.description`**
:   A description that provides more details and context about the Threat Actor.

type: text


**`misp.threat_actor.aliases`**
:   A list of other names that this Threat Actor is believed to use.

type: text


**`misp.threat_actor.roles`**
:   This is a list of roles the Threat Actor plays.  Open Vocab - threat-actor-role-ov. agent,director,independent,sponsor,infrastructure-operator,infrastructure-architect,malware-author

type: text


**`misp.threat_actor.goals`**
:   The high level goals of this Threat Actor, namely, what are they trying to do.

type: text


**`misp.threat_actor.sophistication`**
:   The skill, specific knowledge, special training, or expertise a Threat Actor  must have to perform the attack.  Open Vocab - threat-actor-sophistication-ov. none,minimal,intermediate,advanced,strategic,expert,innovator

type: text


**`misp.threat_actor.resource_level`**
:   This defines the organizational level at which this Threat Actor typically works.  Open Vocab - attack-resource-level-ov. individual,club,contest,team,organization,government

type: text


**`misp.threat_actor.primary_motivation`**
:   The primary reason, motivation, or purpose behind this Threat Actor.  Open Vocab - attack-motivation-ov. accidental,coercion,dominance,ideology,notoriety,organizational-gain,personal-gain,personal-satisfaction,revenge,unpredictable

type: text


**`misp.threat_actor.secondary_motivations`**
:   The secondary reasons, motivations, or purposes behind this Threat Actor.  Open Vocab - attack-motivation-ov. accidental,coercion,dominance,ideology,notoriety,organizational-gain,personal-gain,personal-satisfaction,revenge,unpredictable

type: text


**`misp.threat_actor.personal_motivations`**
:   The personal reasons, motivations, or purposes of the Threat Actor regardless of  organizational goals. Open Vocab - attack-motivation-ov. accidental,coercion,dominance,ideology,notoriety,organizational-gain,personal-gain,personal-satisfaction,revenge,unpredictable

type: text



## tool [_tool]

Tools are legitimate software that can be used by threat actors to perform attacks.

**`misp.tool.id`**
:   Identifier of the Tool.

type: keyword


**`misp.tool.labels`**
:   The kind(s) of tool(s) being described.  Open Vocab - tool-label-ov. denial-of-service,exploitation,information-gathering,network-capture,credential-exploitation,remote-access,vulnerability-scanning

type: keyword


**`misp.tool.name`**
:   The name used to identify the Tool.

type: keyword


**`misp.tool.description`**
:   A description that provides more details and context about the Tool.

type: text


**`misp.tool.tool_version`**
:   The version identifier associated with the Tool.

type: keyword


**`misp.tool.kill_chain_phases`**
:   The list of kill chain phases for which this Tool instance can be used.

type: text



## vulnerability [_vulnerability_2]

A Vulnerability is a mistake in software that can be directly used by a hacker to gain access to a system or network.

**`misp.vulnerability.id`**
:   Identifier of the Vulnerability.

type: keyword


**`misp.vulnerability.name`**
:   The name used to identify the Vulnerability.

type: keyword


**`misp.vulnerability.description`**
:   A description that provides more details and context about the Vulnerability.

type: text


