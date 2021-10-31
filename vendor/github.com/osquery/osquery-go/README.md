# osquery-go

[![CircleCI](https://circleci.com/gh/osquery/osquery-go/tree/master.svg?style=svg)](https://circleci.com/gh/osquery/osquery-go/tree/master)
[![GoDoc](https://godoc.org/github.com/osquery/osquery-go?status.svg)](http://godoc.org/github.com/osquery/osquery-go)

[osquery](https://github.com/facebook/osquery) exposes an operating system as a high-performance relational database. This allows you to write SQL-based queries to explore operating system data. With osquery, SQL tables represent abstract concepts such as running processes, loaded kernel modules, open network connections, browser plugins, hardware events or file hashes.

If you're interested in learning more about osquery, visit the [GitHub project](https://github.com/facebook/osquery), the [website](https://osquery.io), and the [users guide](https://osquery.readthedocs.io).

## What is osquery-go?

In osquery, SQL tables, configuration retrieval, log handling, etc. are implemented via a robust plugin and extensions API. This project contains Go bindings for creating osquery extensions in Go. To create an extension, you must create an executable binary which instantiates an `ExtensionManagerServer` and registers the plugins that you would like to be added to osquery. You can then have osquery load the extension in your desired context (ie: in a long running instance of `osqueryd` or during an interactive query session with `osqueryi`). For more information about how this process works at a lower level, see the osquery [wiki](https://osquery.readthedocs.io/en/latest/development/osquery-sdk/).

## Install

This library is compatible with Go Modules. To install:

``` go
go get github.com/osquery/osquery-go
```

## Using the library

### Creating a new osquery table

If you want to create a custom osquery table in Go, you'll need to write an extension which registers the implementation of your table. Consider the following Go program:


```go
package main

import (
	"context"
	"log"
	"os"
	"flag"

	"github.com/osquery/osquery-go"
	"github.com/osquery/osquery-go/plugin/table"
)

func main() {
	socket := flag.String("socket", "", "Path to osquery socket file")
	flag.Parse()
	if *socket == "" {
		log.Fatalf(`Usage: %s --socket SOCKET_PATH`, os.Args[0])
	}

	server, err := osquery.NewExtensionManagerServer("foobar", *socket)
	if err != nil {
		log.Fatalf("Error creating extension: %s\n", err)
	}

	// Create and register a new table plugin with the server.
	// table.NewPlugin requires the table plugin name,
	// a slice of Columns and a Generate function.
	server.RegisterPlugin(table.NewPlugin("foobar", FoobarColumns(), FoobarGenerate))
	if err := server.Run(); err != nil {
		log.Fatalln(err)
	}
}

// FoobarColumns returns the columns that our table will return.
func FoobarColumns() []table.ColumnDefinition {
	return []table.ColumnDefinition{
		table.TextColumn("foo"),
		table.TextColumn("baz"),
	}
}

// FoobarGenerate will be called whenever the table is queried. It should return
// a full table scan.
func FoobarGenerate(ctx context.Context, queryContext table.QueryContext) ([]map[string]string, error) {
	return []map[string]string{
		{
			"foo": "bar",
			"baz": "baz",
		},
		{
			"foo": "bar",
			"baz": "baz",
		},
	}, nil
}
```

To test this code, start an osquery shell and find the path of the osquery extension socket:

```sql
osqueryi --nodisable_extensions
osquery> select value from osquery_flags where name = 'extensions_socket';
+-----------------------------------+
| value                             |
+-----------------------------------+
| /Users/USERNAME/.osquery/shell.em |
+-----------------------------------+
```

Then start the Go extension and have it communicate with osqueryi via the extension socket that you retrieved above:

```bash
go run ./my_table_plugin.go --socket /Users/USERNAME/.osquery/shell.em
```

Alternatively, you can also autoload your extension when starting an osquery shell:

```bash
go build -o my_table_plugin my_table_plugin.go
osqueryi --extension /path/to/my_table_plugin
```

This will register a table called "foobar". As you can see, the table will return two rows:

```sql
osquery> select * from foobar;
+-----+-----+
| foo | baz |
+-----+-----+
| bar | baz |
| bar | baz |
+-----+-----+
osquery>
```

This is obviously a contrived example, but it's easy to imagine the possibilities.

Using the instructions found on the [wiki](https://osquery.readthedocs.io/en/latest/development/osquery-sdk/), you can deploy your extension with an existing osquery deployment.

### Creating logger and config plugins

The process required to create a config and/or logger plugin is very similar to the process outlined above for creating an osquery table. Specifically, you would create an `ExtensionManagerServer` instance in `func main()`, register your plugin and launch the extension as described above. The only difference is that the implementation of your plugin would be different. Each plugin package has a `NewPlugin` function which takes the plugin name as the first argument, followed by a list of required arguments to implement the plugin.
For example, consider the implementation of an example logger plugin:

```go
func main() {
    // create and register the plugin
	server.RegisterPlugin(logger.NewPlugin("example_logger", LogString))
}

func LogString(ctx context.Context, typ logger.LogType, logText string) error {
	log.Printf("%s: %s\n", typ, logText)
	return nil
}
```

Additionally, consider the implementation of an example config plugin:

```go
func main() {
    // create and register the plugin
	server.RegisterPlugin(config.NewPlugin("example", GenerateConfigs))
}

func GenerateConfigs(ctx context.Context) (map[string]string, error) {
	return map[string]string{
		"config1": `
{
  "options": {
    "host_identifier": "hostname",
    "schedule_splay_percent": 10
  },
  "schedule": {
    "macos_kextstat": {
      "query": "SELECT * FROM kernel_extensions;",
      "interval": 10
    },
    "foobar": {
      "query": "SELECT foo, bar, pid FROM foobar_table;",
      "interval": 600
    }
  }
}
`,
	}, nil
}
```

All of these examples and more can be found in the [examples](./examples) subdirectory of this repository.

### Execute queries in Go

This library can also be used to create a Go client for the osqueryd or osqueryi's extension socket. You can use this to add the ability to performantly execute osquery queries to your Go program. Consider the following example:

```go
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/osquery/osquery-go"
)

func main() {
	if len(os.Args) != 3 {
		log.Fatalf("Usage: %s SOCKET_PATH QUERY", os.Args[0])
	}

	client, err := osquery.NewClient(os.Args[1], 10*time.Second)
	if err != nil {
		log.Fatalf("Error creating Thrift client: %v", err)
	}
	defer client.Close()

	resp, err := client.Query(os.Args[2])
	if err != nil {
		log.Fatalf("Error communicating with osqueryd: %v",err)
	}
	if resp.Status.Code != 0 {
		log.Fatalf("osqueryd returned error: %s", resp.Status.Message)
	}

	fmt.Printf("Got results:\n%#v\n", resp.Response)
}
```

### Loading extensions with osqueryd

If you write an extension with a logger or config plugin, you'll likely want to autoload the extensions when `osqueryd` starts. `osqueryd` has a few requirements for autoloading extensions, documented on the [wiki](https://osquery.readthedocs.io/en/latest/deployment/extensions/). Here's a quick example using a logging plugin to get you started:

1. Build the plugin. Make sure to add `.ext` as the file extension. It is required by osqueryd.
```go build -o /usr/local/osquery_extensions/my_logger.ext```

2. Set the correct permissions on the file and directory. If `osqueryd` runs as root, the directory for the extension must only be writable by root.

```
sudo chown -R root /usr/local/osquery_extensions/
```

3. Create an `extensions.load` file with the path of your extension.

```
echo "/usr/local/osquery_extensions/my_logger.ext" > /tmp/extensions.load
```

4. Start `osqueryd` with the `--extensions_autoload` flag.

```
sudo osqueryd --extensions_autoload=/tmp/extensions.load --logger-plugin=my_logger -verbose
```


## Contributing

For more information on contributing to this project, see [CONTRIBUTING.md](./CONTRIBUTING.md).

## Vulnerabilities

If you find a vulnerability in this software, please email [security@kolide.co](mailto:security@kolide.co).
