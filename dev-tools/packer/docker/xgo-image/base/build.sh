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
#   TARGETS        - Comma separated list of build targets to compile for



# Download the canonical import path (may fail, don't allow failures beyond)
SRC_FOLDER=$SOURCE

BEAT_PATH=$1
DST_FOLDER=`dirname $GOPATH/src/$BEAT_PATH`
GIT_REPO=$BEAT_PATH

if [ "$PUREGO" == "yes" ]; then
    CGO_ENABLED=0
else
    CGO_ENABLED=1
fi

# If it is an official beat, libbeat is not vendored, need special treatment
if [[ $GIT_REPO == "github.com/elastic/beats"* ]]; then
    echo "Overwrite directories because official beat"
    DST_FOLDER=$GOPATH/src/github.com/elastic/beats
    GIT_REPO=github.com/elastic/beats
fi

# It is assumed all dependencies are inside the working directory
# The working directory is the parent of the beat directory
WORKING_DIRECTORY=$DST_FOLDER

echo "Working directory=$WORKING_DIRECTORY"

if [ "$SOURCE" != "" ]; then
        mkdir -p ${DST_FOLDER}
        echo "Copying main source folder ${SRC_FOLDER} to folder ${DST_FOLDER}"
        rsync --exclude ".git"  --exclude "build/" -a ${SRC_FOLDER}/ ${DST_FOLDER}
else
        mkdir -p $GOPATH/src/${GIT_REPO}
        cd $GOPATH/src/${GIT_REPO}
        echo "Fetching main git repository ${GIT_REPO} in folder $GOPATH/src/${GIT_REPO}"
        git clone https://${GIT_REPO}.git
fi

set -e

cd $WORKING_DIRECTORY

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
BUILD_DEPS=/build_deps.sh
DEPS_FOLDER=/deps
LIST_DEPS=""
mkdir -p $DEPS_FOLDER
DEPS=($DEPS) && for dep in "${DEPS[@]}"; do
  dep_filename=${dep##*/}
  echo "Downloading $dep to $DEPS_FOLDER/$dep_filename"
  wget -q $dep --directory-prefix=$DEPS_FOLDER
  dep_name=$(tar --list --no-recursion --file=$DEPS_FOLDER/$dep_filename  --exclude="*/*" | sed 's/\///g')
  LIST_DEPS="${LIST_DEPS} ${dep_name}"
  if [ "${dep_filename##*.}" == "tar" ]; then tar -xf  $DEPS_FOLDER/$dep_filename --directory $DEPS_FOLDER/  ; fi
  if [ "${dep_filename##*.}" == "gz"  ]; then tar -xzf $DEPS_FOLDER/$dep_filename --directory $DEPS_FOLDER/  ; fi
  if [ "${dep_filename##*.}" == "bz2" ]; then tar -xj  $DEPS_FOLDER/$dep_filename --directory $DEPS_FOLDER/  ; fi
done

# Configure some global build parameters
NAME=${PACK}
if [ "$OUT" != "" ]; then
  NAME=$OUT
fi


if [ "$FLAG_V" == "true" ]; then V=-v; fi
if [ "$FLAG_RACE" == "true" ]; then R=-race; fi
if [ "$STATIC" == "true" ]; then LDARGS=--ldflags\ \'-extldflags\ \"-static\"\'; fi

if [ -n $BEFORE_BUILD ]; then
	chmod +x /scripts/$BEFORE_BUILD
	echo "Execute /scripts/$BEFORE_BUILD ${BEAT_PATH}"
	/scripts/$BEFORE_BUILD ${BEAT_PATH}
fi


# If no build targets were specified, inject a catch all wildcard
if [ "$TARGETS" == "" ]; then
  TARGETS="./."
fi


built_targets=0
for TARGET in $TARGETS; do
	# Split the target into platform and architecture
	XGOOS=`echo $TARGET | cut -d '/' -f 1`
	XGOARCH=`echo $TARGET | cut -d '/' -f 2`

	# Check and build for Linux targets
	if ([ $XGOOS == "." ] || [ $XGOOS == "linux" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]); then
		echo "Compiling $PACK for linux/amd64..."
		HOST=x86_64-linux PREFIX=/usr/local $BUILD_DEPS /deps $LIST_DEPS
		export PKG_CONFIG_PATH=/usr/aarch64-linux-gnu/lib/pkgconfig

		GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go get -d ./$PACK
		sh -c "GOOS=linux GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build $V $R $LDARGS -o $NAME-linux-amd64$R ./$PACK"
		built_targets=$((built_targets+1))
	fi
	if ([ $XGOOS == "." ] || [ $XGOOS == "linux" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "386" ]); then
		echo "Compiling $PACK for linux/386..."
		CFLAGS=-m32 CXXFLAGS=-m32 LDFLAGS=-m32 HOST=i686-linux PREFIX=/usr/local $BUILD_DEPS /deps $LIST_DEPS
		GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go get -d ./$PACK
		sh -c "GOOS=linux GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build $V $R $LDARGS -o $NAME-linux-386$R ./$PACK"
		built_targets=$((built_targets+1))
	fi
	if ([ $XGOOS == "." ] || [ $XGOOS == "linux" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "arm" ]); then
		echo "Compiling $PACK for linux/arm..."
		CC=arm-linux-gnueabi-gcc CXX=rm-linux-gnueabi-g++ HOST=arm-linux PREFIX=/usr/local/arm $BUILD_DEPS /deps $LIST_DEPS

		CC=arm-linux-gnueabi-gcc CXX=rm-linux-gnueabi-g++ GOOS=linux GOARCH=arm CGO_ENABLED=${CGO_ENABLED} GOARM=5 go get -d ./$PACK
		CC=arm-linux-gnueabi-gcc CXX=rm-linux-gnueabi-g++ GOOS=linux GOARCH=arm CGO_ENABLED=${CGO_ENABLED} GOARM=5 go build $V -o $NAME-linux-arm ./$PACK
		built_targets=$((built_targets+1))
	fi

	# Check and build for Windows targets
	if [ $XGOOS == "." ] || [[ $XGOOS == windows* ]]; then
		# Split the platform version and configure the Windows NT version
		PLATFORM=`echo $XGOOS | cut -d '-' -f 2`
		if [ "$PLATFORM" == "" ] || [ "$PLATFORM" == "." ] || [ "$PLATFORM" == "windows" ]; then
		  PLATFORM=4.0 # Windows NT
		fi

	    MAJOR=`echo $PLATFORM | cut -d '.' -f 1`
		if [ "${PLATFORM/.}" != "$PLATFORM" ] ; then
		  MINOR=`echo $PLATFORM | cut -d '.' -f 2`
		fi
		CGO_NTDEF="-D_WIN32_WINNT=0x`printf "%02d" $MAJOR``printf "%02d" $MINOR`"

		# Build the requested windows binaries
		if [ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]; then
			echo "Compiling $PACK for windows-$PLATFORM/amd64..."
			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ CFLAGS="$CGO_NTDEF" CXXFLAGS="$CGO_NTDEF" HOST=x86_64-w64-mingw32 PREFIX=/usr/x86_64-w64-mingw32 $BUILD_DEPS /deps $LIST_DEPS
			export PKG_CONFIG_PATH=/usr/x86_64-w64-mingw32/lib/pkgconfig

			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ GOOS=windows GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go get -d ./$PACK
			CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ GOOS=windows GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go build  $V $R -o $NAME-windows-amd64$R.exe ./$PACK
			built_targets=$((built_targets+1))
		fi

		if [ $XGOARCH == "." ] || [ $XGOARCH == "386" ]; then
			echo "Compiling $PACK for windows-$PLATFORM/386..."
			CC=i686-w64-mingw32-gcc CXX=i686-w64-mingw32-g++ CFLAGS="$CGO_NTDEF" CXXFLAGS="$CGO_NTDEF" HOST=i686-w64-mingw32 PREFIX=/usr/i686-w64-mingw32 $BUILD_DEPS /deps $LIST_DEPS
			export PKG_CONFIG_PATH=/usr/i686-w64-mingw32/lib/pkgconfig

			CC=i686-w64-mingw32-gcc CXX=i686-w64-mingw32-g++ GOOS=windows GOARCH=386 CGO_ENABLED=${CGO_ENABLED} CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go get -d ./$PACK
			CC=i686-w64-mingw32-gcc CXX=i686-w64-mingw32-g++ GOOS=windows GOARCH=386 CGO_ENABLED=${CGO_ENABLED} CGO_CFLAGS="$CGO_NTDEF" CGO_CXXFLAGS="$CGO_NTDEF" go build $V -o $NAME-windows-386.exe ./$PACK
			built_targets=$((built_targets+1))
		fi
	fi

	# Check and build for OSX targets
	if ([ $XGOOS == "." ] || [ $XGOOS == "darwin" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "amd64" ]); then
		echo "Compiling $PACK for darwin/amd64..."
		CC=o64-clang CXX=o64-clang++ HOST=x86_64-apple-darwin10 PREFIX=/usr/local $BUILD_DEPS /deps $LIST_DEPS
		CC=o64-clang CXX=o64-clang++ GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go get -d ./$PACK
		CC=o64-clang CXX=o64-clang++ GOOS=darwin GOARCH=amd64 CGO_ENABLED=${CGO_ENABLED} go build -ldflags=-s $V $R -o $NAME-darwin-amd64$R ./$PACK
		built_targets=$((built_targets+1))
	fi
	if ([ $XGOOS == "." ] || [ $XGOOS == "darwin" ]) && ([ $XGOARCH == "." ] || [ $XGOARCH == "386" ]); then
		echo "Compiling for darwin/386..."
		CC=o32-clang CXX=o32-clang++ HOST=i386-apple-darwin10 PREFIX=/usr/local $BUILD_DEPS /deps $LIST_DEPS
		CC=o32-clang CXX=o32-clang++ GOOS=darwin GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go get -d ./$PACK
		CC=o32-clang CXX=o32-clang++ GOOS=darwin GOARCH=386 CGO_ENABLED=${CGO_ENABLED} go build $V -o $NAME-darwin-386 ./$PACK
		built_targets=$((built_targets+1))
	fi
done


# The binary files are the last created files
echo "Moving $built_targets $PACK binaries to host folder..."
ls -t | head -n $built_targets
cp `ls -t | head -n $built_targets ` /build

echo "Build process completed"
