package util

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

func ReadIntFromFile(path string) (value int, err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return -1, err
	}
	text := string(data)
	if len(text) <= 0 {
		return 0, errors.New(fmt.Sprintf("File is empty: %s", path))
	}
	text = strings.TrimSpace(text)
	value, err = strconv.Atoi(text)
	return value, err
}

// WriteIntToFile write a single integer to a file.go path
func WriteIntToFile(value int, path string) error {
	evaluatedPath, err := filepath.EvalSymlinks(path)
	if len(evaluatedPath) > 0 && err == nil {
		path = evaluatedPath
	}
	valueAsString := fmt.Sprintf("%d", value)
	err = ioutil.WriteFile(path, []byte(valueAsString), 644)
	return err
}
