#!/bin/sh
APIKEY=${1:-"abc123"}
AGENTID=${2:-"agent007"}
FILE=${3:-"checkin.json"}
curl -H "Authorization: ApiKey $APIKEY" -X POST --data "@$FILE" http://localhost:8080/api/fleet/agents/$AGENTID/checkin
