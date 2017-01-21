
# Runs test build for the count metricset
.PHONY: test
test:
	$(MAKE) -C beat test
	$(MAKE) -C metricbeat/metricset test

# Cleans up environment
.PHONY: clean
clean:
	$(MAKE) -C beat clean
	$(MAKE) -C metricbeat/metricset clean
