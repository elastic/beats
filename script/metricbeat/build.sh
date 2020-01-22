#!/bin/bash

make collect 
make
go build -o metricbeat main.go
GOOS=windows GOARCH=amd64 go build -o metricbeat.exe main.go
