#!/bin/sh

set -e

if [ $BEAT_NAME = "packetbeat" ]; then
	patch -p1 < /gopacket_pcap.patch
fi

cd $GOPATH/src/$BEAT_PATH

# Files must be copied before before-build calls to allow modifications on the config files

PREFIX=/build

# Copy fields.yml
cp fields.yml $PREFIX/fields.yml

# linux
cp $BEAT_NAME.yml $PREFIX/$BEAT_NAME-linux.yml
chmod 0600 $PREFIX/$BEAT_NAME-linux.yml
chmod 0600 $PREFIX/$BEAT_NAME-linux-386.yml || true
cp $BEAT_NAME.reference.yml $PREFIX/$BEAT_NAME-linux.reference.yml
rm -rf $PREFIX/modules.d-linux
cp -r modules.d/ $PREFIX/modules.d-linux || true
[ -d "$PREFIX/modules.d-linux" ] && chmod 0755 $PREFIX/modules.d-linux

# darwin
cp $BEAT_NAME.yml $PREFIX/$BEAT_NAME-darwin.yml
chmod 0600 $PREFIX/$BEAT_NAME-darwin.yml
cp $BEAT_NAME.reference.yml $PREFIX/$BEAT_NAME-darwin.reference.yml
rm -rf $PREFIX/modules.d-darwin
cp -r modules.d/ $PREFIX/modules.d-darwin || true
[ -d "$PREFIX/modules.d-darwin" ] && chmod 0755 $PREFIX/modules.d-darwin

# win
cp $BEAT_NAME.yml $PREFIX/$BEAT_NAME-win.yml
chmod 0600 $PREFIX/$BEAT_NAME-win.yml
cp $BEAT_NAME.reference.yml $PREFIX/$BEAT_NAME-win.reference.yml
rm -rf $PREFIX/modules.d-win
cp -r modules.d/ $PREFIX/modules.d-win || true
[ -d "$PREFIX/modules.d-win" ] && chmod 0755 $PREFIX/modules.d-win

# Runs beat specific tasks which should be done before building
PREFIX=$PREFIX make before-build

# Add data to the home directory
mkdir -p $PREFIX/homedir
make install-home HOME_PREFIX=$PREFIX/homedir LICENSE_FILE=${LICENSE_FILE}

if [ -n "BUILDID" ]; then
    echo "$BUILDID" > $PREFIX/homedir/.build_hash.txt
fi

# Append doc versions to package.yml
cat ${ES_BEATS}/libbeat/docs/version.asciidoc >> ${PREFIX}/package.yml

# Make variable naming of doc-branch compatible with gotpl. Generate and copy README.md into homedir
# Add " to the version as gotpl interprets 6.0 as 6
sed -i -e 's/:doc-branch: \(.*\)/doc_branch: "\1" /g' ${PREFIX}/package.yml

# Create README file
/go/bin/gotpl /templates/readme.md.j2 < ${PREFIX}/package.yml > ${PREFIX}/homedir/README.md
