#!/bin/bash

# Usage examples:
# env KIBANA_INDEX='.kibana_env1' ./kibana_import.sh
# ./kibana_import.sh -dir etc/kibana -url http://test.com:9200
# ./kibana_import.sh -dir etc/kibana -url http://test.com:9200 -user admin:secret
# ./kibana_import.sh -dir etc/kibana -url http://test.com:9200 -index .kibana-test

# The default value of the variable. Initialize your own variables here
ELASTICSEARCH=http://localhost:9200
CURL="curl -H Expect:"
KIBANA_INDEX=".kibana"
DIR=.
BEAT_CONFIG=".beatconfig"

print_usage() {
  echo "

Import the dashboards, visualizations and index patterns into Kibana.

The Kibana dashboards together with its dependencies are saved into a
special index pattern in Elasticsearch (by default .kibana), so you need to
specify the Elasticsearch URL and optionally an username and password.

Usage:
  $(basename "$0") -url ${ELASTICSEARCH} -user admin:secret -index ${KIBANA_INDEX}

Options:
  -h | -help
    Print the help menu.
  -d | -dir
    Local directory where the dashboards, visualizations, searches and index pattern are saved.
    By default is current directory.
  -l | -url
    Elasticseacrh URL. By default is ${ELASTICSEARCH}.
  -u | -user
    Username and password for authenticating to Elasticsearch using Basic
    Authentication. The username and password should be separated by a
    colon (i.e. "admin:secret"). By default no username and password are
    used.
  -i | -index
    Kibana index pattern where to save the dashboards, visualizations,
    index patterns. By default is ${KIBANA_INDEX}.

" >&2
}

while [ "$1" != "" ]; do
case $1 in
    -d | -dir )
        DIR=$2
        if [ "$DIR" = "" ]; then
            echo "Error: Missing directory"
            print_usage
            exit 1
        fi
        ;;

    -l | -url )
        ELASTICSEARCH=$2
        if [ "$ELASTICSEARCH" = "" ]; then
            echo "Error: Missing Elasticsearch URL"
            print_usage
            exit 1
        fi
        ;;

    -u | -user )
        USER=$2
        if [ "$USER" = "" ]; then
            echo "Error: Missing username"
            print_usage
            exit 1
        fi
        CURL="curl --user ${USER}"
        ;;

    -i | -index )
        KIBANA_INDEX=$2
        if [ "$KIBANA_INDEX" = "" ]; then
            echo "Error: Missing Kibana index pattern"
            print_usage
            exit 1
        fi
        ;;

    -h | -help )
        print_usage
        exit 0
        ;;

     *)
        echo "Error: Unknown option $2"
        print_usage
        exit 1
        ;;

esac
shift 2
done

if [ "$DIR" = "" ]; then
    echo "Error: Missing directory. Please specify the local directory containing the dashboards, visualizations,
         searches and index patterns."
    print_usage
    exit 1
fi

echo "Import dashboards,visualizations, searches and index pattern from ${DIR} to ${ELASTICSEARCH} in ${KIBANA_INDEX}"

# Workaround for: https://github.com/elastic/beats-dashboards/issues/94
${CURL} -XPUT "${ELASTICSEARCH}/${KIBANA_INDEX}" > /dev/null 2>&1
${CURL} -XPUT "${ELASTICSEARCH}/${KIBANA_INDEX}/_mapping/search" -d'{"search": {"properties": {"hits": {"type": "integer"},
"version": {"type": "integer"}}}}' > /dev/null 2>&1


if [ -f ${BEAT_CONFIG} ]; then
  for ln in `cat ${BEAT_CONFIG}`; do
    BUILD_STRING="${BUILD_STRING}s/${ln}/g;"
  done
  SED_STRING=`echo ${BUILD_STRING} | sed 's/;$//'`
fi
# Failsafe
if [ -z ${SED_STRING} ]; then
  SED_STRING="s/packetbeat-/packetbeat-/g;s/filebeat-/filebeat-/g;s/metricbeat-/metricbeat-/g;s/winlogonbeat-/winlogonbeat-/g"
fi

if [ -d /tmp ]; then
	TMP_SED_FILE="/tmp/load-dashboards-tmp-search.json"
else
	TMP_SED_FILE="${DIR}/search/tmp_search.json"
fi

if [ -d "${DIR}/search" ]; then
    for file in ${DIR}/search/*.json
    do
        NAME=`basename ${file} .json`
        echo "Import search ${NAME}:"
        sed ${SED_STRING} ${file} > ${TMP_SED_FILE}
        ${CURL} -XPUT ${ELASTICSEARCH}/${KIBANA_INDEX}/search/${NAME} \
            -d @${TMP_SED_FILE} || exit 1
        echo
    done
    if [ -f "${TMP_SED_FILE}" ]; then
      rm ${TMP_SED_FILE}
    fi
fi

if [ -d "${DIR}/visualization" ]; then
    for file in ${DIR}/visualization/*.json
    do
        NAME=`basename ${file} .json`
        echo "Import visualization ${NAME}:"
        ${CURL} -XPUT ${ELASTICSEARCH}/${KIBANA_INDEX}/visualization/${NAME} \
            -d @${file} || exit 1
        echo
    done
fi

if [ -d "${DIR}/dashboard" ]; then
    for file in ${DIR}/dashboard/*.json
    do
        NAME=`basename ${file} .json`
        echo "Import dashboard ${NAME}:"
        ${CURL} -XPUT ${ELASTICSEARCH}/${KIBANA_INDEX}/dashboard/${NAME} \
            -d @${file} || exit 1
        echo
    done
fi

if [ -d "${DIR}/index-pattern" ]; then
    for file in ${DIR}/index-pattern/*.json
    do
        NAME=`awk '$1 == "\"title\":" {gsub(/[",]/, "", $2); print $2}' ${file}`
        echo "Import index pattern ${NAME}:"

        ${CURL} -XPUT ${ELASTICSEARCH}/${KIBANA_INDEX}/index-pattern/${NAME} \
            -d @${file} || exit 1
        echo
    done
fi


