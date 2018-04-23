PROJECT=beats-preference-pane.xcodeproj
CONFIGURATION?=Release
CODE_SIGN_IDENTITY?=''
CODE_SIGNING_REQUIRED?=NO

default: pref-pane

.PHONY: pref-pane

pref-pane:
	xcodebuild build -project $(PROJECT) -alltargets -configuration $(CONFIGURATION) CODE_SIGN_IDENTITY=$(CODE_SIGN_IDENTITY) CODE_SIGNING_REQUIRED=$(CODE_SIGNING_REQUIRED)

clean:
	rm -rf build/

