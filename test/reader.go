package test_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"testing"
)

// LoadBytes is helper to load file contents from testdata directory
func LoadBytes(t *testing.T, name string) []byte {
	return loadBytes(t, fmt.Sprintf("testdata/%v", name), 2)
}

func loadBytes(t *testing.T, name string, callDepth int) []byte {
	_, b, _, _ := runtime.Caller(callDepth)
	basepath := filepath.Dir(b)

	path := filepath.Join(basepath, name) // relative path
	bytes, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}
