// +build !linux

package packages

func listRPMPackages() ([]*Package, error) {
	return rpmPackagesByExec()
}
