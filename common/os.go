package common

import (
	"fmt"
	"io/ioutil"
	"os"
)

// WriteFileAtomic writes to newBytes to filePath.
// Guaranteed not to lose *both* oldBytes and newBytes,
// (assuming that the OS is perfect)
func WriteFileAtomic(filePath string, newBytes []byte, mode os.FileMode) error {
	// If a file already exists there, copy to filePath+".bak" (overwrite anything)
	if _, err := os.Stat(filePath); !os.IsNotExist(err) {
		fileBytes, err := ioutil.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("Could not read file %v. %v", filePath, err)
		}
		err = ioutil.WriteFile(filePath+".bak", fileBytes, mode)
		if err != nil {
			return fmt.Errorf("Could not write file %v. %v", filePath+".bak", err)
		}
	}
	// Write newBytes to filePath.new
	err := ioutil.WriteFile(filePath+".new", newBytes, mode)
	if err != nil {
		return fmt.Errorf("Could not write file %v. %v", filePath+".new", err)
	}
	// Move filePath.new to filePath
	err = os.Rename(filePath+".new", filePath)
	return err
}
