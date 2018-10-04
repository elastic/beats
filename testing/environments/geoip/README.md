Files to generate GeoIP2 fixtures for testing in order to avoid unexpected
changes on modules using geoip plugin.

Data is directly provided from the `createdb.pl` script. To update the database
just run edit the script and run `make`, it builds a docker image and runs the
script on it.

To read from the DB, `readdb.pl` can be used.
