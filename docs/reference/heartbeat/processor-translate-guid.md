---
navigation_title: "translate_ldap_attribute"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/processor-translate-guid.html
applies_to:
  stack: ga
  serverless: ga
---

# Translate LDAP Attribute [processor-translate-ldap-attribute]

The `translate_ldap_attribute` processor translates LDAP attributes into friendlier values. The typical use case is converting an Active Directory Global Unique Identifier (GUID) into a human-readable name (for example the object's `cn`).

Every object on an Active Directory or an LDAP server is issued a GUID. Internal processes refer to their GUID’s rather than the object's name and these values sometimes appear in logs.

If the search attribute is invalid (malformed) or does not map to any object on the domain the processor returns an error unless `ignore_failure` is set.

The result of this operation is an array of values, given that a single attribute can hold multiple values.

The search attribute is expected to map to a single object. If multiple entries match, only the first entry's mapped attribute values are returned.

```yaml
processors:
  - translate_ldap_attribute:
      field: winlog.event_data.ObjectGuid
      ignore_missing: true
      ignore_failure: true
      # ldap_domain: "example.com"  # Optional - override the OS-discovered domain used for SRV/LOGONSERVER hints
      # ldap_address: "ldap://ds.example.com:389"  # Optional - Beats discovers controllers when omitted
      # ldap_base_dn: "dc=example,dc=com"  # Optional - otherwise rootDSE and hostname inference are used
```

The `translate_ldap_attribute` processor has the following configuration settings:

| Name | Required | Default | Description |
| --- | --- | --- | --- |
| `field` | yes |  | Source field containing a GUID. |
| `target_field` | no |  | Target field for the mapped attribute value. If not set it will be replaced in place. |
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
| `ignore_missing` | no | false | Ignore errors when the source field is missing. |
| `ignore_failure` | no | false | Ignore all errors produced by the processor. |

## Authentication flow

Beats attempts LDAP authentication in the following order:

1. Simple bind using `ldap_bind_user` and `ldap_bind_password` when both are supplied.
2. Automatic bind when both values are empty. On Windows Beats creates an SSPI (Kerberos/NTLM) client for the SPN `ldap/<hostname derived from ldap_address>`. Other platforms do not yet implement automatic authentication and fall back to unauthenticated bind immediately.
3. If automatic authentication is unavailable or fails, Beats issues an unauthenticated bind. When `ldap_bind_user` is set without a password that identity is used; otherwise Beats binds anonymously.

Always prefer specifying `ldap_address` as an FQDN (for example `ldap://dc1.example.com:389`) so the SPN built for SSPI matches the controller's service principal and TLS certificates.

## Windows SSPI requirements

SSPI (Security Support Provider Interface) enables passwordless authentication to Active Directory by using the credentials of the Windows process identity. However, SSPI only works when the Beat runs under an account that has valid Kerberos credentials.

| Account Type | SSPI Works | Reason |
| --- | --- | --- |
| Local System (`NT AUTHORITY\SYSTEM`) | ✅ Yes | Uses the computer account's domain credentials on domain-joined machines. |
| Network Service (`NT AUTHORITY\NETWORK SERVICE`) | ✅ Yes | Uses the computer account's domain credentials on domain-joined machines. |
| Domain user account | ✅ Yes | Uses the domain user's Kerberos credentials. |
| Group Managed Service Account (gMSA) | ✅ Yes | Domain account with AD-managed password. Ideal for services. |
| Local user account | ❌ **No** | Local accounts have no Active Directory identity and cannot obtain Kerberos tickets. |

::::{important}
When running Beats as a local user account, SSPI authentication will not work. You must either:

* Provide explicit credentials using `ldap_bind_user` and `ldap_bind_password`.
* Configure the LDAP server to allow unauthenticated/anonymous queries for the required attributes.
* Run the Beat under a domain account, gMSA, or Local System instead of a local user.
::::

For domain accounts or gMSA, ensure the account has read permissions on the LDAP objects being queried.

## Server auto-discovery

When `ldap_address` is omitted Beats resolves controllers dynamically:

1. **Domain discovery.** Beats determines the DNS domain from `ldap_domain` (if set) or OS metadata. Windows checks `USERDNSDOMAIN`, `GetComputerNameEx`, the TCP/IP and Kerberos registry keys, and the machine's FQDN. Linux/macOS read `/etc/resolv.conf`, `/etc/krb5.conf`, and the hostname suffix. If no domain is available SRV lookups are skipped.
2. **DNS SRV queries.** When a domain is known Beats queries `_ldaps._tcp.<domain>` first and `_ldap._tcp.<domain>` second using the system resolver. Results are sorted by priority/weight per RFC 2782 and converted to `ldaps://host:port` or `ldap://host:port` URLs.
3. **Windows LOGONSERVER fallback.** If SRV queries return no controllers or no domain was discovered, Beats reads the `LOGONSERVER` environment variable. When a domain is known the NetBIOS name is combined with it to build an FQDN so TLS validation and SSPI SPNs remain valid.

Each candidate address is attempted in order (LDAPS before LDAP) until a connection and bind succeed.

When `ldap_base_dn` is empty the client queries the controller's rootDSE for `defaultNamingContext` or the first non-system `namingContexts` entry. If neither is present Beats cannot continue and you must provide `ldap_base_dn` explicitly.

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
