---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/libbeat/current/config-file-format-type.html
---

# Config file data types [config-file-format-type]

Values of configuration settings are interpreted as required by beats. If a value can not be correctly interpreted as the required type - for example a string is given when a number is required - the beat will fail to start up.

## Boolean [_boolean]

Boolean values can be either `true` or `false`. Alternative names for `true` are `yes` and `on`. Instead of `false` the values `no` and `off` can be used.

```yaml
enabled: true
disabled: false
```


## Number [_number]

Number values require you to enter the number to use without using single or double quotes. Some settings only support a restricted number range though.

```yaml
integer: 123
negative: -1
float: 5.4
```


## String [_string]

In YAML[[http://www.yaml.org](http://www.yaml.org)], multiple styles of string definitions are supported: double-quoted, single-quoted, unquoted.

The double-quoted style is specified by surrounding the string with `"`. This style provides support for escaping unprintable characters using `\`, but comes at the cost of having to escape `\` and `"` characters.

The single-quoted style is specified by surrounding the string with `'`. This style supports no escaping (use `''` to quote a single quote). Only printable characters can be used when using this form.

Unquoted style requires no quotes, but does not support any escaping plus care needs to be taken to not use any symbol that has a special meaning in YAML.

Note: Single-quoted style is recommended when defining regular expressions, event format strings, windows file paths, or non-alphabetical symbolic characters.


## Duration [_duration]

Durations require a numeric value with optional fraction and required unit. Valid time units are `ns`, `us`, `ms`, `s`, `m`, `h`. Sometimes features based on durations can be disabled by using zero or negative durations.

```yaml
duration1: 2.5s
duration2: 6h
duration_disabled: -1s
```


## Regular expression [_regular_expression]

Regular expressions are special strings getting compiled into regular expressions at load time.

As regular expressions and YAML use `\` for escaping characters in strings, it’s highly recommended to use single quoted strings when defining regular expressions. When single quoted strings are used, `\` character is not interpreted by YAML parser as escape symbol.


## Format String (sprintf) [_format_string_sprintf]

Format strings enable you to refer to event field values creating a string based on the current event being processed. Variable expansions are enclosed in expansion braces `%{<accessor>:default value}`. Event fields are accessed using field references `[fieldname]`. Optional default values can be specified in case the field name is missing from the event.

You can also format time stored in the `@timestamp` field using the `+FORMAT` syntax where FORMAT is a valid [time format](https://godoc.org/github.com/elastic/beats/libbeat/common/dtfmt).

```yaml
constant-format-string: 'constant string'
field-format-string: '%{[fieldname]} string'
format-string-with-date: '%{[fieldname]}-%{+yyyy.MM.dd}'
```


