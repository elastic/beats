---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-file-format.html
---

# Config file format [config-file-format]

Beats config files are based on [YAML](http://www.yaml.org), a file format that is easier to read and write than other common data formats like XML or JSON. Config files must be encoded in UTF-8.

In beats all YAML files start with a dictionary, an unordered collection of name/value pairs. In addition to dictionaries, YAML also supports lists, numbers, strings, and many other data types. All members of the same list or dictionary must have the same indentation level.

Dictionaries are represented by simple `key: value` pairs all having the same indentation level. The colon after `key` must be followed by a space.

```yaml
name: John Doe
age: 34
country: Canada
```

Lists are introduced by dashes `- `. All list members will be lines beginning with `- ` at the same indentation level.

```yaml
- Red
- Green
- Blue
```

Lists and dictionaries are used in beats to build structured configurations.

```yaml
filebeat:
  inputs:
    - type: log
      paths:
        - /var/log/*.log
      multiline:
        pattern: '^['
        match: after
```

Lists and dictionaries can also be represented in abbreviated form. Abbreviated form is somewhat similar to JSON using `{}` for dictionaries and `[]` for lists:

```yaml
person: {name: "John Doe", age: 34, country: "Canada"}
colors: ["Red", "Green", "Blue"]
```

The following topics provide more detail to help you understand and work with config files in YAML:

* [Namespacing](/reference/libbeat/config-file-format-namespacing.md)
* [Config file data types](/reference/libbeat/config-file-format-type.md)
* [Environment variables](/reference/libbeat/config-file-format-env-vars.md)
* [Reference variables](/reference/libbeat/config-gile-format-refs.md)
* [Config file ownership and permissions](/reference/libbeat/config-file-permissions.md)
* [Command line arguments](/reference/libbeat/config-file-format-cli.md)
* [YAML tips and gotchas](/reference/libbeat/config-file-format-tips.md)








