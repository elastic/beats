#
# Variables
#
PYTHON_ENV     ?= .
PYTHON_VE_DIR  ?= $(PYTHON_ENV)/build/ve/$(shell uname -s | tr A-Z a-z)
PYTHON_VE_REQS ?= $(ES_BEATS)/libbeat/tests/system/requirements.txt

#
# Targets
#

# Create a Python virtualenv. All Beats share the same virtual environment.
python-env: $(PYTHON_VE_DIR)/bin/activate
$(PYTHON_VE_DIR)/bin/activate: $(ES_BEATS)/libbeat/tests/system/requirements.txt
	@test -d $(PYTHON_VE_DIR) || virtualenv $(PYTHON_VE_DIR)
	@$(PYTHON_VE_DIR)/bin/pip install -Ur $(PYTHON_VE_REQS)
	@touch $(PYTHON_VE_DIR)/bin/activate
