---
navigation_title: "translate_ldap_attribute"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/processor-translate-guid.html
applies_to:
  stack: ga
---

# Translate GUID [processor-translate-guid]


The `translate_ldap_attribute` processor translates an LDAP attributes between eachother. It is typically used to translate AD Global Unique Identifiers (GUID) into their common names.

Every object on an Active Directory or an LDAP server is issued a GUID. Internal processes refer to their GUID’s rather than the object’s name and these values sometimes appear in logs.

If the search attribute is invalid (malformed) or does not map to any object on the domain then this will result in the processor returning an error unless `ignore_failure` is set.

The result of this operation is an array of values, given that a single attribute can hold multiple values.

Note: the search attribute is expected to map to a single object. If it doesn’t, no error will be returned, but only results of the first entry will be added to the event.

```yaml
processors:
  - translate_ldap_attribute:
      field: winlog.event_data.ObjectGuid
      ldap_address: "ldap://"
      ldap_base_dn: "dc=example,dc=com"
      ignore_missing: true
      ignore_failure: true
```

The `translate_ldap_attribute` processor has the following configuration settings:

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `field` | yes |  | Source field containing a GUID. |
| `target_field` | no |  | Target field for the mapped attribute value. If not set it will be replaced in place. |
<<<<<<< HEAD
| `ldap_address` | yes |  | LDAP server address. eg: `ldap://ds.example.com:389` |
| `ldap_base_dn` | yes |  | LDAP base DN. eg: `dc=example,dc=com` |
| `ldap_bind_user` | no |  | LDAP user. |
| `ldap_bind_password` | no |  | LDAP password. |
| `ldap_search_attribute` | yes | `objectGUID` | LDAP attribute to search by. |
| `ldap_mapped_attribute` | yes | `cn` | LDAP attribute to map to. |
| `ldap_search_time_limit` | no | 30 | LDAP search time limit in seconds. |
| `ldap_ssl`\* | no | 30 | LDAP TLS/SSL connection settings. |
=======
| `ldap_domain` | no |  | {applies_to}`stack: ga 9.2.4` DNS domain name (for example, `example.com`) used for DNS SRV discovery and to construct FQDNs from `LOGONSERVER`. When omitted Beats inspects OS metadata to infer the domain (Windows: `USERDNSDOMAIN`, `GetComputerNameEx`, TCP/IP + Kerberos registry keys, hostname; Linux/macOS: `/etc/resolv.conf`, `/etc/krb5.conf`, hostname). |
| `ldap_address` | {applies_to}`stack: ga 9.2.4+!` no<br><br>{applies_to}`stack: ga 9.0.0!-9.2.3!` yes |  | LDAP server address (for example, `ldap://ds.example.com:389`). When omitted Beats auto-discovers controllers by querying `_ldaps._tcp.<domain>` first, `_ldap._tcp.<domain>` second, and finally the Windows `LOGONSERVER` variable if available. Candidates are tried in order until one succeeds. |
| `ldap_base_dn` | {applies_to}`stack: ga 9.2.4+!` no<br><br>{applies_to}`stack: ga 9.0.0!-9.2.3!` yes |  | LDAP base DN (for example, `dc=example,dc=com`). When omitted Beats queries the server's rootDSE for `defaultNamingContext`/`namingContexts`. If the controller does not expose those attributes, client initialization fails and you must configure the value manually. |
| `ldap_bind_user` | no |  | LDAP DN/UPN for simple bind. When provided with `ldap_bind_password` Beats performs a standard bind. When set without a password Beats issues an unauthenticated bind using this identity (useful for servers that expect a bind DN even for anonymous operations). |
| `ldap_bind_password` | no |  | LDAP password for simple bind. When both the username and password are omitted Beats attempts automatic authentication: on Windows it first tries SSPI with the Beat's service or user identity using the SPN `ldap/<hostname derived from ldap_address>` and falls back to an unauthenticated bind if that fails. Non-Windows platforms immediately use an unauthenticated bind. |
| `ldap_search_attribute` | yes | `objectGUID` | LDAP attribute to search by. |
| `ldap_mapped_attribute` | yes | `cn` | LDAP attribute to map to. |
| `ldap_search_time_limit` | no | 30 | LDAP search time limit in seconds. |
| `ldap_ssl` | no | {applies_to}`stack: ga 9.2.4+!` no default<br><br>{applies_to}`stack: ga 9.0.0!-9.2.3!` `30` | LDAP TLS/SSL connection settings. Refer to [SSL](/reference/heartbeat/configuration-ssl.md). |
| `ad_guid_translation` | no | `auto` | {applies_to}`stack: ga 9.2.4` Controls GUID binary conversion for Active Directory attributes. `auto` (default) converts when the LDAP search attribute equals `objectGUID` (case-insensitive). Use `always` to force conversion or `never` to disable it. |
>>>>>>> 5e37672f2 (Refine `applies_to` syntax (#48333))
| `ignore_missing` | no | false | Ignore errors when the source field is missing. |
| `ignore_failure` | no | false | Ignore all errors produced by the processor. |

\* Also see [SSL](/reference/heartbeat/configuration-ssl.md) for a full description of the `ldap_ssl` options.

If the searches are slow or you expect a high amount of different key attributes to be found, consider using a cache processor to speed processing:

```yaml
processors:
  - cache:
      backend:
        memory:
          id: ldapguids
      get:
        key_field: winlog.event_data.ObjectGuid
        target_field: winlog.common_name
      ignore_missing: true
  - if:
      not:
        - has_fields: winlog.common_name
    then:
      - translate_ldap_attribute:
          field: winlog.event_data.ObjectGuid
          target_field: winlog.common_name
          ldap_address: "ldap://"
          ldap_base_dn: "dc=example,dc=com"
      - cache:
          backend:
            memory:
              id: ldapguids
            capacity: 10000
          put:
            key_field: winlog.event_data.ObjectGuid
            value_field: winlog.common_name
```

