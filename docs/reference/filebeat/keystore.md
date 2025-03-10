---
navigation_title: "Secrets keystore"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/filebeat/current/keystore.html
---

# Secrets keystore for secure settings [keystore]


When you configure Filebeat, you might need to specify sensitive settings, such as passwords. Rather than relying on file system permissions to protect these values, you can use the Filebeat keystore to obfuscate stored secret values for use in configuration settings.

After adding a key and its secret value to the keystore, you can use the key in place of the secret value when you configure sensitive settings.

The syntax for referencing keys is identical to the syntax for environment variables:

`${KEY}`

Where KEY is the name of the key.

For example, imagine that the keystore contains a key called `ES_PWD` with the value `yourelasticsearchpassword`:

* In the configuration file, use `output.elasticsearch.password: "${ES_PWD}"`
* On the command line, use: `-E "output.elasticsearch.password=\${ES_PWD}"`

When Filebeat unpacks the configuration, it resolves keys before resolving environment variables and other variables.

Notice that the Filebeat keystore differs from the Elasticsearch keystore. Whereas the Elasticsearch keystore lets you store `elasticsearch.yml` values by name, the Filebeat keystore lets you specify arbitrary names that you can reference in the Filebeat configuration.

To create and manage keys, use the `keystore` command. See the [command reference](/reference/filebeat/command-line-options.md#keystore-command) for the full command syntax, including optional flags.

::::{note}
The `keystore` command must be run by the same user who will run Filebeat.
::::



## Create a keystore [creating-keystore]

To create a secrets keystore, use:

```sh
filebeat keystore create
```

Filebeat creates the keystore in the directory defined by the `path.data` configuration setting.


## Add keys [add-keys-to-keystore]

To store sensitive values, such as authentication credentials for Elasticsearch, use the `keystore add` command:

```sh
filebeat keystore add ES_PWD
```

When prompted, enter a value for the key.

To overwrite an existing key’s value, use the `--force` flag:

```sh
filebeat keystore add ES_PWD --force
```

To pass the value through stdin, use the `--stdin` flag. You can also use `--force`:

```sh
cat /file/containing/setting/value | filebeat keystore add ES_PWD --stdin --force
```


## List keys [list-settings]

To list the keys defined in the keystore, use:

```sh
filebeat keystore list
```


## Remove keys [remove-settings]

To remove a key from the keystore, use:

```sh
filebeat keystore remove ES_PWD
```

