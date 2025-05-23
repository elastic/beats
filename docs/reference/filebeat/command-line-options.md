---
navigation_title: "Command reference"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/command-line-options.html
---

# Filebeat command reference [command-line-options]


Filebeat provides a command-line interface for starting Filebeat and performing common tasks, like testing configuration files and loading dashboards.

The command-line also supports [global flags](#global-flags) for controlling global behaviors.

::::{tip}
Use `sudo` to run the following commands if:

* the config file is owned by `root`, or
* Filebeat is configured to capture data that requires `root` access

::::


Some of the features described here require an Elastic license. For more information, see [https://www.elastic.co/subscriptions](https://www.elastic.co/subscriptions) and [License Management](docs-content://deploy-manage/license/manage-your-license-in-self-managed-cluster.md).

| Commands |  |
| --- | --- |
| [`export`](#export-command) | Exports the configuration, index template, ILM policy, or a dashboard to stdout. |
| [`help`](#help-command) | Shows help for any command. |
| [`keystore`](#keystore-command) | Manages the [secrets keystore](/reference/filebeat/keystore.md). |
| [`modules`](#modules-command) | Manages configured modules. |
| [`run`](#run-command) | Runs Filebeat. This command is used by default if you start Filebeat without specifying a command. |
| [`setup`](#setup-command) | Sets up the initial environment, including the index template, ILM policy and write alias, {{kib}} dashboards (when available), and machine learning jobs (when available). |
| [`test`](#test-command) | Tests the configuration. |
| [`version`](#version-command) | Shows information about the current version. |

Also see [Global flags](#global-flags).

## `export` command [export-command]

Exports the configuration, index template, ILM policy, or a dashboard to stdout. You can use this command to quickly view your configuration, see the contents of the index template and the ILM policy, or export a dashboard from {{kib}}.

**SYNOPSIS**

```sh
filebeat export SUBCOMMAND [FLAGS]
```

**SUBCOMMANDS**

**`config`**
:   Exports the current configuration to stdout. If you use the `-c` flag, this command exports the configuration that’s defined in the specified file.

$$$dashboard-subcommand$$$**`dashboard`**
:   Exports a dashboard. You can use this option to store a dashboard on disk in a module and load it automatically. For example, to export the dashboard to a JSON file, run:

    ```shell
    filebeat export dashboard --id="DASHBOARD_ID" > dashboard.json
    ```

    To find the `DASHBOARD_ID`, look at the URL for the dashboard in {{kib}}. By default, `export dashboard` writes the dashboard to stdout. The example shows how to write the dashboard to a JSON file so that you can import it later. The JSON file will contain the dashboard with all visualizations and searches. You must load the index pattern separately for Filebeat.

    To load the dashboard, copy the generated `dashboard.json` file into the `kibana/6/dashboard` directory of Filebeat, and run `filebeat setup --dashboards` to import the dashboard.

    If {{kib}} is not running on `localhost:5061`, you must also adjust the Filebeat configuration under `setup.kibana`.


$$$template-subcommand$$$**`template`**
:   Exports the index template to stdout. You can specify the `--es.version` flag to further define what gets exported. Furthermore you can export the template to a file instead of `stdout` by defining a directory via `--dir`.

$$$ilm-policy-subcommand$$$

**`ilm-policy`**
:   Exports the index lifecycle management policy to stdout. You can specify the `--es.version` and a `--dir` to which the policy should be exported as a file rather than exporting to `stdout`.

**FLAGS**

**`--es.version VERSION`**
:   When used with [`template`](#template-subcommand), exports an index template that is compatible with the specified version.  When used with [`ilm-policy`](#ilm-policy-subcommand), exports the ILM policy if the specified ES version is enabled for ILM.

**`-h, --help`**
:   Shows help for the `export` command.

**`--dir DIRNAME`**
:   Define a directory to which the template, pipelines, and ILM policy should be exported to as files instead of printing them to `stdout`.

**`--id DASHBOARD_ID`**
:   When used with [`dashboard`](#dashboard-subcommand), specifies the dashboard ID.

Also see [Global flags](#global-flags).

**EXAMPLES**

```sh subs=true
filebeat export config
filebeat export template --es.version {{stack-version}}
filebeat export dashboard --id="a7b35890-8baa-11e8-9676-ef67484126fb" > dashboard.json
```


## `help` command [help-command]

Shows help for any command. If no command is specified, shows help for the `run` command.

**SYNOPSIS**

```sh
filebeat help COMMAND_NAME [FLAGS]
```

**`COMMAND_NAME`**
:   Specifies the name of the command to show help for.

**FLAGS**

**`-h, --help`**
:   Shows help for the `help` command.

Also see [Global flags](#global-flags).

**EXAMPLE**

```sh
filebeat help export
```


## `keystore` command [keystore-command]

Manages the [secrets keystore](/reference/filebeat/keystore.md).

**SYNOPSIS**

```sh
filebeat keystore SUBCOMMAND [FLAGS]
```

**SUBCOMMANDS**

**`add KEY`**
:   Adds the specified key to the keystore. Use the `--force` flag to overwrite an existing key. Use the `--stdin` flag to pass the value through `stdin`.

**`create`**
:   Creates a keystore to hold secrets. Use the `--force` flag to overwrite the existing keystore.

**`list`**
:   Lists the keys in the keystore.

**`remove KEY`**
:   Removes the specified key from the keystore.

**FLAGS**

**`--force`**
:   Valid with the `add` and `create` subcommands. When used with `add`, overwrites the specified key. When used with `create`, overwrites the keystore.

**`--stdin`**
:   When used with `add`, uses the stdin as the source of the key’s value.

**`-h, --help`**
:   Shows help for the `keystore` command.

Also see [Global flags](#global-flags).

**EXAMPLES**

```sh
filebeat keystore create
filebeat keystore add ES_PWD
filebeat keystore remove ES_PWD
filebeat keystore list
```

See [Secrets keystore](/reference/filebeat/keystore.md) for more examples.


## `modules` command [modules-command]

Manages configured modules. You can use this command to enable and disable specific module configurations defined in the `modules.d` directory. The changes you make with this command are persisted and used for subsequent runs of Filebeat.

To see which modules are enabled and disabled, run the `list` subcommand.

**SYNOPSIS**

```sh
filebeat modules SUBCOMMAND [FLAGS]
```

**SUBCOMMANDS**

**`disable MODULE_LIST`**
:   Disables the modules specified in the space-separated list.

**`enable MODULE_LIST`**
:   Enables the modules specified in the space-separated list.

**`list`**
:   Lists the modules that are currently enabled and disabled.

**FLAGS**

**`-h, --help`**
:   Shows help for the `modules` command.

Also see [Global flags](#global-flags).

**EXAMPLES**

```sh
filebeat modules list
filebeat modules enable apache2 auditd mysql
```


## `run` command [run-command]

Runs Filebeat. This command is used by default if you start Filebeat without specifying a command.

**SYNOPSIS**

```sh
filebeat run [FLAGS]
```

Or:

```sh
filebeat [FLAGS]
```

**FLAGS**

**`-N, --N`**
:   Disables publishing for testing purposes. This option disables all outputs except the [File output](/reference/filebeat/file-output.md).

**`--cpuprofile FILE`**
:   Writes CPU profile data to the specified file. This option is useful for troubleshooting Filebeat.

**`-h, --help`**
:   Shows help for the `run` command.

**`--httpprof [HOST]:PORT`**
:   Starts an http server for profiling. This option is useful for troubleshooting and profiling Filebeat.

**`--memprofile FILE`**
:   Writes memory profile data to the specified output file. This option is useful for troubleshooting Filebeat.

**`--modules MODULE_LIST`**
:   Specifies a comma-separated list of modules to run. For example:

    ```sh
    filebeat run --modules nginx,mysql,system
    ```

    Rather than specifying the list of modules every time you run Filebeat, you can use the [`modules`](#modules-command) command to enable and disable specific modules. Then when you run Filebeat, it will run any modules that are enabled.


**`--once`**
:   When the `--once` flag is used, Filebeat starts all configured harvesters and inputs, and runs each input until the harvesters are closed. If you set the `--once` flag, you should also set `close_eof` so the harvester is closed when the end of the file is reached. By default harvesters are closed after `close_inactive` is reached.

    The `--once` option is not currently supported with the [`filestream`](/reference/filebeat/filebeat-input-filestream.md) input type.


**`--system.hostfs MOUNT_POINT`**
:   Specifies the mount point of the host’s filesystem for use in monitoring a host. This flag is depricated, and an alternate hostfs should be specified via the `hostfs` module config value.

Also see [Global flags](#global-flags).

**EXAMPLE**

```sh
filebeat run -e
```

Or:

```sh
filebeat -e
```


## `setup` command [setup-command]

Sets up the initial environment, including the index template, ILM policy and write alias, {{kib}} dashboards (when available), and machine learning jobs (when available)

* The index template ensures that fields are mapped correctly in Elasticsearch. If index lifecycle management is enabled it also ensures that the defined ILM policy and write alias are connected to the indices matching the index template. The ILM policy takes care of the lifecycle of an index, when to do a rollover, when to move an index from the hot phase to the next phase, etc.
* The {{kib}} dashboards make it easier for you to visualize Filebeat data in {{kib}}.
* The machine learning jobs contain the configuration information and metadata necessary to analyze data for anomalies.

This command sets up the environment without actually running Filebeat and ingesting data. Specify optional flags to set up a subset of assets.

**SYNOPSIS**

```sh
filebeat setup [FLAGS]
```

**FLAGS**

**`--dashboards`**
:   Sets up the {{kib}} dashboards (when available). This option loads the dashboards from the Filebeat package. For more options, such as loading customized dashboards, see [Importing Existing Beat Dashboards](../../extend/import-dashboards.md).

**`-h, --help`**
:   Shows help for the `setup` command.

**`--modules MODULE_LIST`**
:   Specifies a comma-separated list of modules. Use this flag to avoid errors when there are no modules defined in the `filebeat.yml` file.

**`--pipelines`**
:   Sets up ingest pipelines for configured filesets. Filebeat looks for enabled modules in the `filebeat.yml` file. If you used the [`modules`](#modules-command) command to enable modules in the `modules.d` directory, also specify the `--modules` flag.

**`--enable-all-filesets`**
:   Enables all modules and filesets. This is useful with `--pipelines` if you want to load all ingest pipelines. Without this option you would have to list every module with the [`modules`](#modules-command) command and enable every fileset within the module with a `-M` option, to load all of the ingest pipelines.

**`--force-enable-module-filesets`**
:   Enables all filesets in the enabled modules. This is useful with `--pipelines` if you want to load ingest pipelines. Without this option you enable every fileset within the module with a `-M` option, to load the ingest pipelines.

**`--index-management`**
:   Sets up components related to Elasticsearch index management including template, ILM policy, and write alias (if supported and configured).

Also see [Global flags](#global-flags).

**EXAMPLES**

```sh
filebeat setup --dashboards
filebeat setup --pipelines
filebeat setup --pipelines --modules system,nginx,mysql <1>
filebeat setup --index-management
```

1. If you used the [`modules`](#modules-command) command to enable modules in the `modules.d` directory, also specify the `--modules` flag to indicate which modules to load pipelines for.



## `test` command [test-command]

Tests the configuration.

**SYNOPSIS**

```sh
filebeat test SUBCOMMAND [FLAGS]
```

**SUBCOMMANDS**

**`config`**
:   Tests the configuration settings.

**`output`**
:   Tests that Filebeat can connect to the output by using the current settings.

**FLAGS**

**`-h, --help`**
:   Shows help for the `test` command.

Also see [Global flags](#global-flags).

**EXAMPLE**

```sh
filebeat test config
```


## `version` command [version-command]

Shows information about the current version.

**SYNOPSIS**

```sh
filebeat version [FLAGS]
```

**FLAGS**

**`-h, --help`**
:   Shows help for the `version` command.

Also see [Global flags](#global-flags).

**EXAMPLE**

```sh
filebeat version
```


## Global flags [global-flags]

These global flags are available whenever you run Filebeat.

**`-E, --E "SETTING_NAME=VALUE"`**
:   Overrides a specific configuration setting. You can specify multiple overrides. For example:

    ```sh
    filebeat -E "name=mybeat" -E "output.elasticsearch.hosts=['http://myhost:9200']"
    ```

    This setting is applied to the currently running Filebeat process. The Filebeat configuration file is not changed.


**`-M, --M "VAR_NAME=VALUE"`**
:   Overrides the default configuration for a Filebeat module. You can specify multiple variable overrides. For example:

    ```sh
    filebeat --modules=nginx -M "nginx.access.var.paths=['/var/log/nginx/access.log*']" -M "nginx.access.var.pipeline=no_plugins"
    ```


**`-c, --c FILE`**
:   Specifies the configuration file to use for Filebeat. The file you specify here is relative to `path.config`. If the `-c` flag is not specified, the default config file, `filebeat.yml`, is used.

**`-d, --d SELECTORS`**
:   Enables debugging for the specified selectors. For the selectors, you can specify a comma-separated list of components, or you can use `-d "*"` to enable debugging for all components. For example, `-d "publisher"` displays all the publisher-related messages.

**`-e, --e`**
:   Logs to stderr and disables syslog/file output.

**`--environment`**
:   For logging purposes, specifies the environment that Filebeat is running in. This setting is used to select a default log output when no log output is configured. Supported values are: `systemd`, `container`, `macos_service`, and `windows_service`. If `systemd` or `container` is specified, Filebeat will log to stdout and stderr by default.

**`--path.config`**
:   Sets the path for configuration files. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details.

**`--path.data`**
:   Sets the path for data files. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details.

**`--path.home`**
:   Sets the path for miscellaneous files. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details.

**`--path.logs`**
:   Sets the path for log files. See the [Directory layout](/reference/filebeat/directory-layout.md) section for details.

**`--strict.perms`**
:   Sets strict permission checking on configuration files. The default is `--strict.perms=true`. See [Config file ownership and permissions](/reference/libbeat/config-file-permissions.md) for more information.

**`-v, --v`**
:   Logs INFO-level messages.


