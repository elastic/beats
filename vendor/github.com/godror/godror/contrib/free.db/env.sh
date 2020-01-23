export TNS_ADMIN="$(dirname "$(find "$PWD" -type f -name tnsnames.ora | sort -r | head -n1)")"
export GODROR_TEST_USERNAME=test
export GODROR_TEST_PASSWORD=r97oUPimsmTOIcBaeeDF
export GODROR_TEST_DB=free_high
export GODROR_TEST_STANDALONE=1
