// +build !darwin

package file_integrity

// GetFileOrigin is not supported in this platform and always returns an empty
// list and no error.
func GetFileOrigin(fileName string) ([]string, error) {
	return nil, nil
}
