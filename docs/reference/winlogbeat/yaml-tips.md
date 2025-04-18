---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/winlogbeat/current/yaml-tips.html
---

# Avoid YAML formatting problems [yaml-tips]

The configuration file uses [YAML](http://yaml.org/) for its syntax. When you edit the file to modify configuration settings, there are a few things that you should know.


## Use spaces for indentation [_use_spaces_for_indentation]

Indentation is meaningful in YAML. Make sure that you use spaces, rather than tab characters, to indent sections.

In the default configuration files and in all the examples in the documentation, we use 2 spaces per indentation level. We recommend you do the same.


## Look at the default config file for structure [_look_at_the_default_config_file_for_structure]

The best way to understand where to define a configuration option is by looking at the provided sample configuration files. The configuration files contain most of the default configurations that are available for the Beat. To change a setting, simply uncomment the line and change the values.


## Test your config file [_test_your_config_file]

You can test your configuration file to verify that the structure is valid. Simply change to the directory where the binary is installed, and run the Beat in the foreground with the `test config` command specified. For example:

```shell
winlogbeat test config -c winlogbeat.yml
```

You’ll see a message if the Beat finds an error in the file.


## Wrap regular expressions in single quotation marks [_wrap_regular_expressions_in_single_quotation_marks]

If you need to specify a regular expression in a YAML file, it’s a good idea to wrap the regular expression in single quotation marks to work around YAML’s tricky rules for string escaping.

For more information about YAML, see [http://yaml.org/](http://yaml.org/).


## Wrap paths in single quotation marks [wrap-paths-in-quotes]

Windows paths in particular sometimes contain spaces or characters, such as drive letters or triple dots, that may be misinterpreted by the YAML parser.

To avoid this problem, it’s a good idea to wrap paths in single quotation marks.


## Avoid using leading zeros in numeric values [avoid-leading-zeros]

If you use a leading zero (for example, `09`) in a numeric field without wrapping the value in single quotation marks, the value may be interpreted incorrectly by the YAML parser. If the value is a valid octal, it’s converted to an integer. If not, it’s converted to a float.

To prevent unwanted type conversions, avoid using leading zeros in field values, or wrap the values in single quotation marks.


## Avoid accidental template variable resolution [dollar-sign-strings]

The templating engine that allows the config to resolve data from environment variables can result in errors in strings with `$` characters. For example, if a password field contains `$$`, the engine will resolve this to `$`.

To work around this, either use the [Secrets keystore](/reference/winlogbeat/keystore.md) or escape all instances of `$` with `$$`.

