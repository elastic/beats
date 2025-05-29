---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/creating-metricsets.html
---

# Creating a Metricset [creating-metricsets]

::::{important}
Elastic provides no warranty or support for the code used to generate metricsets. The generator is mainly offered as guidance for developers who want to create their own data shippers.
::::


A metricset is the part of a Metricbeat module that fetches and structures the data from the remote service. Each module can have multiple metricsets. In this guide, you learn how to create your own metricset.

When creating a metricset for the first time, it generally helps to look at the implementation of existing metricsets for inspiration.

To create a new metricset:

1. Run the following command inside the metricbeat beat directory:

    ```bash
    make create-metricset
    ```

    You need Python to run this command, then, you’ll be prompted to enter a module and metricset name. Remember that a module represents the service you want to retrieve metrics from (like Redis) and a metricset is a specific set of grouped metrics (like `info` on Redis). Only use characters `[a-z]` and, if required, underscores (`_`). No other characters are allowed.

    When you run `make create-metricset`, it creates all the basic files for your metricset, along with the required module files if the module does not already exist. See [Creating a Metricbeat Module](/extend/creating-metricbeat-module.md) for more details about the module files.

    ::::{note}
    We use `{{metricset}}`, `{{module}}`, and `{{beat}}` in this guide as placeholders. You need to replace these with the actual names of your metricset, module, and beat.
    ::::


    The metricset that you created is already a functioning metricset and can be compiled.

2. Compile your new metricset by running the following command:

    ```bash
    mage update
    mage build
    ```

    The first command, `mage update`, updates all generated files with the most recent files, data, and meta information from the metricset. The second command, `mage build`, compiles your source code and provides you with a binary called metricbeat in the same folder. You can run the binary in debug mode with the following command:

    ```bash
    ./metricbeat -e -d "*"
    ```


After running the mage commands, you’ll find the metricset, along with its generated files, under `module/{{module}}/{metricset}`. This directory contains the following files:

* `\{{metricset}}.go`
* `_meta/docs.asciidoc`
* `_meta/data.json`
* `_meta/fields.yml`

Let’s look at the files in more detail next.


## `{{metricset}}.go` File [_metricset_go_file]

The first file is `{{metricset}}.go`. It contains the logic on how to fetch data from the service and convert it for sending to the output.

The generated file looks like this:

[https://github.com/elastic/beats/blob/main/metricbeat/scripts/module/metricset/metricset.go.tmpl](https://github.com/elastic/beats/blob/main/metricbeat/scripts/module/metricset/metricset.go.tmpl)

```go
package {metricset}

import (
    "github.com/elastic/elastic-agent-libs/mapstr"
	"github.com/elastic/beats/v7/libbeat/common/cfgwarn"
	"github.com/elastic/beats/v7/metricbeat/mb"
)

// init registers the MetricSet with the central registry as soon as the program
// starts. The New function will be called later to instantiate an instance of
// the MetricSet for each host is defined in the module's configuration. After the
// MetricSet has been created then Fetch will begin to be called periodically.
func init() {
	mb.Registry.MustAddMetricSet("{module}", "{metricset}", New)
}

// MetricSet holds any configuration or state information. It must implement
// the mb.MetricSet interface. And this is best achieved by embedding
// mb.BaseMetricSet because it implements all of the required mb.MetricSet
// interface methods except for Fetch.
type MetricSet struct {
	mb.BaseMetricSet
	counter int
}

// New creates a new instance of the MetricSet. New is responsible for unpacking
// any MetricSet specific configuration options if there are any.
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {
	cfgwarn.Beta("The {module} {metricset} metricset is beta.")

	config := struct{}{}
	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
		counter:       1,
	}, nil
}

// Fetch method implements the data gathering and data conversion to the right
// format. It publishes the event which is then forwarded to the output. In case
// of an error set the Error field of mb.Event or simply call report.Error().
func (m *MetricSet) Fetch(report mb.ReporterV2) error {
	report.Event(mb.Event{
		MetricSetFields: mapstr.M{
			"counter": m.counter,
		},
	})
	m.counter++

	return nil
}
```

The `package` clause and `import` declaration are part of the base structure of each Go file. You should only modify this part of the file if your implementation requires more imports.


### Initialisation [_initialisation]

The init method registers the metricset with the central registry. In Go the `init()` function is called before the execution of all other code. This means the module will be automatically registered with the global registry.

The `New` method, which is passed to `MustAddMetricSet`, will be called after the setup of the module and before starting to fetch data. You normally don’t need to change this part of the file.

```go
func init() {
	mb.Registry.MustAddMetricSet("{module}", "{metricset}", New)
}
```


### Definition [_definition]

The MetricSet type defines all fields of the metricset. As a minimum it must be composed of the `mb.BaseMetricSet` fields, but can be extended with additional entries. These variables can be used to persist data or configuration between multiple fetch calls.

You can add more fields to the MetricSet type, as you can see in the following example where the `username` and `password` string fields are added:

```go
type MetricSet struct {
	mb.BaseMetricSet
	username    string
	password    string
}
```


### Creation [_creation]

The `New` function creates a new instance of the MetricSet. The setup process of the MetricSet is also part of `New`. This method will be called before `Fetch` is called the first time.

The `New` function also sets up the configuration by processing additional configuration entries, if needed.

```go
func New(base mb.BaseMetricSet) (mb.MetricSet, error) {

	config := struct{}{}

	if err := base.Module().UnpackConfig(&config); err != nil {
		return nil, err
	}

	return &MetricSet{
		BaseMetricSet: base,
	}, nil
}
```


### Fetching [_fetching]

The `Fetch` method is the central part of the metricset. `Fetch` is called every time new data is retrieved. If more than one host is defined, `Fetch` is called once for each host. The frequency of calling `Fetch` is based on the `period` defined in the configuration file.

`Fetch` must publish the event using the `mb.ReporterV2.Event` method. If an error happens, `Fetch` can return an error, or if `Event` is being called in a loop, published using the `mb.ReporterV2.Error` method. This means that Metricbeat always sends an event, even on failure. You must make sure that the error message helps to identify the actual error.

The following example shows a metricset `Fetch` method with a counter that is incremented for each `Fetch` call:

```go
func (m *MetricSet) Fetch(report mb.ReporterV2) error {

	report.Event(mb.Event{
		MetricSetFields: common.MapStr{
			"counter": m.counter,
		}
	})
	m.counter++

	return nil
}
```

The JSON output derived from the reported event will be identical to the naming and structure you use in `common.MapStr`. For more details about `MapStr` and its functions, see the [MapStr API docs](https://godoc.org/github.com/elastic/beats/libbeat/common#MapStr).


### Multi Fetching [_multi_fetching]

`Event` can be called multiple times inside of the `Fetch` method for metricsets that might expose multiple events. `Event` returns a bool that indicates if the metricset is already closed and no further events can be processed, in which case `Fetch` should return immediately. If there is an error while processing one of many events, it can be published using the `mb.ReporterV2.Error` method, as opposed to returning an error value.


### Parsing and Normalizing Fields [_parsing_and_normalizing_fields]

In Metricbeat we aim to normalize the metric names from all metricsets to respect a common [set of conventions](/extend/event-conventions.md). This makes it easy for users to find and interpret metrics. To simplify parsing, converting, renaming, and restructuring of the object read from the monitored system to the Metricbeat format, we have created the [schema](https://godoc.org/github.com/elastic/beats/libbeat/common/schema) package that allows you to declaratively define transformations.

For example, assuming this input object:

```go
input := map[string]interface{}{
	"testString":     "hello",
	"testInt":        "42",
	"testBool":       "true",
	"testFloat":      "42.1",
	"testObjString":  "hello, object",
}
```

And the requirement to transform it into this one:

```go
common.MapStr{
	"test_string": "hello",
	"test_int":    int64(42),
	"test_bool":   true,
	"test_float":  42.1,
	"test_obj": common.MapStr{
		"test_obj_string": "hello, object",
	},
}
```

You can use the schema package to transform the data, and optionally mark some fields in a schema as required or not. For example:

```go
import (
	s "github.com/elastic/beats/libbeat/common/schema"
	c "github.com/elastic/beats/libbeat/common/schema/mapstrstr"
)

var (
	schema = s.Schema{
		"test_string": c.Str("testString", s.Required), <1>
		"test_int":    c.Int("testInt"), <2>
		"test_bool":   c.Bool("testBool", s.Optional), <3>
		"test_float":  c.Float("testFloat"),
		"test_obj": s.Object{
			"test_obj_string": c.Str("testObjString", s.IgnoreAllErrors), <4>
		},
	}
)

func eventMapping(input map[string]interface{}) common.MapStr {
	return schema.Apply(input) <5>
}
```

1. Marks a field as required.
2. If a field has no schema option set, it is equivalent to `Required`.
3. Marks the field as optional.
4. Ignore any value conversion error
5. By default, `Apply` will fail and return an error if any required field is missing. Using the optional second argument, you can specify how `Apply` handles different fields of the  schema. The possible values are:
* `AllRequired` is the default behavior. Returns an error if any required field is missing, including fields that are required because no schema option is set.
* `FailOnRequired`  will fail if a field explicitly marked as `required` is missing.
* `NotFoundKeys(cb func([]string))` takes a callback function that will be called with a list of missing keys, allowing for finer-grained error handling.



In the above example, note that it is possible to create the schema object once and apply it to all events. You can also use `ApplyTo` to add additional data to an existing `MapStr` object:

```go
var (
	schema = s.Schema{
		"test_string": c.Str("testString"),
		"test_int":    c.Int("testInt"),
		"test_bool":   c.Bool("testBool"),
		"test_float":  c.Float("testFloat"),
		"test_obj": s.Object{
			"test_obj_string": c.Str("testObjString"),
		},
	}

	additionalSchema = s.Schema{
		"second_string": c.Str("secondString"),
		"second_int": c.Int("secondInt"),
	}
)

	data, err := schema.Apply(input)
	if err != nil {
		return err
	}

	if m.parseMoreData{
		_, err := additionalSchema.ApplyTo(data, input)
		if len(err) > 0 { <1>
			return err.Err()
		}
	}
```

1. `ApplyTo` returns a raw MultiError object, making it suitable for finer-grained error handling.



## Configuration File [_configuration_file]

The configuration file for a metricset is handled by the module. If there are multiple metricsets in one module, make sure you add all metricsets to the configuration. For example:

```go
metricbeat:
  modules:
    - module: {module-name}
      metricsets: ["{metricset1}", "{metricset2}"]
```

::::{note}
Make sure that you run `make collect` after updating the config file so that your changes are also applied to the global configuration file and the docs.
::::


For more details about the Metricbeat configuration file, see the topic about [Modules](/reference/metricbeat/configuration-metricbeat.md) in the Metricbeat documentation.


## What to Do Next [_what_to_do_next]

This topic provides basic steps for creating a metricset. For more details about metricsets and how to extend your metricset further, see [Metricset Details](/extend/metricset-details.md).

