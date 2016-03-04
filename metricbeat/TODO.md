List of things to keep track on, thoughts on what should be improved. More can also be found in the Github issue under https://github.com/elastic/beats/issues/619


# Kibana Dashboards
Each metricset and module should have its dashboard. The basic scripts to create and aggregate dashboards for metricsets and modules are in place. The current basic dashboards have to be extended with additional visualizations and cleaned up.

The following points are open question:
* Should there be one module per metricset or per module or both?
* Storage of visualizations only for metricsets or modules should be possible. The reasoning is that if metricbeat is used as a library to add a metricset, it should be possible to also have a dashboard inside (self contained)

# Mapping
Scripts to generate the template from fields.yml are implemented. It must be checked in details what additional default mapping should be introduced for metricbeat. Some ideas:

* disable _all for all fields
* disable _source for all fields (use doc values instead)

The mapping for each metricset must be completed and verified

# Filtering
A more generic filtering should be introduced so not all data is sent by default. In general I think every metricset should be able to provide the full data set, but by default it should only send a reasonable amount of data.

Filtering can happen through generic filtering but a less complex options with levels or something similar would be nice to have. One idea would be to have 3 levels on the data side: Minimum, Basic, Full. Each metricset would have to support that and the level would be configurable. An alternative is that each metricset has to implement its own configuration on how to handle which parts are enabled.

Generic filtering support must be added to the module.

# Topbeat
Topbeat should be added to metricbeat (see https://github.com/elastic/beats/pull/1081). To make the two better integrated some refactoring on the Topbeat side is needed so MapStr can be consumed directly.

# More
* Add service host as default event informartion. See https://github.com/elastic/beats/issues/619#issuecomment-185242407
* Add version number of service. See https://github.com/elastic/beats/issues/619#issuecomment-185242407
