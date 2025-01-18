package tools

import "os"

// CreateDirWithPath create full path if dir isn't exist, except other error
func CreateDirWithPath(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}

	return err
}
