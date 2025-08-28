---
mapped_pages:
  - https://www.elastic.co/guide/en/beats/devguide/current/dev-faq.html
---

# Metricbeat Developer FAQ [dev-faq]

This is a list of common questions when creating a metricset and the potential answers.


## Metricset is not compiled [_metricset_is_not_compiled]

You are compiling your Beat, but the newly created metricset is not compiled?

Make sure that the path to your module and metricset are added as an import path either in your `main.go` file or your `include/list.go` file. You can do this manually or by running `make imports`.


## Metricset is not started [_metricset_is_not_started]

The metricset is compiled, but not started when starting Metricbeat?

After creating your metricset, make sure you run `make collect`. This command adds the configuration of your metricset to the default configuration. If the metricset still doesn’t start, check your default configuration file to see if the metricset is listed there.

