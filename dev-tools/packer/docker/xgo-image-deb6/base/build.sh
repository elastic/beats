#!/bin/bash
#
# Contains the main cross compiler, that individually sets up each target build
# platform, compiles all the C dependencies, then build the requested executable
# itself.
#
# Usage: build.sh <import path>
#
# Needed environment variables:
#   REPO_REMOTE - Optional VCS remote if not the primary repository is needed
#   REPO_BRANCH - Optional VCS branch to use, if not the master branch
#   DEPS        - Optional list of C dependency packages to build
#   PACK        - Optional sub-package, if not the import path is being built
#   OUT         - Optional output prefix to override the package name
#   FLAG_V      - Optional verbosity flag to set on the Go builder
#   FLAG_RACE   - Optional race flag to set on the Go builder

# Download the canonical import path (may fail, don't allow failures beyond)

SRC_FOLDER=$SOURCE
DST_FOLDER=$GOPATH/src/$1

if [ $1 = "github.com/elastic/beats" ]; then
        WORKING_DIRECTORY=$GOPATH/src/$1
else
        WORKING_DIRECTORY=$GOPATH/src/`dirname $1`
fi

echo "Working directory=$WORKING_DIRECTORY"

if [ "$SOURCE" != "" ]; then
        mkdir -p ${DST_FOLDER}
        echo "Copying main source folder ${SRC_FOLDER} to folder ${DST_FOLDER}"
        rsync --exclude ".git"  --exclude "build/" -a ${SRC_FOLDER}/ ${DST_FOLDER}
else
        mkdir -p $GOPATH/src/`dirname $1`
        cd $GOPATH/src/`dirname $1`
        echo "Fetching main git repository $1 in folder $GOPATH/src/`dirname $1`"
        git clone https://$1.git
fi

set -e

cd $WORKING_DIRECTORY
export GOPATH=$GOPATH:`pwd`/Godeps/_workspace
export GO15VENDOREXPERIMENT=1

# Switch over the code-base to another checkout if requested
if [ "$REPO_REMOTE" != "" ]; then
  echo "Switching over to remote $REPO_REMOTE..."
  if [ -d ".git" ]; then
    git remote set-url origin $REPO_REMOTE
    git pull
  elif [ -d ".hg" ]; then
    echo -e "[paths]\ndefault = $REPO_REMOTE\n" >> .hg/hgrc
    hg pull
  fi
fi

if [ "$REPO_BRANCH" != "" ]; then
  echo "Switching over to branch $REPO_BRANCH..."
  if [ -d ".git" ]; then
    git checkout $REPO_BRANCH
  elif [ -d ".hg" ]; then
    hg checkout $REPO_BRANCH
  fi
fi

# Download all the C dependencies
echo "Fetching dependencies..."
mkdir -p /deps
DEPS=($DEPS) && for dep in "${DEPS[@]}"; do
  echo Downloading $dep
  if [ "${dep##*.}" == "tar" ]; then wget -q $dep -O - | tar -C /deps -x; fi
  if [ "${dep##*.}" == "gz" ]; then wget -q $dep -O - | tar -C /deps -xz; fi
  if [ "${dep##*.}" == "bz2" ]; then wget -q $dep -O - | tar -C /deps -xj; fi
done

# Configure some global build parameters
NAME=`basename $1/$PACK`
if [ "$OUT" != "" ]; then
  NAME=$OUT
fi


if [ "$FLAG_V" == "true" ]; then V=-v; fi
if [ "$FLAG_RACE" == "true" ]; then R=-race; fi
if [ "$STATIC" == "true" ]; then LDARGS=--ldflags\ \'-extldflags\ \"-static\"\'; fi

if [ -n $BEFORE_BUILD ]; then
	chmod +x /scripts/$BEFORE_BUILD
	echo "Execute /scripts/$BEFORE_BUILD ${1}"
	/scripts/$BEFORE_BUILD ${1}
fi

# Build for each platform individually
echo "Compiling $PACK for linux/amd64..."
HOST=x86_64-linux PREFIX=/usr/local $BUILD_DEPS /deps
GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go get -d ./$PACK
sh -c "GOOS=linux GOARCH=amd64 CGO_ENABLED=1 go build $V $R $LDARGS -o $NAME-linux-amd64$R ./$PACK"

echo "Compiling $PACK for linux/386..."
HOST=i686-linux PREFIX=/usr/local $BUILD_DEPS /deps
GOOS=linux GOARCH=386 CGO_ENABLED=1 go get -d ./$PACK
sh -c "GOOS=linux GOARCH=386 CGO_ENABLED=1 go build $V $LDARGS -o $NAME-linux-386 ./$PACK"

#echo "Compiling $PACK for linux/arm..."
#CC=arm-linux-gnueabi-gcc HOST=arm-linux PREFIX=/usr/local/arm $BUILD_DEPS /deps
#CC=arm-linux-gnueabi-gcc GOOS=linux GOARCH=arm CGO_ENABLED=1 GOARM=5 go get -d ./$PACK
#CC=arm-linux-gnueabi-gcc GOOS=linux GOARCH=arm CGO_ENABLED=1 GOARM=5 go build $V -o $NAME-linux-arm ./$PACK

#echo "Compiling $PACK for windows/amd64..."
#CC=x86_64-w64-mingw32-gcc HOST=x86_64-w64-mingw32 PREFIX=/usr/x86_64-w64-mingw32 $BUILD_DEPS /deps
#CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go get -d ./$PACK
#CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build  $V $R -o $NAME-windows-amd64$R.exe ./$PACK

#echo "Compiling $PACK for windows/386..."
#CC=i686-w64-mingw32-gcc HOST=i686-w64-mingw32 PREFIX=/usr/i686-w64-mingw32 $BUILD_DEPS /deps
#CC=i686-w64-mingw32-gcc GOOS=windows GOARCH=386 CGO_ENABLED=1 go get -d ./$PACK
#CC=i686-w64-mingw32-gcc GOOS=windows GOARCH=386 CGO_ENABLED=1 go build $V -o $NAME-windows-386.exe ./$PACK

#echo "Compiling $PACK for darwin/amd64..."
#CC=o64-clang HOST=x86_64-apple-darwin10 PREFIX=/usr/local $BUILD_DEPS /deps
#CC=o64-clang GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go get -d ./$PACK
#CC=o64-clang GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build $V $R -o $NAME-darwin-amd64$R ./$PACK

#echo "Compiling $PACK for darwin/386..."
#CC=o32-clang HOST=i386-apple-darwin10 PREFIX=/usr/local $BUILD_DEPS /deps
#CC=o32-clang GOOS=darwin GOARCH=386 CGO_ENABLED=1 go get -d ./$PACK
#CC=o32-clang GOOS=darwin GOARCH=386 CGO_ENABLED=1 go build $V -o $NAME-darwin-386 ./$PACK

# The binary files are the 2 last created files
echo "Moving binaries to host..."
ls -t | head -n 2
cp `ls -t | head -n 2` /build

echo "Build process completed"
