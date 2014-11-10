# Automatic tests for Packetbeat

This repository contains integration tests for the [Packetbeat](http://packetbeat.com)
project.

## Running

        make test

Running a single test, e.g.:

        . env/bin/activate
        nose

## CI

These tests are executed automatically by Travis-CI here:
https://travis-ci.org/packetbeat/packetbeat-tests
