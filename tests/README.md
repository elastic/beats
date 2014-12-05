# Automatic tests for Packetbeat

This repository contains the integration tests for the [Packetbeat](http://packetbeat.com)
project.

## Running

        make test

Running a single test, e.g.:

        . env/bin/activate
        nosetests test_0002_thrift_basics.py:Test.test_thrift_integration

## Build status

[![Build Status](https://travis-ci.org/packetbeat/packetbeat.svg?branch=master)](https://travis-ci.org/packetbeat/packetbeat)
