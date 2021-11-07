#!/bin/sh
set -e
go run go.elastic.co/fastjson/cmd/generate-fastjson -f -o marshal_fastjson.go .
exec go-licenser marshal_fastjson.go
