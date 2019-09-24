BEAT_TYPE       = metricbeat
PREPARE_COMMAND = MODULE=elastic METRICSET=test make create-metricset;

include ../common/Makefile

# Collects all dependencies and then calls update
.PHONY: collect
collect:
