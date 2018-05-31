PROJECT=beats-preference-pane.xcodeproj
CONFIGURATION?=Release
SIGN_IDENTITY_APP?=
SIGN_IDENTITY_INSTALLER?=
PACKAGE_NAME?=BeatsPrefPane.pkg
INSTALL_DIR=/Library/PreferencePanes
IDENTIFIER?=co.elastic.beats.preference-pane
VERSION?=1.0.0

default: build

build:
	xcodebuild build -project $(PROJECT) -alltargets -configuration $(CONFIGURATION) CODE_SIGN_IDENTITY="" CODE_SIGNING_REQUIRED=NO
	echo version="${VERSION}"
	test -n '$(SIGN_IDENTITY_APP)' && \
	    codesign -s '$(SIGN_IDENTITY_APP)' --force --keychain '$(KEYCHAIN)' --timestamp \
	    build/$(CONFIGURATION)/*.prefPane/Contents/MacOS/helper && \
	    codesign -s '$(SIGN_IDENTITY_APP)' --force --keychain '$(KEYCHAIN)' --timestamp \
	    build/$(CONFIGURATION)/*.prefPane

.PHONY: clean pkg

pkg: $(PACKAGE_NAME)

$(PACKAGE_NAME): build
	mkdir -p build/root/$(INSTALL_DIR)
	cp -a build/$(CONFIGURATION)/Beats.prefPane build/root/$(INSTALL_DIR)/
	SIGN_ARGS='' ;\
        test -z '$(SIGN_IDENTITY_INSTALLER)' || SIGN_ARGS="--sign $(SIGN_IDENTITY_INSTALLER)" ;\
	pkgbuild --root build/root/ \
	    --identifier $(IDENTIFIER) \
	    --version '$(VERSION)' \
	    $$SIGN_ARGS --keychain "$(KEYCHAIN)" --timestamp \
	    $(PACKAGE_NAME)

clean:
	rm -rf build/ $(PACKAGE_NAME)
