:::::{admonition} Prefer to use {{agent}} for this use case?
Refer to the [Elastic Integrations documentation](integration-docs://reference/mongodb/index.md).

::::{dropdown} Learn more
{{agent}} is a single, unified way to add monitoring for logs, metrics, and other types of data to a host. It can also protect hosts from security threats, query data from operating systems, forward data from remote services or hardware, and more. Refer to the documentation for a detailed [comparison of {{beats}} and {{agent}}](docs-content://reference/fleet/index.md).

::::


:::::


This module periodically fetches metrics from [MongoDB](https://www.mongodb.com) servers.


## Module-specific configuration notes [_module_specific_configuration_notes_11]

When configuring the `hosts` option, you must use MongoDB URLs of the following format:

```
[mongodb://][user:pass@]host[:port][?options]
```

Or

```
mongodb://[username:password@]host1[:port1][,...hostN[:portN]][/[defaultauthdb][?options]]
```

The URL can be as simple as:

```yaml
- module: mongodb
  hosts: ["localhost"]
```

Or more complex like:

```yaml
- module: mongodb
  hosts: ["mongodb://myuser:mypass@localhost:40001", "otherhost:40001"]
```

Some more supported URLs are:

```yaml
- module: mongodb
  hosts: ["mongodb://localhost:27017,localhost:27022,localhost:27023"]
```

```yaml
- module: mongodb
  hosts: ["mongodb://localhost:27017/?directConnection=true"]
```

When the parameter `directConnection=true` is included in the connection URI, all operations are executed on the host specified in the URI. It’s important to note that `directConnection=true` must be explicitly specified in the URI, as it won’t be added automatically unless specified.

```yaml
- module: mongodb
  hosts: ["mongodb://localhost:27017,localhost:27022,localhost:27023/?replicaSet=dbrs"]
```

The username and password can be included in the URL or they can be set using the respective configuration options. The credentials in the URL take precedence over the username and password configuration options.

```yaml
- module: mongodb
  metricsets: ["status"]
  hosts: ["localhost:27017"]
  username: root
  password: test
```

The default metricsets are `collstats`, `dbstats` and `status`.


## Compatibility [_compatibility_34]

The MongoDB metricsets were tested with MongoDB 5.0 and are expected to work with all versions >= 5.0.


## MongoDB Privileges [_mongodb_privileges]

In order to use the metricsets, the MongoDB user specified in the module configuration needs to have certain [privileges](https://docs.mongodb.com/manual/core/authorization/#privileges).

We recommend using the [`clusterMonitor` role](https://docs.mongodb.com/manual/reference/built-in-roles/#clusterMonitor) to cover all the necessary privileges.

You can use the following command in Mongo shell to create the privileged user (make sure you are using the `admin` db by using `db` command in Mongo shell).

```js
db.createUser(
    {
        user: "beats",
        pwd: "pass",
        roles: ["clusterMonitor"]
    }
)
```

You can use the following command in Mongo shell to grant the role to an existing user (make sure you are using the `admin` db by using `db` command in Mongo shell).

```js
db.grantRolesToUser("user", ["clusterMonitor"])
```
