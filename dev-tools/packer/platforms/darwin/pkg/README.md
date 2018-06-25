Beats installer for macOS
=========================

This directory contains the scripts necessary to build an installer (.pkg file)
for macOS. The installer will be wrapped in a DMG file and signed with 
certificates issued by Apple.

Dependencies
------------

The following tools should already be installed in stock macOS or by XCode:

| Tool         | Use                         |
|--------------|-----------------------------|
| hdiutil      | Generate disk images (dmg)  |
| pkgbuild     | To build pkg files          |
| productbuild | Same                        |
| codesign     | Sign files and applications |

The following 3rd party tools need to be installed:

| Tool     | Source               | Use                         |
|----------|----------------------|-----------------------------|
| gotpl    | github.com/tsg/gotpl | Golang templates            |
| markdown | HomeBrew             | markdown to HTML conversion |


Parameters
----------

This parameters are required to be passed via environment variables:

| Variable      | Description                          |
|---------------|--------------------------------------|
| ARCH          | `amd64` or `386`                     |
| BUILD_DIR     | Output directory of the build process. Contains the `package.yml` file and `upload` directory. |
| SNAPSHOT      | `yes` or `no`                        |

Certificates
------------

Currently this script doesn't support building packages without code-signing them,
so valid signing code-certificates provided by Apple are required.

~~For convenience, this script expects the certificates inside its own keychain
file (trivial to create with `Keychain Access`).~~ Due to changes introduced
in macOS 10.12, it is not possible to automate the handling of keychains outside
a desktop session. To workaround this issue, the certificates are expected to be
readily available in one of the keychains of the user running the packaging
scripts.

The required contents of the keychain are:

- Code-signing certificate, called `Developer ID Application: SomeCompany (APPLE_ID_NNN)`.
- Private key for the code-signing certificate.
- Installer certificate, called `Developer ID Installer: SomeCompany (APPLE_ID_NNN)`.
- Private key for the installer certificate.
- Certification chain for both certificates. Currently this means:
  - Developer ID Certification Authority
  - Apple Root CA


Running
-------

    $ ./build.sh

If everything goes right, it will output a .pkg and dmg file to `${BUILD_DIR}/upload/macOS`

Debugging
---------

Add `-x` to the shebang line.

Pass `KEEP_TMP=1` to keep the temporary files.
