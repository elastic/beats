---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/creating-metricbeat-module.html
---

# Creating a Metricbeat Module [creating-metricbeat-module]

Metricbeat modules are used to group multiple metricsets together and to implement shared functionality of the metricsets. In most cases, no implementation of the module is needed and the default module implementation is automatically picked.

It’s important to complete the configuration and documentation files for a module. When you create a new metricset by running `make create-metricset`, default versions of these files are generated in the `_meta` directory.


## Module Files [_module_files]

* `config.yml` and `config.reference.yml`
* `docs.asciidoc`
* `fields.yml`

After updating any of these files, make sure you run `make update` in your beat directory so all generated files are updated.


### config.yml and config.reference.yml [_config_yml_and_config_reference_yml]

The `config.yml` file contains the basic configuration options and looks like this:

```yaml
- module: {module}
  metricsets: ["{metricset}"]
  enabled: false
  period: 10s
  hosts: ["localhost"]
```

It contains the module name, your metricset, and the default period. If you have multiple metricsets in your module, make sure that you extend the metricset array:

```yaml
  metricsets: ["{metricset1}", "{metricset2}"]
```

The `full.config.yml` file is optional and by default has the same content as the `config.yml`. It is used to add and document more advanced configuration options that should not be part of the minimal config file shipped by default.


### docs.asciidoc [_docs_asciidoc]

The `docs.asciidoc` file contains the documentation about your module. During generation of the documentation, the default config file will be appended to the docs. Use this file to describe your module in more detail and to document specific configuration options.

```asciidoc
This is the {module} module.
```


### fields.yml [_fields_yml_2]

The `fields.yml` file contains the top level structure for the fields in your metricset. It’s used in combination with the `fields.yml` file in each metricset to generate the template and documentation for the fields.

The default file looks like this:

```yaml
- key: {module}
  title: "{module}"
  release: beta
  description: >
    {module} module
  fields:
    - name: {module}
      type: group
      description: >
      fields:
```

Make sure that you update at least the description of the module.


## Testing [_testing_2]

It’s a common pattern to use a `testing.go` file in the module package to share some testing functionality among the metricsets. This file does not have `_test.go` in the name because otherwise it would not be compiled for sub packages.

To see an example of the `testing.go` file, look at the [mysql module](https://github.com/elastic/beats/tree/master/metricbeat/module/mysql).


### Test a Metricbeat module manually [_test_a_metricbeat_module_manually]

To test a Metricbeat module manually, follow the steps below.

First we have to build the Docker image which is available for the modules. The Dockerfile is located inside a `_meta` folder within each module folder. As an example let’s take MySQL module.

This steps assume you have checked out the Beats repository from Github and are inside `beats` directory. First, we have to enter in the `_meta` folder mentioned above and build the Docker image called `metricbeat-mysql`:

```bash
$ cd metricbeat/module/mysql/_meta/
$ docker build -t metricbeat-mysql .
...
Removing intermediate container 0e58cfb7b197
 ---> 9492074840ea
Step 5/5 : COPY test.cnf /etc/mysql/conf.d/test.cnf
 ---> 002969e1d810
Successfully built 002969e1d810
Successfully tagged metricbeat-mysql:latest
```

Before we run the container we have just created, we also need to know which port to expose. The port is listed in the `metricbeat/{{module}}/_meta/env` file:

```bash
$ cat env
MYSQL_DSN=root:test@tcp(mysql:3306)/
MYSQL_HOST=mysql
MYSQL_PORT=3306
```

As we see, the port is 3306. We now have all the information to start our MySQL service locally:

```bash
$ docker run -p 3306:3306 -e MYSQL_ROOT_PASSWORD=secret metricbeat-mysql
```

This starts the container and you can now use it for testing the MySQL module.

To run Metricbeat with the module we need to build the binary, enable the module first. The assumption is now that you are back in the `beats` folder path:

```bash
$ cd metricbeat
$ mage build
$ ./metricbeat modules enable mysql
```

This will enable the module and rename file `metricbeat/modules.d/mysql.yml.disabled` to `metricbeat/modules.d/mysql.yml`. According to our [documentation](/reference/metricbeat/metricbeat-module-mysql.md) we should specify username and password to user MySQL. It’s always a good idea to take a look at the docs to see also that a pre-built dashboard is also available. So tweaking the config a bit, this is how it looks like:

```yaml
$ cat modules.d/mysql.yml

# Module: mysql
# Docs: /beats/docs/reference/ingestion-tools/beats-metricbeat/metricbeat-module-mysql.md

- module: mysql
  metricsets:
    - status
  #  - galera_status
  period: 10s

  # Host DSN should be defined as "user:pass@tcp(127.0.0.1:3306)/"
  # or "unix(/var/lib/mysql/mysql.sock)/",
  # or another DSN format supported by <https://github.com/Go-SQL-Driver/MySQL/>.
  # The username and password can either be set in the DSN or using the username
  # and password config options. Those specified in the DSN take precedence.
  hosts: ["tcp(127.0.0.1:3306)/"]

  # Username of hosts. Empty by default.
  username: root

  # Password of hosts. Empty by default.
  password: secret
```

It’s now sending data to your local Elasticsearch instance. If you need to modify the mysql config, adjust `modules.d/mysql.yml` and restart Metricbeat.


### Run Environment tests for one module [_run_environment_tests_for_one_module]

All the environments are setup with docker. `make integration-tests-environment` and `make system-tests-environment` can be used to run tests for all modules. In case you are developing a module it is convenient to run the tests only for one module and directly run it on your machine.

First you need to start the environment for your module to test and expose the port to your local machine. For this you can run the following command inside the metricbeat directory:

```bash
MODULE=apache PORT=80 make run-module
```

Note: The apache module with port 80 is taken here as an example. You must put the name and port for your own module here.

This will start the environment and you must wait until the service is completely started. After that you can run the test which require an environment:

```bash
MODULE=apache make test-module
```

This will run the integration and system tests connecting to the environment in your docker container.

