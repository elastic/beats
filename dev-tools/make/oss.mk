#
# Variables
#
ES_BEATS   ?= ..
PYTHON_ENV ?= $(ES_BEATS)

#
# Includes
#
include $(ES_BEATS)/dev-tools/make/mage_wrapper.mk
include $(ES_BEATS)/dev-tools/make/misspell.mk
include $(ES_BEATS)/dev-tools/make/gox.mk
