package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestUpload(t *testing.T) {
	globFiles, err := filepath.Glob("temp/*")

	if err != nil {
		t.Fatal(err)
	}

	for i, f := range globFiles {
		globFiles[i] = strings.Replace(f, "\\", "/", -1)
	}

	err = Upload("http://localhost:9090/upload", "abcdefg", globFiles)

	if err != nil {
		t.Fatal(err)
	}
}
