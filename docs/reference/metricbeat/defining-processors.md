---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/defining-processors.html
---

# Define processors [defining-processors]

You can use processors to filter and enhance data before sending it to the configured output. To define a processor, you specify the processor name, an optional condition, and a set of parameters:

```yaml
processors:
  - <processor_name>:
      when:
        <condition>
      <parameters>

  - <processor_name>:
      when:
        <condition>
      <parameters>

...
```

Where:

* `<processor_name>` specifies a [processor](#processors) that performs some kind of action, such as selecting the fields that are exported or adding metadata to the event.
* `<condition>` specifies an optional [condition](#conditions). If the condition is present, then the action is executed only if the condition is fulfilled. If no condition is set, then the action is always executed.
* `<parameters>` is the list of parameters to pass to the processor.

More complex conditional processing can be accomplished by using the if-then-else processor configuration. This allows multiple processors to be executed based on a single condition.

```yaml
processors:
  - if:
      <condition>
    then: <1>
      - <processor_name>:
          <parameters>
      - <processor_name>:
          <parameters>
      ...
    else: <2>
      - <processor_name>:
          <parameters>
      - <processor_name>:
          <parameters>
      ...
```

1. `then` must contain a single processor or a list of one or more processors to execute when the condition evaluates to true.
2. `else` is optional. It can contain a single processor or a list of processors to execute when the conditional evaluate to false.


## Where are processors valid? [where-valid]

Processors are valid:

* At the top-level in the configuration. The processor is applied to all data collected by Metricbeat.
* Under a specific module. The processor is applied to the data collected for that module.

    ```yaml
    - module: <module_name>
      metricsets: ["<metricset_name>"]
      processors:
        - <processor_name>:
            when:
              <condition>
            <parameters>
    ```



## Processors [processors]

The supported processors are:

* [`add_cloud_metadata`](/reference/metricbeat/add-cloud-metadata.md)
* [`add_cloudfoundry_metadata`](/reference/metricbeat/add-cloudfoundry-metadata.md)
* [`add_docker_metadata`](/reference/metricbeat/add-docker-metadata.md)
* [`add_fields`](/reference/metricbeat/add-fields.md)
* [`add_host_metadata`](/reference/metricbeat/add-host-metadata.md)
* [`add_id`](/reference/metricbeat/add-id.md)
* [`add_kubernetes_metadata`](/reference/metricbeat/add-kubernetes-metadata.md)
* [`add_labels`](/reference/metricbeat/add-labels.md)
* [`add_locale`](/reference/metricbeat/add-locale.md)
* [`add_nomad_metadata`](/reference/metricbeat/add-nomad-metadata.md)
* [`add_observer_metadata`](/reference/metricbeat/add-observer-metadata.md)
* [`add_process_metadata`](/reference/metricbeat/add-process-metadata.md)
* [`add_tags`](/reference/metricbeat/add-tags.md)
* [`append`](/reference/metricbeat/append.md)
* [`community_id`](/reference/metricbeat/community-id.md)
* [`convert`](/reference/metricbeat/convert.md)
* [`copy_fields`](/reference/metricbeat/copy-fields.md)
* [`decode_base64_field`](/reference/metricbeat/decode-base64-field.md)
* [`decode_duration`](/reference/metricbeat/decode-duration.md)
* [`decode_json_fields`](/reference/metricbeat/decode-json-fields.md)
* [`decode_xml`](/reference/metricbeat/decode-xml.md)
* [`decode_xml_wineventlog`](/reference/metricbeat/decode-xml-wineventlog.md)
* [`decompress_gzip_field`](/reference/metricbeat/decompress-gzip-field.md)
* [`detect_mime_type`](/reference/metricbeat/detect-mime-type.md)
* [`dissect`](/reference/metricbeat/dissect.md)
* [`dns`](/reference/metricbeat/processor-dns.md)
* [`drop_event`](/reference/metricbeat/drop-event.md)
* [`drop_fields`](/reference/metricbeat/drop-fields.md)
* [`extract_array`](/reference/metricbeat/extract-array.md)
* [`fingerprint`](/reference/metricbeat/fingerprint.md)
* [`include_fields`](/reference/metricbeat/include-fields.md)
* [`move-fields`](/reference/metricbeat/move-fields.md)
* [`rate_limit`](/reference/metricbeat/rate-limit.md)
* [`registered_domain`](/reference/metricbeat/processor-registered-domain.md)
* [`rename`](/reference/metricbeat/rename-fields.md)
* [`replace`](/reference/metricbeat/replace-fields.md)
* [`script`](/reference/metricbeat/processor-script.md)
* [`syslog`](/reference/metricbeat/syslog.md)
* [`translate_ldap_attribute`](/reference/metricbeat/processor-translate-guid.md)
* [`translate_sid`](/reference/metricbeat/processor-translate-sid.md)
* [`truncate_fields`](/reference/metricbeat/truncate-fields.md)
* [`urldecode`](/reference/metricbeat/urldecode.md)


## Conditions [conditions]

Each condition receives a field to compare. You can specify multiple fields under the same condition by using `AND` between the fields (for example, `field1 AND field2`).

For each field, you can specify a simple field name or a nested map, for example `dns.question.name`.

See [Exported fields](/reference/metricbeat/exported-fields.md) for a list of all the fields that are exported by Metricbeat.

The supported conditions are:

* [`equals`](#condition-equals)
* [`contains`](#condition-contains)
* [`regexp`](#condition-regexp)
* [`range`](#condition-range)
* [`network`](#condition-network)
* [`has_fields`](#condition-has_fields)
* [`or`](#condition-or)
* [`and`](#condition-and)
* [`not`](#condition-not)


#### `equals` [condition-equals]

With the `equals` condition, you can compare if a field has a certain value. The condition accepts only an integer or a string value.

For example, the following condition checks if the response code of the HTTP transaction is 200:

```yaml
equals:
  http.response.code: 200
```


#### `contains` [condition-contains]

The `contains` condition checks if a value is part of a field. The field can be a string or an array of strings. The condition accepts only a string value.

For example, the following condition checks if an error is part of the transaction status:

```yaml
contains:
  status: "Specific error"
```


#### `regexp` [condition-regexp]

The `regexp` condition checks the field against a regular expression. The condition accepts only strings.

For example, the following condition checks if the process name starts with `foo`:

```yaml
regexp:
  system.process.name: "^foo.*"
```


#### `range` [condition-range]

The `range` condition checks if the field is in a certain range of values. The condition supports `lt`, `lte`, `gt` and `gte`. The condition accepts only integer, float, or strings that can be converted to either of these as values.

For example, the following condition checks for failed HTTP transactions by comparing the `http.response.code` field with 400.

```yaml
range:
  http.response.code:
    gte: 400
```

This can also be written as:

```yaml
range:
  http.response.code.gte: 400
```

The following condition checks if the CPU usage in percentage has a value between 0.5 and 0.8.

```yaml
range:
  system.cpu.user.pct.gte: 0.5
  system.cpu.user.pct.lt: 0.8
```


#### `network` [condition-network]

The `network` condition checks whether a field’s value falls within a specified IP network range. If multiple fields are provided, each field value must match its corresponding network range. You can specify multiple network ranges for a single field, and a match occurs if any one of the ranges matches. If the field value is an array of IPs, it will match if any of the IPs fall within any of the given ranges. Both IPv4 and IPv6 addresses are supported.

The network range may be specified using CIDR notation, like "192.0.2.0/24" or "2001:db8::/32", or by using one of these named ranges:

* `loopback` - Matches loopback addresses in the range of `127.0.0.0/8` or `::1/128`.
* `unicast` - Matches global unicast addresses defined in RFC 1122, RFC 4632, and RFC 4291 with the exception of the IPv4 broadcast address (`255.255.255.255`). This includes private address ranges.
* `multicast` - Matches multicast addresses.
* `interface_local_multicast` - Matches IPv6 interface-local multicast addresses.
* `link_local_unicast` - Matches link-local unicast addresses.
* `link_local_multicast` - Matches link-local multicast addresses.
* `private` - Matches private address ranges defined in RFC 1918 (IPv4) and RFC 4193 (IPv6).
* `public` - Matches addresses that are not loopback, unspecified, IPv4 broadcast, link local unicast, link local multicast, interface local multicast, or private.
* `unspecified` - Matches unspecified addresses (either the IPv4 address "0.0.0.0" or the IPv6 address "::").

The following condition returns true if the `source.ip` value is within the private address space.

```yaml
network:
  source.ip: private
```

This condition returns true if the `destination.ip` value is within the IPv4 range of `192.168.1.0` - `192.168.1.255`.

```yaml
network:
  destination.ip: '192.168.1.0/24'
```

And this condition returns true when `destination.ip` is within any of the given subnets.

```yaml
network:
  destination.ip: ['192.168.1.0/24', '10.0.0.0/8', loopback]
```


#### `has_fields` [condition-has_fields]

The `has_fields` condition checks if all the given fields exist in the event. The condition accepts a list of string values denoting the field names.

For example, the following condition checks if the `http.response.code` field is present in the event.

```yaml
has_fields: ['http.response.code']
```


#### `or` [condition-or]

The `or` operator receives a list of conditions.

```yaml
or:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

For example, to configure the condition `http.response.code = 304 OR http.response.code = 404`:

```yaml
or:
  - equals:
      http.response.code: 304
  - equals:
      http.response.code: 404
```


#### `and` [condition-and]

The `and` operator receives a list of conditions.

```yaml
and:
  - <condition1>
  - <condition2>
  - <condition3>
  ...
```

For example, to configure the condition `http.response.code = 200 AND status = OK`:

```yaml
and:
  - equals:
      http.response.code: 200
  - equals:
      status: OK
```

To configure a condition like `<condition1> OR <condition2> AND <condition3>`:

```yaml
or:
  - <condition1>
  - and:
    - <condition2>
    - <condition3>
```


#### `not` [condition-not]

The `not` operator receives the condition to negate.

```yaml
not:
  <condition>
```

For example, to configure the condition `NOT status = OK`:

```yaml
not:
  equals:
    status: OK
```


