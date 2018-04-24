#!/bin/sh

set -e

. conf.sh

BASEDIR=$(cd "$(dirname "$0")"; pwd)

FILE_NAME="$BEAT-$VERSION-$ARCH"
PKG_NAME="$FILE_NAME.pkg"
DMG_NAME="$FILE_NAME.dmg"
INNER_NAME="internal-$FILE_NAME.pkg"
VENDOR_DIR="root/$INSTALL_PATH/$VENDOR"
BEAT_DIR="$VENDOR_DIR/$BEAT"
mkdir -p "$VENDOR_DIR"
TAR_NAME="$BEAT-$VERSION-darwin-$ARCH"
tar zxf "$BUILD_DIR/upload/$TAR_NAME".tar.gz
mv "$TAR_NAME" "$BEAT_DIR"

cp launchd-daemon.plist "$BEAT_DIR/$IDENTIFIER.plist"

chmod +x scripts/*
chmod +x dmg/*.app/Contents/MacOS/*

# move beat binary to bin/ subdir as in Linux
mkdir -p "$BEAT_DIR/bin"
mv "$BEAT_DIR/$BEAT" "$BEAT_DIR/bin/$BEAT"

# move configuration to /etc
mkdir -p "root/etc/$BEAT"
mv "$BEAT_DIR/"*.yml "root/etc/$BEAT"

# create logs directory
mkdir -p "root/var/log/$BEAT"

pkgbuild --root root \
    --scripts scripts \
    --component-plist component.plist \
    --identifier "$IDENTIFIER" \
    --version "$VERSION" \
    --sign "$SIGN_IDENTITY_INSTALLER" \
    --timestamp \
    "$INNER_NAME"

markdown < "$BEAT_DIR/README.md" > README.body.html
cat README.header.html README.body.html README.footer.html > README.html

cp "$BEAT_DIR/LICENSE.txt" LICENSE.txt

productbuild --distribution distribution.plist \
    --resources . \
    --package-path "BeatsPrefPane.pkg" \
    --package-path "$INNER_NAME" \
    --component-compression auto \
    --sign "$SIGN_IDENTITY_INSTALLER" \
    --timestamp \
    "$PKG_NAME"

spctl -av -t install "$PKG_NAME"

cp "$PKG_NAME" dmg/
codesign -s "$SIGN_IDENTITY_APP" \
    --timestamp \
    dmg/*.app

spctl -av -t exec dmg/*.app

hdiutil create \
    -volname "$BEAT $VERSION" \
    -srcfolder dmg \
    -ov \
    "$DMG_NAME"

codesign -s "$SIGN_IDENTITY_APP" \
    --timestamp \
    "$DMG_NAME"
spctl -av -t install "$DMG_NAME"

for artifact in "$PKG_NAME" "$DMG_NAME"; do
    shasum -a 512 "$artifact" > "$artifact".sha512
    cp "$artifact" "$artifact".sha512 "$BUILD_DIR/upload/"
done
