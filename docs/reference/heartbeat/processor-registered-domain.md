---
navigation_title: "registered_domain"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/heartbeat/current/processor-registered-domain.html
---

# Registered Domain [processor-registered-domain]


The `registered_domain` processor reads a field containing a hostname and then writes the "registered domain" contained in the hostname to the target field. For example, given `www.google.co.uk` the processor would output `google.co.uk`. In other words the "registered domain" is the effective top-level domain (`co.uk`) plus one level (`google`). Optionally, it can store the rest of the domain, the `subdomain` into another target field.

This processor uses the Mozilla Public Suffix list to determine the value.

```yaml
processors:
  - registered_domain:
      field: dns.question.name
      target_field: dns.question.registered_domain
      target_etld_field: dns.question.top_level_domain
      target_subdomain_field: dns.question.sudomain
      ignore_missing: true
      ignore_failure: true
```

The `registered_domain` processor has the following configuration settings:

| Name | Required | Default | Description |  |
| --- | --- | --- | --- | --- |
| `field` | yes |  | Source field containing a fully qualified domain name (FQDN). |  |
| `target_field` | yes |  | Target field for the registered domain value. |  |
| `target_etld_field` | no |  | Target field for the effective top-level domain value. |  |
| `target_subdomain_field` | no |  | Target subdomain field for the subdomain value. |  |
| `ignore_missing` | no | false | Ignore errors when the source field is missing. |  |
| `ignore_failure` | no | false | Ignore all errors produced by the processor. |  |
| `id` | no |  | An identifier for this processor instance. Useful for debugging. |  |

