# Runs test build for the count metricset
.PHONY: test
test:
	$(MAKE) -C _templates/beat test
	$(MAKE) -C _templates/metricbeat test

# Cleans up environment
.PHONY: clean
clean:
	@$(MAKE) -C _templates/beat clean
	@$(MAKE) -C _templates/metricbeat clean
