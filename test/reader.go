package test_test

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// LoadJSON is helper to load JSON file contents from testdata directory into target struct/slice
func LoadJSON(t *testing.T, filename string, targe interface{}) {
	b := loadBytes(t, fmt.Sprintf("testdata/%v", filename), 2)

	if err := json.Unmarshal(b, &targe); err != nil {
		t.Fatal(fmt.Errorf("test_test.LoadJSON failure: %w", err))
	}
}

// LoadBytes is helper to load file contents from testdata directory
func LoadBytes(t *testing.T, name string) []byte {
	return loadBytes(t, fmt.Sprintf("testdata/%v", name), 2)
}

func loadBytes(t *testing.T, name string, callDepth int) []byte {
	_, b, _, _ := runtime.Caller(callDepth)
	basepath := filepath.Dir(b)

	path := filepath.Join(basepath, name) // relative path
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return bytes
}

type ReadResult struct {
	Read []byte
	Err  error
}

type WriteResult struct {
	N   int
	Err error
}

type MockReaderWriter struct {
	Reads      []ReadResult
	Writes     []WriteResult
	readIndex  int
	writeIndex int
}

func (m *MockReaderWriter) Read(p []byte) (n int, err error) {
	r := m.Reads[m.readIndex]
	m.readIndex = m.readIndex + 1

	if r.Err != nil {
		return len(r.Read), err
	}

	n = copy(p, r.Read)
	return n, nil
}

func (m *MockReaderWriter) Write(p []byte) (n int, err error) {
	w := m.Writes[m.writeIndex]
	m.writeIndex = m.writeIndex + 1
	return w.N, w.Err
}
