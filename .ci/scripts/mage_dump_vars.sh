#!/usr/bin/env bash
set -exuo pipefail

if command -v mage
then
  mage dumpVariables
else
  echo "WARN: cannot dump mage variables because mage is not installed"
fi


