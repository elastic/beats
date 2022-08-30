## Test Scenarios & Framework for Heartbeat

This directory contains only tests, and the framework used to run them. The tests are in `_test.go` files, with the 
framework and scenario definitions contained in the regular `.go` files.

The key types in here are:

- Scenario: A description of a given heartbeat configuration with some additional parameters
- Twist: A way to modify a scenario, by adding one additional configuration parameter perhaps
- ScenarioDB: A database of all known scenarios that lets you easily gather the events created by a single run of each

By using these three types you can create a complex matrix of tests to cover a lot of bases. 

The main idea here is decoupling the generation of test data from the testing of it, letting us test multiple criteria on
a known set of data. This can be thought of as an alternative to fixtures, where each `Scenario` could be thought of as a
dynamic fixture.