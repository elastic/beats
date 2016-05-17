/*
Package module contains Metricbeat modules and their MetricSet implementations.

= Naming conventions

For the key names, metricbeat follows the naming conventions below:

* all field keys lower case
* snake case for combining words
* Group related fields in sub documents, which means using the . notation. Groups are mostly described by common prefixes.
* Prevent namespace duplication. If connections appears in the namespace, it's not needed in the sub document
* Do not use complex abbreviations. A list of standardised abbreviations can be found below.
* Organise the documents from the general to the details, which allows namespacing. The type should always be last, like .pct.
* If two fields are the same but with different units, remove the less granular one
* In case the value correlates with the name of a nested document, use value inside the document
* Do not use . in the names

The goal is to have a similar experience across all metrics.


= Abbrevations

List of standardised words and units across all metricsets. On the left are the ones to be used, on the right the options seen in metricsets.

* avg: average
* count:
* day: days, d
* max: maximumg
* min: minimum
* pct: precentage
* request:
* sec: seconds, second, s
*

*/
package module
