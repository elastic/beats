# Support a new protocol

This is a simple guide, written while implementing support for mongodb protocol in order to make life easier for following developers.

# Have a look at the code

Have a look at [protos.go](./protos/protos.go), it defines the ProtocolPlugin interface that you will have to implement yourself, it also contains a list of implemented protocols that you will have to extend.

Other files that you will have to complete are [main.go](./main.go), [config.go](./config/config.go), [packetbeat.conf](./packetbeat.conf), [packetbeat.dev.conf](./packetbeat.dev.conf) and [fields.yml](./etc/fields.yml).

For a quite simple example have a look at [redis.go](./protos/redis/redis.go) and its unit test file [redis_test.go](./protos/redis_test.go).

# Create a test dataset

Test suites are based on [pcap files](./tests/pcaps), that are dumps from traffic sniffing tools. For example this command listens to all traffic from a mongodb server running in a docker container and writes the result to a file:

    tcpdump -s 0 port 27017 -i docker0 -w tests/pcaps/mongodb_find.pcap

# Nosetests

The 'tests' directory contains tests written in python that run the full packetbeat program. You can add some of yours based on the pcaps files of you test dataset.

You will need to add a few lines into the [template of configuration file](./tests/templates/packetbeat.conf.js) used in these tests.
