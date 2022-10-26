#! /bin/bash -x

# Exit on errors
set -e

BEATS_VERSION="${BEATS_VERSION:=8.4.3}"
FB_TAR_NAME=filebeat-$BEATS_VERSION-linux-x86_64.tar.gz
FB_FOLDER_NAME=filebeat-$BEATS_VERSION-linux-x86_64

MB_TAR_NAME=metricbeat-$BEATS_VERSION-linux-x86_64.tar.gz
MB_FOLDER_NAME=metricbeat-$BEATS_VERSION-linux-x86_64

ES_USER="${ES_USER:=elastic}"
ES_PASS="${ES_PASS:=changeme}"

JSON_LOGS="${JSON_LOGS:=false}"

if [[ "$JSON_LOGS" = 'true' ]]
then
    LOG_FILE="${LOG_FILE:=/tmp/flog.ndjson}"
else
    LOG_FILE="${LOG_FILE:=/tmp/flog.log}"
fi

## Configure ES Cluster to accept metrics
if [[ $VERIFICATION_MODE = "none" ]]
then
    curl --location --request PUT 'https://localhost:9200/_cluster/settings' \
         --insecure \
         -u $ES_USER:$ES_PASS \
         --header 'Content-Type: application/json' \
         --data-raw '{
           "persistent": {
             "xpack.monitoring.collection.enabled": true
           }
         }'
        export CLUSTER_UUID=$(curl --insecure --location --request GET 'https://localhost:9200/' \
                            -u $ES_USER:$ES_PASS | jq '.cluster_uuid')
else
    curl --location --request PUT 'https://localhost:9200/_cluster/settings' \
         -u $ES_USER:$ES_PASS \
         --header 'Content-Type: application/json' \
         --data-raw '{
           "persistent": {
             "xpack.monitoring.collection.enabled": true
           }
         }'
        export CLUSTER_UUID=$(curl --location --request GET 'https://localhost:9200/' \
                            -u $ES_USER:$ES_PASS | jq '.cluster_uuid')
fi

# Install flog
if [[ ! -e flog ]]
then
     git clone https://github.com/belimawr/flog.git
     cd flog
     go install .
     cd ../
else
    echo "Flog folder found, assuming it's already installed"
fi

# Install Filebeat
if [ -e $FB_TAR_NAME ]
then
    echo "Filebeat already downloaded, skipping"
else
    wget https://artifacts.elastic.co/downloads/beats/filebeat/$FB_TAR_NAME
fi

if [ -e $FB_FOLDER_NAME ]
then
   echo "Filebeat already extracted"
else
    tar -xf $FB_TAR_NAME
fi

# Install Metricbeat
if [ -e $MB_TAR_NAME ]
then
    echo "Metricbeat already downloaded, skipping"
else
    wget https://artifacts.elastic.co/downloads/beats/metricbeat/$MB_TAR_NAME
fi

if [ -e $MB_FOLDER_NAME ]
then
   echo "Metricbeat already extracted"
else
    tar -xf $MB_TAR_NAME
fi

# Deploy Metricbeat to the host

# Copy filebeat config file
if [[ "$JSON_LOGS" = 'true' ]]
then
    if [[ ! -e $LOG_FILE ]]
    then
        flog -f json -r 42 -n 2000000 -s 0.001 -t log -w -o $LOG_FILE
    else
        echo "Log file found"
    fi
    cp ./filebeat_json.yml $FB_FOLDER_NAME/filebeat.yml
else
    # Generate a 1Gb file
    if [[ ! -e $LOG_FILE ]]
    then
        flog -f apache_common -r 42 -n 2000000 -s 0.001 -t log -w -o $LOG_FILE
    else
        echo "Log file found"
    fi
    cp ./filebeat.yml $FB_FOLDER_NAME/filebeat.yml
fi

# Copy Metricbeat configuration files
cp -vr ./modules.d $MB_FOLDER_NAME/
cp -v ./metricbeat.yml $MB_FOLDER_NAME/

# Start Metricbeat
cd $MB_FOLDER_NAME
./metricbeat setup
./metricbeat &
cd ../

# Remove the registry
cd $FB_FOLDER_NAME
rm -rvf data

# Start filebeat
./filebeat -e -v |tee filebeat.log

# Kill Metricbeat
killall metricbeat
