# Automatic tests for Packetbeat

This repository contains integration tests for the [Packetbeat](http://packetbeat.com)
project.

## Running

        make test

Running a single test, e.g.:

        . env/bin/activate
        nosetests test_0002_thrift_basics.py:Test.test_thrift_integration

## CI

These tests are executed automatically by Travis-CI here:
https://travis-ci.org/packetbeat/packetbeat-tests
