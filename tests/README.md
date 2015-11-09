# System tests for Packetbeat

This folder contains the system tests for Packetbeat. The system tests
are written in Python and they make use of the nose framework.

## Running

You need python (>=2.7), virtualenv and pip installed. Then you can prepare
the setup and run all the tests with:

        make test

Running a single test, e.g.:

        . env/bin/activate
        nosetests test_0002_thrift_basics.py:Test.test_thrift_integration
