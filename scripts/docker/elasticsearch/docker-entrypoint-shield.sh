#!/bin/bash

# Add the admin user
echo "adding admin user $ES_USER..."
esusers --path.conf=/usr/share/elasticsearch/config useradd $ES_USER -p $ES_PASS -r admin
echo 'done'

# run original entrypoint
exec /docker-entrypoint.sh $@
