#!/usr/bin/env bash

# This script is to generate test input files for different elasticsearch versions
#
# The script creates an index, adds a document and writes the output from _stats
# to a document. The document name is based on the first param passed to the script.
# For es 5.1.2 pass 512
#
# Note: Small corrections were made to the output documents as size of the index
# is not the same across all versions

# Delete index first
curl -XDELETE 'http://localhost:9200/testindex'

# Create index
curl -XPUT 'http://localhost:9200/testindex'

# Add document
curl -XPUT 'http://localhost:9200/testindex/test/1?pretty' -H 'Content-Type: application/json' -d'
{
    "user" : "kimchy",
    "message" : "trying out Elasticsearch"
}
'

# Make sure index is created
curl -XPOST 'http://localhost:9200/_forcemerge'

# Read stats output
curl -XGET 'http://localhost:9200/_stats?pretty' > stats.${1}.json

