#!/bin/bash

INPUT=$1
OUTPUT=$2

PB_VERSION=1.0.0-beta3
ES_VERSION=1.5.2
KIBANA_VERSION=4.0.2
DASHBOARDS_VERSION=1.0.0-beta2

usage() {
    echo "Usage: $0 etc/gettingstarted.in.asciidoc etc/gettingstarted.asciidoc"
}

if [ -z $INPUT ]; then
    usage
    exit 1
fi

if [ -z $OUTPUT ]; then
    usage
    exit 1
fi

cat << EOF > $OUTPUT
////

This file is generated! Edit gettingstarted.in.asciidoc instead and then
re-generate this file with:

  ../scripts/generate_gettingstarted.sh gettingstarted.in.asciidoc gettingstarted.asciidoc

////

EOF

cat $INPUT >> $OUTPUT

sed -i.bk "s/\$PB_VERSION/$PB_VERSION/g" $OUTPUT
sed -i.bk "s/\$ES_VERSION/$ES_VERSION/g" $OUTPUT
sed -i.bk "s/\$KIBANA_VERSION/$KIBANA_VERSION/g" $OUTPUT
sed -i.bk "s/\$DASHBOARDS_VERSION/$DASHBOARDS_VERSION/g" $OUTPUT
rm $OUTPUT.bk
