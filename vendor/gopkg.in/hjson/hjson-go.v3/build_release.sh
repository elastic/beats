#!/bin/bash

cd `dirname $0`
ROOT=$PWD/_dist

if [ -d "$ROOT" ]; then rm -rf $ROOT; fi

mkdir $ROOT

function build() {
  export GOOS=$1
  export GOARCH=$2

  echo build $GOOS $GOARCH
  OUT=$ROOT/${GOOS}_${GOARCH}
  mkdir $OUT
  cd $OUT
  go build github.com/hjson/hjson-go/hjson-cli
  if [[ $3 == "zip" ]]; then
    mv $OUT/hjson-cli.exe $OUT/hjson.exe
    zip -j ${OUT}.zip $OUT/*
  else
    mv $OUT/hjson-cli $OUT/hjson
    tar -czf ${OUT}.tar.gz -C $OUT .
  fi

}

# not all targets can be built on travis

# build android arm
build darwin 386
build darwin amd64
# build darwin arm
# build darwin arm64
build dragonfly amd64
build freebsd 386
build freebsd amd64
build freebsd arm
build linux 386
build linux amd64
build linux arm
build linux arm64
build linux mips64
build linux mips64le
build linux ppc64
build linux ppc64le
build netbsd 386
build netbsd amd64
build netbsd arm
build openbsd 386
build openbsd amd64
build openbsd arm
build plan9 386
build plan9 amd64
build solaris amd64
build windows 386 zip
build windows amd64 zip
