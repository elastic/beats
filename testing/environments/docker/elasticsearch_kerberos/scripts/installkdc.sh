#!/bin/bash

# Licensed to Elasticsearch under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. Elasticsearch licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.

set -e

LOCALSTATEDIR=/etc
KDC_CONFIG=/var/kerberos
LOGDIR=/var/log/krb5

#MARKER_FILE=/etc/marker

# Transfer and interpolate krb5.conf
cp /config/krb5.conf.template $LOCALSTATEDIR/krb5.conf
sed -i 's/${REALM_NAME}/'$REALM_NAME'/g' $LOCALSTATEDIR/krb5.conf
sed -i 's/${KDC_NAME}/'$KDC_NAME'/g' $LOCALSTATEDIR/krb5.conf
sed -i 's/${BUILD_ZONE}/'$BUILD_ZONE'/g' $LOCALSTATEDIR/krb5.conf
sed -i 's/${ELASTIC_ZONE}/'$ELASTIC_ZONE'/g' $LOCALSTATEDIR/krb5.conf


# Transfer and interpolate the kdc.conf
mkdir -p $KDC_CONFIG/krb5kdc
cp /config/kdc.conf.template $KDC_CONFIG/krb5kdc/kdc.conf
sed -i 's/${REALM_NAME}/'$REALM_NAME'/g' $KDC_CONFIG/krb5kdc/kdc.conf
sed -i 's/${KDC_NAME}/'$KDC_NAME'/g' $KDC_CONFIG/krb5kdc/kdc.conf
sed -i 's/${BUILD_ZONE}/'$BUILD_ZONE'/g' $KDC_CONFIG/krb5kdc/kdc.conf
sed -i 's/${ELASTIC_ZONE}/'$ELASTIC_ZONE'/g' $LOCALSTATEDIR/krb5.conf

# Touch logging locations
mkdir -p $LOGDIR
touch $LOGDIR/kadmin.log
touch $LOGDIR/krb5kdc.log
touch $LOGDIR/krb5lib.log

# Update package manager
yum update -qqy

# Install krb5 packages
yum install -qqy krb5-{server,libs,workstation} sudo

# Create kerberos database with stash file and garbage password
kdb5_util create -s -r $REALM_NAME -P zyxwvutsrpqonmlk9876

# Set up admin acls
cat << EOF > /var/kerberos/krb5kdc/kadm5.acl
*/admin@$REALM_NAME	*
*@$REALM_NAME   	*
*/*@$REALM_NAME	    i
EOF

# Create admin principal
kadmin.local -q "addprinc -pw elastic admin/admin@$REALM_NAME"
kadmin.local -q "ktadd -k /etc/admin.keytab admin/admin@$REALM_NAME"

# set ownership for ES
chown -R elasticsearch:elasticsearch $LOGDIR
chown -R elasticsearch:elasticsearch $KDC_CONFIG
chown -R elasticsearch:elasticsearch $LOCALSTATEDIR/krb5.conf
chown -R elasticsearch:elasticsearch $LOCALSTATEDIR/admin.keytab


# Create a link so addprinc.sh is on path
ln -s /scripts/addprinc.sh /usr/bin/
