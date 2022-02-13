package main

import (
	"bytes"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	qt "github.com/frankban/quicktest"
)

func TestFetchPackage(t *testing.T) {
	c := qt.New(t)
	outputDir := t.TempDir()

	c.Assert(fetchPackage("is-sorted", "1.0.5", outputDir), qt.IsNil)

	b, err := ioutil.ReadFile(filepath.Join(outputDir, "npmpackage.json"))

	c.Assert(err, qt.IsNil)
	c.Assert(bytes.Contains(b, []byte("is-sorted")), qt.IsTrue)

	packageDir, err := os.ReadDir(filepath.Join(outputDir, "package"))

	c.Assert(err, qt.IsNil)
	c.Assert(len(packageDir), qt.Equals, 6)
}
