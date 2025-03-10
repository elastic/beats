---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/packetbeat/current/using-environ-vars.html
---

# Use environment variables in the configuration [using-environ-vars]

You can use environment variable references in the config file to set values that need to be configurable during deployment. To do this, use:

`${VAR}`

Where `VAR` is the name of the environment variable.

Each variable reference is replaced at startup by the value of the environment variable. The replacement is case-sensitive and occurs before the YAML file is parsed. References to undefined variables are replaced by empty strings unless you specify a default value or custom error text.

To specify a default value, use:

`${VAR:default_value}`

Where `default_value` is the value to use if the environment variable is undefined.

To specify custom error text, use:

`${VAR:?error_text}`

Where `error_text` is custom text that will be prepended to the error message if the environment variable cannot be expanded.

If you need to use a special character in your configuration file, use `$` to escape the expansion. For example, you can escape `${` or `}` with `$${` or `$}`.

After changing the value of an environment variable, you need to restart Packetbeat to pick up the new value.

::::{note}
You can also specify environment variables when you override a config setting from the command line by using the `-E` option. For example:

`-E name=${NAME}`

::::



## Examples [_examples]

Here are some examples of configurations that use environment variables and what each configuration looks like after replacement:

| Config source | Environment setting | Config after replacement |
| --- | --- | --- |
| `name: ${NAME}` | `export NAME=elastic` | `name: elastic` |
| `name: ${NAME}` | no setting | `name:` |
| `name: ${NAME:beats}` | no setting | `name: beats` |
| `name: ${NAME:beats}` | `export NAME=elastic` | `name: elastic` |
| `name: ${NAME:?You need to set the NAME environment variable}` | no setting | None. Returns an error message that’s prepended with the custom text. |
| `name: ${NAME:?You need to set the NAME environment variable}` | `export NAME=elastic` | `name: elastic` |


## Specify complex objects in environment variables [_specify_complex_objects_in_environment_variables]

You can specify complex objects, such as lists or dictionaries, in environment variables by using a JSON-like syntax.

As with JSON, dictionaries and lists are constructed using `{}` and `[]`. But unlike JSON, the syntax allows for trailing commas and slightly different string quotation rules. Strings can be unquoted, single-quoted, or double-quoted, as a convenience for simple settings and to make it easier for you to mix quotation usage in the shell. Arrays at the top-level do not require brackets (`[]`).

For example, the following environment variable is set to a list:

```yaml
ES_HOSTS="10.45.3.2:9220,10.45.3.1:9230"
```

You can reference this variable in the config file:

```yaml
output.elasticsearch:
  hosts: '${ES_HOSTS}'
```

When Packetbeat loads the config file, it resolves the environment variable and replaces it with the specified list before reading the `hosts` setting.

::::{note}
Do not use double-quotes (`"`) to wrap regular expressions, or the backslash (`\`) will be interpreted as an escape character.
::::


