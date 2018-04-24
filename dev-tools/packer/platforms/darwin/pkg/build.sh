#!/bin/bash

BASEDIR=$(cd "$(dirname "$0")"; pwd)
PACKERDIR=${BASEDIR}/../../../

die() {
    echo "Error: $@" >&2
    exit 1
}

test "$(uname -s)" = Darwin || die "Must be run in macOS"

FAIL=0
for bin in gotpl markdown pkgbuild productbuild hdiutil codesign xcodebuild
do
    which -s "$bin" || {
        echo "Required command '$bin' not found in PATH" >&2
        FAIL=1
    }
done
test $FAIL -ne 0 && die "Dependencies not met."

# Expecting some env vars to be set
for var in ARCH BUILD_DIR SNAPSHOT
do
    eval value=\$$var
    test -z "$value" && {
        FAIL=1
        echo "$var environment variable not set" >&2
    }
done
test $FAIL -ne 0 && die "Missing environment variables"

SIGN_IDENTITY_INSTALLER=$(security find-identity -v | grep 'Developer ID Installer' | sort -u | grep -o '[0-9A-F]\{40\}' | head -1)
SIGN_IDENTITY_APP=$(security find-identity -v | grep 'Developer ID Application' | sort -u | grep -o '[0-9A-F]\{40\}' | head -1)

test -n "SIGN_IDENTITY_INSTALLER" || die "Installer certificate not found"
test -n "SIGN_IDENTITY_APP" || die "Codesigning certificate not found"
export SIGN_IDENTITY_INSTALLER SIGN_IDENTITY_APP

test -f "${BUILD_DIR}/package.yml" || die "package.yml not found in BUILD_DIR"
ARCH_FILE="${PACKERDIR}/archs/$ARCH.yml"
test -f "$ARCH_FILE" || die "$ARCH_FILE not found (check ARCH environment variable)"

TMPDIR=$(mktemp -d)
test "$?" -ne 0 && die "Failed creating temporary directory"
echo "Building in directory $TMPDIR"

cat "${BUILD_DIR}/package.yml" "$ARCH_FILE" "$BASEDIR/base_conf.yml" > "${TMPDIR}/conf.yml" || die "Failed generating conf.yml"

if [ "$SNAPSHOT" = "yes" ]; then
    echo 'snapshot: "-SNAPSHOT"' >> "${TMPDIR}/conf.yml"
else
    echo 'snapshot: ""' >> "${TMPDIR}/conf.yml"
fi

echo 'Building preference-pane'
make -e CODE_SIGNING_REQUIRED=YES -C "${PACKERDIR}/platforms/darwin/preference-pane" clean build pkg || die "Build of preference-pane failed"
cp -a "${PACKERDIR}/platforms/darwin/preference-pane/BeatsPrefPane.pkg" "${TMPDIR}/" || die "Preference pane package not found"

pushd "${BASEDIR}/templates"
for dir in $(find . -type d); do
    mkdir -p "$TMPDIR/$dir" || FAIL=1
done
test $FAIL -ne 0 && die "Failed creating directory tree"

for template in $(find . -type f); do
    if echo "$template" | grep -q '\.j2$'; then
        TARGET=$(echo "$template" | sed 's/\.j2$//')
        gotpl "$template" < "$TMPDIR/conf.yml" > "$TMPDIR/$TARGET" || {
            echo "Failed generating '$TARGET' from '$template'" >&2
            FAIL=1
        }
        if grep -q '<no value>' "$TMPDIR/$TARGET" ; then
            echo "Unbound template variable in '$template'" >&2
            FAIL=1
        fi
    else
        cp "$template" "$TMPDIR/$template" || FAIL=1
    fi
done
popd
test $FAIL -ne 0 && die "Unable to generate files"

pushd "$TMPDIR"
    "$BASEDIR/internal_build.sh" || FAIL=1
popd

test -z "$KEEP_TMP" -o "$KEEP_TMP" = 0 && rm -rf "$TMPDIR"

test $FAIL -eq 0 || die "Failed generating package"
