package goracle

//go:generate bash -c "echo 3.1.4>odpi-version; set -x; curl -L https://github.com/oracle/odpi/archive/v$(cat odpi-version).tar.gz | tar xzvf - odpi-$(cat odpi-version)/{embed,include,src,CONTRIBUTING.md,LICENSE.md,README.md} && rm -rf odpi && mv odpi-$(cat odpi-version) odpi; rm -f odpi-version"

// Version of this driver
const Version = "v2.15.3"
