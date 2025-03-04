---
navigation_title: "Index lifecycle management (ILM)"
mapped_pages:
  - https://www.elastic.co/guide/en/beats/metricbeat/current/ilm.html
---

# Configure index lifecycle management [ilm]


Use the [index lifecycle management](docs-content://manage-data/lifecycle/index-lifecycle-management/tutorial-automate-rollover.md) (ILM) feature in {{es}} to manage your Metricbeat their backing indices of your data streams as they age. Metricbeat loads the default policy automatically and applies it to any data streams created by Metricbeat.

You can view and edit the policy in the **Index lifecycle policies** UI in {{kib}}. For more information about working with the UI, see [Index lifecyle policies](docs-content://manage-data/lifecycle/index-lifecycle-management.md).

Example configuration:

```yaml
setup.ilm.enabled: true
```

::::{warning}
If index lifecycle management is enabled (which is typically the default), `setup.template.name` and `setup.template.pattern` are ignored.
::::



## Configuration options [_configuration_options_10]

You can specify the following settings in the `setup.ilm` section of the `metricbeat.yml` config file:


### `setup.ilm.enabled` [setup-ilm-option]

Enables or disables index lifecycle management on any new indices created by Metricbeat. Valid values are `true` and `false`.


### `setup.ilm.policy_name` [setup-ilm-policy_name-option]

The name to use for the lifecycle policy. The default is `metricbeat`.


### `setup.ilm.policy_file` [setup-ilm-policy_file-option]

The path to a JSON file that contains a lifecycle policy configuration. Use this setting to load your own lifecycle policy.

For more information about lifecycle policies, see [Set up index lifecycle management policy](docs-content://manage-data/lifecycle/index-lifecycle-management/configure-lifecycle-policy.md) in the *{{es}} Reference*.


### `setup.ilm.check_exists` [setup-ilm-check_exists-option]

When set to `false`, disables the check for an existing lifecycle policy. The default is `true`. You need to disable this check if the Metricbeat user connecting to a secured cluster doesn’t have the `read_ilm` privilege.

If you set this option to `false`, lifecycle policy will not be installed, even if `setup.ilm.overwrite` is set to `true`.


### `setup.ilm.overwrite` [setup-ilm-overwrite-option]

When set to `true`, the lifecycle policy is overwritten at startup. The default is `false`.

