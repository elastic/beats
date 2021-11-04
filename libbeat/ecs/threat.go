// Licensed to Elasticsearch B.V. under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. Elasticsearch B.V. licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package ecs

import (
	"time"
)

// Fields to classify events and alerts according to a threat taxonomy such as
// the MITRE ATT&CK® framework.
// These fields are for users to classify alerts from all of their sources
// (e.g. IDS, NGFW, etc.) within a common taxonomy. The threat.tactic.* are
// meant to capture the high level category of the threat (e.g. "impact"). The
// threat.technique.* fields are meant to capture which kind of approach is
// used by this detected threat, to accomplish the goal (e.g. "endpoint denial
// of service").
type Threat struct {
	// A list of associated indicators objects enriching the event, and the
	// context of that association/enrichment.
	Enrichments []Enrichments `ecs:"enrichments"`

	// Name of the threat framework used to further categorize and classify the
	// tactic and technique of the reported threat. Framework classification
	// can be provided by detecting systems, evaluated at ingest time, or
	// retrospectively tagged to events.
	Framework string `ecs:"framework"`

	// The alias(es) of the group for a set of related intrusion activity that
	// are tracked by a common name in the security community.
	// While not required, you can use a MITRE ATT&CK® group alias(es).
	GroupAlias string `ecs:"group.alias"`

	// The id of the group for a set of related intrusion activity that are
	// tracked by a common name in the security community.
	// While not required, you can use a MITRE ATT&CK® group id.
	GroupID string `ecs:"group.id"`

	// The name of the group for a set of related intrusion activity that are
	// tracked by a common name in the security community.
	// While not required, you can use a MITRE ATT&CK® group name.
	GroupName string `ecs:"group.name"`

	// The reference URL of the group for a set of related intrusion activity
	// that are tracked by a common name in the security community.
	// While not required, you can use a MITRE ATT&CK® group reference URL.
	GroupReference string `ecs:"group.reference"`

	// The date and time when intelligence source first reported sighting this
	// indicator.
	IndicatorFirstSeen time.Time `ecs:"indicator.first_seen"`

	// The date and time when intelligence source last reported sighting this
	// indicator.
	IndicatorLastSeen time.Time `ecs:"indicator.last_seen"`

	// The date and time when intelligence source last modified information for
	// this indicator.
	IndicatorModifiedAt time.Time `ecs:"indicator.modified_at"`

	// Number of times this indicator was observed conducting threat activity.
	IndicatorSightings int64 `ecs:"indicator.sightings"`

	// Type of indicator as represented by Cyber Observable in STIX 2.0.
	// Recommended values:
	//   * autonomous-system
	//   * artifact
	//   * directory
	//   * domain-name
	//   * email-addr
	//   * file
	//   * ipv4-addr
	//   * ipv6-addr
	//   * mac-addr
	//   * mutex
	//   * port
	//   * process
	//   * software
	//   * url
	//   * user-account
	//   * windows-registry-key
	//   * x509-certificate
	IndicatorType string `ecs:"indicator.type"`

	// Describes the type of action conducted by the threat.
	IndicatorDescription string `ecs:"indicator.description"`

	// Count of AV/EDR vendors that successfully detected malicious file or
	// URL.
	IndicatorScannerStats int64 `ecs:"indicator.scanner_stats"`

	// Identifies the confidence rating assigned by the provider using STIX
	// confidence scales.
	// Recommended values:
	//   * Not Specified, None, Low, Medium, High
	//   * 0-10
	//   * Admirality Scale (1-6)
	//   * DNI Scale (5-95)
	//   * WEP Scale (Impossible - Certain)
	IndicatorConfidence string `ecs:"indicator.confidence"`

	// Identifies a threat indicator as an IP address (irrespective of
	// direction).
	IndicatorIP string `ecs:"indicator.ip"`

	// Identifies a threat indicator as a port number (irrespective of
	// direction).
	IndicatorPort int64 `ecs:"indicator.port"`

	// Identifies a threat indicator as an email address (irrespective of
	// direction).
	IndicatorEmailAddress string `ecs:"indicator.email.address"`

	// Traffic Light Protocol sharing markings.
	// Recommended values are:
	//   * WHITE
	//   * GREEN
	//   * AMBER
	//   * RED
	IndicatorMarkingTlp string `ecs:"indicator.marking.tlp"`

	// Reference URL linking to additional information about this indicator.
	IndicatorReference string `ecs:"indicator.reference"`

	// The name of the indicator's provider.
	IndicatorProvider string `ecs:"indicator.provider"`

	// The id of the software used by this threat to conduct behavior commonly
	// modeled using MITRE ATT&CK®.
	// While not required, you can use a MITRE ATT&CK® software id.
	SoftwareID string `ecs:"software.id"`

	// The name of the software used by this threat to conduct behavior
	// commonly modeled using MITRE ATT&CK®.
	// While not required, you can use a MITRE ATT&CK® software name.
	SoftwareName string `ecs:"software.name"`

	// The alias(es) of the software for a set of related intrusion activity
	// that are tracked by a common name in the security community.
	// While not required, you can use a MITRE ATT&CK® associated software
	// description.
	SoftwareAlias string `ecs:"software.alias"`

	// The platforms of the software used by this threat to conduct behavior
	// commonly modeled using MITRE ATT&CK®.
	// Recommended Values:
	//   * AWS
	//   * Azure
	//   * Azure AD
	//   * GCP
	//   * Linux
	//   * macOS
	//   * Network
	//   * Office 365
	//   * SaaS
	//   * Windows
	//
	// While not required, you can use a MITRE ATT&CK® software platforms.
	SoftwarePlatforms string `ecs:"software.platforms"`

	// The reference URL of the software used by this threat to conduct
	// behavior commonly modeled using MITRE ATT&CK®.
	// While not required, you can use a MITRE ATT&CK® software reference URL.
	SoftwareReference string `ecs:"software.reference"`

	// The type of software used by this threat to conduct behavior commonly
	// modeled using MITRE ATT&CK®.
	// Recommended values
	//   * Malware
	//   * Tool
	//
	//  While not required, you can use a MITRE ATT&CK® software type.
	SoftwareType string `ecs:"software.type"`

	// The id of tactic used by this threat. You can use a MITRE ATT&CK®
	// tactic, for example. (ex. https://attack.mitre.org/tactics/TA0002/ )
	TacticID string `ecs:"tactic.id"`

	// Name of the type of tactic used by this threat. You can use a MITRE
	// ATT&CK® tactic, for example. (ex.
	// https://attack.mitre.org/tactics/TA0002/)
	TacticName string `ecs:"tactic.name"`

	// The reference url of tactic used by this threat. You can use a MITRE
	// ATT&CK® tactic, for example. (ex.
	// https://attack.mitre.org/tactics/TA0002/ )
	TacticReference string `ecs:"tactic.reference"`

	// The id of technique used by this threat. You can use a MITRE ATT&CK®
	// technique, for example. (ex. https://attack.mitre.org/techniques/T1059/)
	TechniqueID string `ecs:"technique.id"`

	// The name of technique used by this threat. You can use a MITRE ATT&CK®
	// technique, for example. (ex. https://attack.mitre.org/techniques/T1059/)
	TechniqueName string `ecs:"technique.name"`

	// The reference url of technique used by this threat. You can use a MITRE
	// ATT&CK® technique, for example. (ex.
	// https://attack.mitre.org/techniques/T1059/)
	TechniqueReference string `ecs:"technique.reference"`

	// The full id of subtechnique used by this threat. You can use a MITRE
	// ATT&CK® subtechnique, for example. (ex.
	// https://attack.mitre.org/techniques/T1059/001/)
	TechniqueSubtechniqueID string `ecs:"technique.subtechnique.id"`

	// The name of subtechnique used by this threat. You can use a MITRE
	// ATT&CK® subtechnique, for example. (ex.
	// https://attack.mitre.org/techniques/T1059/001/)
	TechniqueSubtechniqueName string `ecs:"technique.subtechnique.name"`

	// The reference url of subtechnique used by this threat. You can use a
	// MITRE ATT&CK® subtechnique, for example. (ex.
	// https://attack.mitre.org/techniques/T1059/001/)
	TechniqueSubtechniqueReference string `ecs:"technique.subtechnique.reference"`
}

type Enrichments struct {
	// Object containing associated indicators enriching the event.
	Indicator map[string]interface{} `ecs:"indicator"`

	// The date and time when intelligence source first reported sighting this
	// indicator.
	IndicatorFirstSeen time.Time `ecs:"indicator.first_seen"`

	// The date and time when intelligence source last reported sighting this
	// indicator.
	IndicatorLastSeen time.Time `ecs:"indicator.last_seen"`

	// The date and time when intelligence source last modified information for
	// this indicator.
	IndicatorModifiedAt time.Time `ecs:"indicator.modified_at"`

	// Number of times this indicator was observed conducting threat activity.
	IndicatorSightings int64 `ecs:"indicator.sightings"`

	// Type of indicator as represented by Cyber Observable in STIX 2.0.
	// Recommended values:
	//   * autonomous-system
	//   * artifact
	//   * directory
	//   * domain-name
	//   * email-addr
	//   * file
	//   * ipv4-addr
	//   * ipv6-addr
	//   * mac-addr
	//   * mutex
	//   * port
	//   * process
	//   * software
	//   * url
	//   * user-account
	//   * windows-registry-key
	//   * x509-certificate
	IndicatorType string `ecs:"indicator.type"`

	// Describes the type of action conducted by the threat.
	IndicatorDescription string `ecs:"indicator.description"`

	// Count of AV/EDR vendors that successfully detected malicious file or
	// URL.
	IndicatorScannerStats int64 `ecs:"indicator.scanner_stats"`

	// Identifies the confidence rating assigned by the provider using
	// STIX confidence scales. Expected values:
	//   * Not Specified, None, Low, Medium, High
	//   * 0-10
	//   * Admirality Scale (1-6)
	//   * DNI Scale (5-95)
	//   * WEP Scale (Impossible - Certain)
	IndicatorConfidence string `ecs:"indicator.confidence"`

	// Identifies a threat indicator as an IP address (irrespective of
	// direction).
	IndicatorIP string `ecs:"indicator.ip"`

	// Identifies a threat indicator as a port number (irrespective of
	// direction).
	IndicatorPort int64 `ecs:"indicator.port"`

	// Identifies a threat indicator as an email address (irrespective of
	// direction).
	IndicatorEmailAddress string `ecs:"indicator.email.address"`

	// Traffic Light Protocol sharing markings. Recommended values are:
	//   * WHITE
	//   * GREEN
	//   * AMBER
	//   * RED
	IndicatorMarkingTlp string `ecs:"indicator.marking.tlp"`

	// Reference URL linking to additional information about this indicator.
	IndicatorReference string `ecs:"indicator.reference"`

	// The name of the indicator's provider.
	IndicatorProvider string `ecs:"indicator.provider"`

	// Identifies the atomic indicator value that matched a local environment
	// endpoint or network event.
	MatchedAtomic string `ecs:"matched.atomic"`

	// Identifies the field of the atomic indicator that matched a local
	// environment endpoint or network event.
	MatchedField string `ecs:"matched.field"`

	// Identifies the _id of the indicator document enriching the event.
	MatchedID string `ecs:"matched.id"`

	// Identifies the _index of the indicator document enriching the event.
	MatchedIndex string `ecs:"matched.index"`

	// Identifies the type of match that caused the event to be enriched with
	// the given indicator
	MatchedType string `ecs:"matched.type"`
}
