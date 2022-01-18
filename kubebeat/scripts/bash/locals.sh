#!/usr/bin/env bash

  for file in deploy/k8s/*.yaml; do
      cp "$file" "${file%.yaml}-local.yaml"
  done
