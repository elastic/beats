---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/filebeat-modules-devguide.html
---

# Creating a New Filebeat Module [filebeat-modules-devguide]

::::{important}
Elastic provides no warranty or support for the code used to generate modules and filesets. The generator is mainly offered as guidance for developers who want to create their own data shippers.
::::


This guide will walk you through creating a new Filebeat module.

All Filebeat modules currently live in the main [Beats](https://github.com/elastic/beats) repository. To clone the repository and build Filebeat (which you will need for testing), please follow the general instructions in [*Contributing to Beats*](./index.md).


## Overview [_overview]

Each Filebeat module is composed of one or more "filesets". We usually create a module for each service that we support (`nginx` for Nginx, `mysql` for Mysql, and so on) and a fileset for each type of log that the service creates. For example, the Nginx module has `access` and `error` filesets. You can contribute a new module (with at least one fileset), or a new fileset for an existing module.

::::{note}
In this guide we use `{{module}}` and `{{fileset}}` as placeholders for the module and fileset names. You need to replace these with the actual names you entered when your created the module and fileset. Only use characters `[a-z]` and, if required, underscores (`_`). No other characters are allowed.
::::



## Creating a new module [_creating_a_new_module]

Run the following command in the `filebeat` folder:

```bash
make create-module MODULE={module}
```

After running the `make create-module` command, you’ll find the module, along with its generated files, under `module/{{module}}`. This directory contains the following files:

```bash
module/{module}
├── module.yml
└── _meta
    └── docs.asciidoc
    └── fields.yml
    └── kibana
```

Let’s look at these files one by one.


### module.yml [_module_yml]

This file contains list of all the dashboards available for the module and used by `export_dashboards.go` script for exporting dashboards. Each dashboard is defined by an id and the name of json file where the dashboard is saved locally. At generation new fileset this file will be automatically updated with "default" dashboard settings for new fileset. Please ensure that this settings are correct.


### _meta/docs.asciidoc [_metadocs_asciidoc]

This file contains module-specific documentation. You should include information about which versions of the service were tested and the variables that are defined in each fileset.


### _meta/fields.yml [_metafields_yml]

The module level `fields.yml` contains descriptions for the module-level fields. Please review and update the title and the descriptions in this file. The title is used as a title in the docs, so it’s best to capitalize it.


### _meta/kibana [_metakibana]

This folder contains the sample Kibana dashboards for this module. To create them, you can build them visually in Kibana and then export them with `export_dashboards`.

The tool will export all of the dashboard dependencies (visualizations, saved searches) automatically.

You can see various ways of using `export_dashboards` at [Exporting New and Modified Beat Dashboards](/extend/export-dashboards.md). The recommended way to export them is to list your dashboards in your module’s `module.yml` file:

```yaml
dashboards:
- id: 69f5ae20-eb02-11e7-8f04-beef1daadb05
  file: mymodule-overview.json
- id: c0a7ce90-cafe-4242-8647-534bb4c21040
  file: mymodule-errors.json
```

Then run `export_dashboards` like this:

```shell
$ cd dev-tools/cmd/dashboards
$ make # if export_dashboard is not built yet
$ ./export_dashboards --yml '../../../filebeat/module/{module}/module.yml'
```

New Filebeat modules might not be compatible with Kibana 5.x. To export dashboards that are compatible with 5.x, run the following command inside the developer virtual environment:

```shell
$ cd filebeat
$ make python-env
$ cd module/{module}/
$ python ../../../dev-tools/export_5x_dashboards.py --regex {module} --dir _meta/kibana/5.x
```

Where the `--regex` parameter should match the dashboard you want to export.

Please note that dashboards exported from Kibana 5.x are not compatible with Kibana 6.x.

You can find more details about the process of creating and exporting the Kibana dashboards by reading [this guide](http://www.elastic.co/guide/en/beats/devguide/master/new-dashboards.md).


## Creating a new fileset [_creating_a_new_fileset]

Run the following command in the `filebeat` folder:

```bash
make create-fileset MODULE={module} FILESET={fileset}
```

After running the `make create-fileset` command, you’ll find the fileset, along with its generated files, under `module/{{module}}/{fileset}`. This directory contains the following files:

```bash
module/{module}/{fileset}
├── manifest.yml
├── config
│   └── {fileset}.yml
├── ingest
│   └── pipeline.json
├── _meta
│   └── fields.yml
│   └── kibana
│       └── default
└── test
```

Let’s look at these files one by one.


### manifest.yml [_manifest_yml]

The `manifest.yml` is the control file for the module, where variables are defined and the other files are referenced. It is a YAML file, but in many places in the file, you can use built-in or defined variables by using the `{{.variable}}` syntax.

The `var` section of the file defines the fileset variables and their default values. The module variables can be referenced in other configuration files, and their value can be overridden at runtime by the Filebeat configuration.

As the fileset creator, you can use any names for the variables you define. Each variable must have a default value. So in it’s simplest form, this is how you can define a new variable:

```yaml
var:
  - name: pipeline
    default: with_plugins
```

Most fileset should have a `paths` variable defined, which sets the default paths where the log files are located:

```yaml
var:
  - name: paths
    default:
      - /example/test.log*
    os.darwin:
      - /usr/local/example/test.log*
      - /example/test.log*
    os.windows:
      - c:/programdata/example/logs/test.log*
```

There’s quite a lot going on in this file, so let’s break it down:

* The name of the variable is `paths` and the default value is an array with one element: `"/example/test.log*"`.
* Note that variable values don’t have to be strings. They can be also numbers, objects, or as shown in this example, arrays.
* We will use the `paths` variable to set the input `paths` setting, so "glob" values can be used here.
* Besides the `default` value, the file defines values for particular operating systems: a default for darwin/OS X/macOS systems and a default for Windows systems. These are introduced via the `os.darwin` and `os.windows` keywords. The values under these keys become the default for the variable, if Filebeat is executed on the respective OS.

Besides the variable definition, the `manifest.yml` file also contains references to the ingest pipeline and input configuration to use (see next sections):

```yaml
ingest_pipeline: ingest/pipeline.json
input: config/testfileset.yml
```

These should point to the respective files from the fileset.

Note that when evaluating the contents of these files, the variables are expanded, which enables you to select one file or the other depending on the value of a variable. For example:

```yaml
ingest_pipeline: ingest/{{.pipeline}}.json
```

This example selects the ingest pipeline file based on the value of the `pipeline` variable. For the `pipeline` variable shown earlier, the path would resolve to `ingest/with_plugins.json` (assuming the variable value isn’t overridden at runtime.)

In 6.6 and later, you can specify multiple ingest pipelines.

```yaml
ingest_pipeline:
  - ingest/main.json
  - ingest/plain_logs.json
  - ingest/json_logs.json
```

When multiple ingest pipelines are specified the first one in the list is considered to be the entry point pipeline.

One reason for using multiple pipelines might be to send all logs harvested by this fileset to the entry point pipeline and have it delegate different parts of the processing to other pipelines. You can read details about setting this up in [the `ingest/*.json` section](#ingest-json-entry-point-pipeline).


### config/*.yml [_config_yml]

The `config/` folder contains template files that generate Filebeat input configurations. The Filebeat inputs are primarily responsible for tailing files, filtering, and multi-line stitching, so that’s what you configure in the template files.

A typical example looks like this:

```yaml
type: log
paths:
{{ range $i, $path := .paths }}
 - {{$path}}
{{ end }}
exclude_files: [".gz$"]
```

You’ll find this example in the template file that gets generated automatically when you run `make create-fileset`. In this example, the `paths` variable is used to construct the `paths` list for the input `paths` option.

Any template files that you add to the `config/` folder need to generate a valid Filebeat input configuration in YAML format. The options accepted by the input configuration are documented in the [Filebeat Inputs](/reference/filebeat/configuration-filebeat-options.md) section of the Filebeat documentation.

The template files use the templating language defined by the [Go standard library](https://golang.org/pkg/text/template/).

Here is another example that also configures multiline stitching:

```yaml
type: log
paths:
{{ range $i, $path := .paths }}
 - {{$path}}
{{ end }}
exclude_files: [".gz$"]
multiline:
  pattern: "^# User@Host: "
  negate: true
  match: after
```

Although you can add multiple configuration files under the `config/` folder, only the file indicated by the `manifest.yml` file will be loaded. You can use variables to dynamically switch between configurations.


### ingest/*.json [_ingest_json]

The `ingest/` folder contains {{es}} [ingest pipeline](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md) configurations. Ingest pipelines are responsible for parsing the log lines and doing other manipulations on the data.

The files in this folder are JSON or YAML documents representing [pipeline definitions](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md). Just like with the `config/` folder, you can define multiple pipelines, but a single one is loaded at runtime based on the information from `manifest.yml`.

The generator creates a JSON object similar to this one:

```json
{
  "description": "Pipeline for parsing {module} {fileset} logs",
  "processors": [
    ],
  "on_failure" : [{
    "set" : {
      "field" : "error.message",
      "value" : "{{ _ingest.on_failure_message }}"
    }
  }]
}
```

Alternatively, you can use YAML formatted pipelines, which uses a simpler syntax:

```yaml
description: "Pipeline for parsing {module} {fileset} logs"
processors:
on_failure:
 - set:
     field: error.message
     value: "{{ _ingest.on_failure_message }}"
```

From here, you would typically add processors to the `processors` array to do the actual parsing. For information about available ingest processors, see the [processor reference documentation](elasticsearch://reference/enrich-processor/index.md). In particular, you will likely find the [grok processor](elasticsearch://reference/enrich-processor/grok-processor.md) to be useful for parsing. Here is an example for parsing the Nginx access logs.

```json
{
  "grok": {
    "field": "message",
    "patterns":[
      "%{IPORHOST:nginx.access.remote_ip} - %{DATA:nginx.access.user_name} \\[%{HTTPDATE:nginx.access.time}\\] \"%{WORD:nginx.access.method} %{DATA:nginx.access.url} HTTP/%{NUMBER:nginx.access.http_version}\" %{NUMBER:nginx.access.response_code} %{NUMBER:nginx.access.body_sent.bytes} \"%{DATA:nginx.access.referrer}\" \"%{DATA:nginx.access.agent}\""
      ],
    "ignore_missing": true
  }
}
```

Note that you should follow the convention of naming of fields prefixed with the module and fileset name: `{{module}}.{fileset}.field`, e.g. `nginx.access.remote_ip`. Also, please review our [Naming Conventions](/extend/event-conventions.md).

$$$ingest-json-entry-point-pipeline$$$
In 6.6 and later, ingest pipelines can use the [`pipeline` processor](docs-content://manage-data/ingest/transform-enrich/ingest-pipelines.md) to delegate parts of the processings to other pipelines.

This can be useful if you want a fileset to ingest the same *logical* information presented in different formats, e.g. csv vs. json versions of the same log files. Imagine an entry point ingest pipeline that detects the format of a log entry and then conditionally delegates further processing of that log entry, depending on the format, to another pipeline.

```json
{
  "processors": [
    {
      "grok": {
        "field": "message",
        "patterns": [
          "^%{CHAR:first_char}"
        ],
        "pattern_definitions": {
          "CHAR": "."
        }
      }
    },
    {
      "pipeline": {
        "if": "ctx.first_char == '{'",
        "name": "{< IngestPipeline "json-log-processing-pipeline" >}" <1>
      }
    },
    {
      "pipeline": {
        "if": "ctx.first_char != '{'",
        "name": "{< IngestPipeline "plain-log-processing-pipeline" >}"
      }
    }
  ]
}
```

1. Use the `IngestPipeline` template function to resolve the name. This function converts the specified name into the fully qualified pipeline ID that is stored in Elasticsearch.


In order for the above pipeline to work, Filebeat must load the entry point pipeline as well as any sub-pipelines into Elasticsearch. You can tell Filebeat to do so by specifying all the necessary pipelines for the fileset in its `manifest.yml` file. The first pipeline in the list is considered to be the entry point pipeline.

```yaml
ingest_pipeline:
  - ingest/main.json
  - ingest/plain_logs.yml
  - ingest/json_logs.json
```

While developing the pipeline definition, we recommend making use of the [Simulate Pipeline API](https://www.elastic.co/docs/api/doc/elasticsearch/operation/operation-ingest-simulate) for testing and quick iteration.

By default Filebeat does not update Ingest pipelines if already loaded. If you want to force updating your pipeline during development, use `./filebeat setup --pipelines` command. This uploads pipelines even if they are already available on the node.


### _meta/fields.yml [_metafields_yml_2]

The `fields.yml` file contains the top-level structure for the fields in your fileset. It is used as the source of truth for:

* the generated Elasticsearch mapping template
* the generated Kibana index pattern
* the generated documentation for the exported fields

Besides the `fields.yml` file in the fileset, there is also a `fields.yml` file at the module level, placed under `module/{{module}}/_meta/fields.yml`, which should contain the fields defined at the module level, and the description of the module itself. In most cases, you should add the fields at the fileset level.

After `pipeline.json` is created, it is possible to generate a base `field.yml`.

```bash
make create-fields MODULE={module} FILESET={fileset}
```

Please, always check the generated file and make sure the fields are correct. You must add field documentation manually.

If the fields are correct, it is time to generate documentation, configuration and Kibana index patterns.

```bash
make update
```


### test [_test]

In the `test/` directory, you should place sample log files generated by the service. We have integration tests, automatically executed by CI, that will run Filebeat on each of the log files under the `test/` folder and check that there are no parsing errors and that all fields are documented.

In addition, assuming you have a `test.log` file, you can add a `test.log-expected.json` file in the same directory that contains the expected documents as they are found via an Elasticsearch search. In this case, the integration tests will automatically check that the result is the same on each run.

In order to test the filesets with the sample logs and/or generate the expected output one should run the tests locally for a specific module, using the following procedure under Filebeat directory:

1. Start an Elasticsearch instance locally. For example, using Docker:

    ```bash
    docker run \
      --name elasticsearch \
      -p 9200:9200 -p 9300:9300 \
      -e "xpack.security.http.ssl.enabled=false"  -e "ELASTIC_PASSWORD=changeme" \
      -e "discovery.type=single-node" \
      --pull always --rm --detach \
      docker.elastic.co/elasticsearch/elasticsearch:master-SNAPSHOT
    ```

2. Create an "admin" user on that Elasticsearch instance:

    ```bash
    curl -u elastic:changeme \
      http://localhost:9200/_security/user/admin \
      -X POST -H 'Content-Type: application/json' \
      -d '{"password": "changeme", "roles": ["superuser"]}'
    ```

3. Create the testing binary: `make filebeat.test`
4. Update fields yaml: `make update`
5. Create python env: `make python-env`
6. Source python env: `source ./build/python-env/bin/activate`
7. Run a test, for example to check nginx access log parsing:

    ```bash
    INTEGRATION_TESTS=1 BEAT_STRICT_PERMS=false ES_PASS=changeme \
    TESTING_FILEBEAT_MODULES=nginx \
    pytest tests/system/test_modules.py -v --full-trace
    ```

8. Add and remove option env vars as required. Here are some useful ones:

    * `TESTING_FILEBEAT_ALLOW_OLDER`: if set to 1, allow connecting older versions of Elasticsearch
    * `TESTING_FILEBEAT_MODULES`: comma separated list of modules to test.
    * `TESTING_FILEBEAT_FILESETS`: comma separated list of filesets to test.
    * `TESTING_FILEBEAT_FILEPATTERN`: glob pattern for log files within the fileset to test.
    * `GENERATE`: if set to 1, the expected documents will be generated.


The filebeat logs are writen to the `build` directory. It may be useful to tail them in another terminal using `tail -F build/system-tests/run/test_modules.Test.*/output.log`.

For example if there’s a syntax error in an ingest pipeline, the test will probably just hang. The filebeat log output will contain the error message from elasticsearch.

