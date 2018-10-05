#!/bin/bash

# Copy everything to OSS folder
cp -R mssql ../../../metricbeat/module

# Move working directory there
cd ../../../metricbeat/module/mssql

# Execute tests there
go test -cover -tags='integration' ./...

# Delete contents there
cd ..
# rm -r mssql

# Come back to initial WD
cd ../../x-pack/metricbeat/module