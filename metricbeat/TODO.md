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

# Connections
For most metricset, setup creates the connections to the remote hosts. One potential issue is, that if one connection goes down, that it is not setup again means an error is reported in the future. Does this mean Setup should be called every time before fetch but must be able to handle multiple calls? What is the best approach here to guarantee reconnection in case some connections go down?

# More
* Add version number of service. See https://github.com/elastic/beats/issues/619#issuecomment-185242407
