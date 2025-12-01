---
navigation_title: "translate_ldap_attribute"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/processor-translate-ldap-attribute.html
applies_to:
  stack: ga
---

# Translate LDAP Attribute [processor-translate-ldap-attribute]


The `translate_ldap_attribute` processor translates LDAP attributes into friendlier values. The typical use case is converting an Active Directory Global Unique Identifier (GUID) into a human-readable name (for example the object's `cn`).

Every object on an Active Directory or an LDAP server is issued a GUID. Internal processes refer to their GUID’s rather than the object’s name and these values sometimes appear in logs.

If the search attribute is invalid (malformed) or does not map to any object on the domain the processor returns an error unless `ignore_failure` is set.

The result of this operation is an array of values, given that a single attribute can hold multiple values.

Note: the search attribute is expected to map to a single object. If multiple entries match, only the first entry's mapped attribute values are returned.

```yaml
processors:
  - translate_ldap_attribute:
      field: winlog.event_data.ObjectGuid
      ignore_missing: true
      ignore_failure: true
      # ldap_address: "ldap://ds.example.com:389"  # Optional - resolve via DNS SRV/LOGONSERVER when omitted
      # ldap_base_dn: "dc=example,dc=com"  # Optional - discovered via rootDSE or inferred from server domain
```

The `translate_ldap_attribute` processor has the following configuration settings:

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `field` | yes |  | Source field containing a GUID. |
| `target_field` | no |  | Target field for the mapped attribute value. If not set it will be replaced in place. |
| `ldap_address` | no |  | LDAP server address (eg: `ldap://ds.example.com:389`). If not provided, auto-discovery will be attempted via DNS SRV records and, on Windows, the LOGONSERVER environment variable. |
| `ldap_base_dn` | no |  | LDAP base DN (eg: `dc=example,dc=com`). If not provided, auto-discovery will be attempted via rootDSE query or inferred from the server domain. |
| `ldap_bind_user` | no |  | LDAP user. If both `ldap_bind_user` and `ldap_bind_password` are omitted, the processor will attempt Windows SSPI authentication (on Windows) using the current process user's credentials, or fall back to an unauthenticated bind. |
| `ldap_bind_password` | no |  | LDAP password. If both `ldap_bind_user` and `ldap_bind_password` are omitted, the processor will attempt Windows SSPI authentication (on Windows) using the current process user's credentials, or fall back to an unauthenticated bind. |
| `ldap_search_attribute` | yes | `objectGUID` | LDAP attribute to search by. |
| `ldap_mapped_attribute` | yes | `cn` | LDAP attribute to map to. |
| `ldap_search_time_limit` | no | 30 | LDAP search time limit in seconds. |
| `ldap_ssl`\* | no |  | LDAP TLS/SSL connection settings. See [SSL](/reference/filebeat/configuration-ssl.md). |
| `ad_guid_translation` | no | `auto` | Controls GUID binary conversion for Active Directory attributes. `auto` (default) converts when the LDAP search attribute equals `objectGUID` (case-insensitive). Use `always` to force conversion or `never` to disable it. |
| `ignore_missing` | no | false | Ignore errors when the source field is missing. |
| `ignore_failure` | no | false | Ignore all errors produced by the processor. |

\* Also see [SSL](/reference/filebeat/configuration-ssl.md) for a full description of the `ldap_ssl` options.

## Server auto-discovery

When `ldap_address` is omitted the processor attempts to discover controllers in the following order:

1. DNS SRV lookups for `_ldaps._tcp` (preferred) and `_ldap._tcp`. Results are ordered by SRV priority and weighted randomly (RFC 2782) to avoid overloading a single host.
2. On Windows, the `LOGONSERVER` environment variable. The processor keeps the hostname for TLS validation and may also try the resolved IP as a fallback.

Each candidate server is tried sequentially until one responds. Likewise, if `ldap_base_dn` is not supplied the client queries the server's rootDSE for `defaultNamingContext`/`namingContexts`, and if that fails, infers the DN from the server's domain name (for example `dc=example,dc=com`).

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

