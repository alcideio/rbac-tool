package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"syscall"
)

// FileExists checks if specified file exists.
func FileExists(filename string) (bool, error) {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func WriteFile(outfile string, data string) error {
	if outfile == "-" {
		fmt.Println(data)
		return nil
	}

	if exist, err := FileExists(outfile); err == nil && exist {
		syscall.Unlink(outfile)
	}

	return ioutil.WriteFile(outfile, []byte(data), 0644)
}
