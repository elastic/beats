#!/bin/sh -x

echo 'Yes, do as I say!' | echo apt-get install -y --force-yes --no-install-recommends $*
