package tools

import (
	"os"
	"path/filepath"
)

// CreateDirWithPath create full path if dir isn't exist, except other error
func CreateDirWithPath(dir string) error {
	_, err := os.Stat(dir)
	if os.IsNotExist(err) {
		return os.MkdirAll(dir, os.ModePerm)
	}

	return err
}

// CreatePathWithFilepath checkout output whether exist
func CreatePathWithFilepath(fp string) error {
	_, err := os.Stat(fp)
	if os.IsExist(err) {
		return ErrFileExist
	} else {
		// create all save path, except filename
		path := filepath.Dir(fp)
		_, err2 := os.Stat(path)
		if !os.IsExist(err2) {
			if err := os.MkdirAll(path, os.ModePerm); err != nil {
				return err
			}
		}
	}
	return nil
}
