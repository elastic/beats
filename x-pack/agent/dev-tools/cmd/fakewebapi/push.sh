#!/bin/sh
FILE=${1:-"action_example.json"}
curl -X POST --data "@$FILE" http://localhost:8080/admin/actions
