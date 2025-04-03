#!/usr/bin/env bash

# This script is to generate test input files for different elasticsearch versions
#
# The script creates an index, adds a document and writes the output from _stats
# to a document. The document name is based on the first param passed to the script.
# For es 5.1.2 pass 512
#
# Note: Small corrections were made to the output documents as size of the index
# is not the same across all versions

# It is assumed you used elastic-package to start the elasticsearch server
USER=elastic
PASS=elastic
ENDPOINT=http://localhost:9200

# Delete index first
curl --insecure -u ${USER}:${PASS} -XDELETE ${ENDPOINT}/testindex

# Create index
curl --insecure -u ${USER}:${PASS} -XPUT ${ENDPOINT}/testindex

# Add document
curl --insecure -u ${USER}:${PASS} -XPUT ${ENDPOINT}/testindex/_doc/1?pretty -H 'Content-Type: application/json' -d'
{
    "user" : "kimchy",
    "message" : "trying out Elasticsearch"
}
'

# Make sure index is created
curl --insecure -u ${USER}:${PASS} -XPOST ${ENDPOINT}/testindex/_forcemerge

# Read root
curl --insecure -u ${USER}:${PASS} -XGET ${ENDPOINT}/?pretty > root.${1}.json

# Read stats output
curl --insecure -u ${USER}:${PASS} -XGET ${ENDPOINT}/_stats?pretty > stats.${1}.json

# Read index settings
# /!\ A test case with missing settings is required! Make sure you curate the result accordingly
# See data.go and the things logged as debug for more information
curl --insecure -u ${USER}:${PASS} -XGET ${ENDPOINT}/*,.*/_settings?pretty > settings.${1}.json

# Read cluster state
curl --insecure -u ${USER}:${PASS} -XGET ${ENDPOINT}/_cluster/state?pretty > cluster_state.${1}.json

# Read xpack usage
curl --insecure -u ${USER}:${PASS} -XGET ${ENDPOINT}/_xpack/usage?pretty > xpack_usage.${1}.json
