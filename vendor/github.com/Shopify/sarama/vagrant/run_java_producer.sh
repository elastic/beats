#!/bin/sh

set -ex

wget https://github.com/FrancoisPoinsot/simplest-uncommitted-msg/releases/download/0.1/simplest-uncommitted-msg-0.1-jar-with-dependencies.jar
java -jar simplest-uncommitted-msg-0.1-jar-with-dependencies.jar -b ${KAFKA_HOSTNAME}:9092 -c 4