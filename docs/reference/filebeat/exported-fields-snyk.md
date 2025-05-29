---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/exported-fields-snyk.html
---

# Snyk fields [exported-fields-snyk]

Snyk module


## snyk [_snyk]

Module for parsing Snyk project vulnerabilities.

**`snyk.projects`**
:   Array with all related projects objects.

type: flattened


**`snyk.related.projects`**
:   Array of all the related project ID’s.

type: keyword



## audit [_audit_5]

Module for parsing Snyk audit logs.

**`snyk.audit.org_id`**
:   ID of the related Organization related to the event.

type: keyword


**`snyk.audit.project_id`**
:   ID of the project related to the event.

type: keyword


**`snyk.audit.content`**
:   Overview of the content that was changed, both old and new values.

type: flattened



## vulnerabilities [_vulnerabilities]

Module for parsing Snyk project vulnerabilities.

**`snyk.vulnerabilities.cvss3`**
:   CSSv3 scores.

type: keyword


**`snyk.vulnerabilities.disclosure_time`**
:   The time this vulnerability was originally disclosed to the package maintainers.

type: date


**`snyk.vulnerabilities.exploit_maturity`**
:   The Snyk exploit maturity level.

type: keyword


**`snyk.vulnerabilities.id`**
:   The vulnerability reference ID.

type: keyword


**`snyk.vulnerabilities.is_ignored`**
:   If the vulnerability report has been ignored.

type: boolean


**`snyk.vulnerabilities.is_patchable`**
:   If vulnerability is fixable by using a Snyk supplied patch.

type: boolean


**`snyk.vulnerabilities.is_patched`**
:   If the vulnerability has been patched.

type: boolean


**`snyk.vulnerabilities.is_pinnable`**
:   If the vulnerability is fixable by pinning a transitive dependency.

type: boolean


**`snyk.vulnerabilities.is_upgradable`**
:   If the vulnerability fixable by upgrading a dependency.

type: boolean


**`snyk.vulnerabilities.language`**
:   The package’s programming language.

type: keyword


**`snyk.vulnerabilities.package`**
:   The package identifier according to its package manager.

type: keyword


**`snyk.vulnerabilities.package_manager`**
:   The package manager.

type: keyword


**`snyk.vulnerabilities.patches`**
:   Patches required to resolve the issue created by Snyk.

type: flattened


**`snyk.vulnerabilities.priority_score`**
:   The CVS priority score.

type: long


**`snyk.vulnerabilities.publication_time`**
:   The vulnerability publication time.

type: date


**`snyk.vulnerabilities.jira_issue_url`**
:   Link to the related Jira issue.

type: keyword


**`snyk.vulnerabilities.original_severity`**
:   The original severity of the vulnerability.

type: long


**`snyk.vulnerabilities.reachability`**
:   If the vulnerable function from the library is used in the code scanned. Can either be No Info, Potentially reachable and Reachable.

type: keyword


**`snyk.vulnerabilities.title`**
:   The issue title.

type: keyword


**`snyk.vulnerabilities.type`**
:   The issue type. Can be either "license" or "vulnerability".

type: keyword


**`snyk.vulnerabilities.unique_severities_list`**
:   A list of related unique severities.

type: keyword


**`snyk.vulnerabilities.version`**
:   The package version this issue is applicable to.

type: keyword


**`snyk.vulnerabilities.introduced_date`**
:   The date the vulnerability was initially found.

type: date


**`snyk.vulnerabilities.is_fixed`**
:   If the related vulnerability has been resolved.

type: boolean


**`snyk.vulnerabilities.credit`**
:   Reference to the person that original found the vulnerability.

type: keyword


**`snyk.vulnerabilities.semver`**
:   One or more semver ranges this issue is applicable to. The format varies according to package manager.

type: flattened


**`snyk.vulnerabilities.identifiers.alternative`**
:   Additional vulnerability identifiers.

type: keyword


**`snyk.vulnerabilities.identifiers.cwe`**
:   CWE vulnerability identifiers.

type: keyword


