package main

import (
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/bep/fetch-npm-package/internal/lib"
)

func main() {
	if len(os.Args) != 4 {
		log.Fatal("usage: fetch-npm-package <package> <version> <output-dir>")
	}

	packageName := os.Args[1]
	version := lib.NormalizeSemver(os.Args[2])
	outputDir := os.Args[3]
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		log.Fatal("output directory does not exist")
	}
	packageDir := filepath.Join(outputDir, "package")
	if _, err := os.Stat(packageDir); err == nil {
		log.Fatalf("package dir %q already exists", packageDir)
	}

	log.Printf("Fetching package %s@%s", packageName, version)

	v, err := lib.FetchPackageVersion(packageName, version)
	if err != nil {
		log.Fatal(err)
	}

	if err := lib.DownloadTarballAndUnpack(v.Dist, outputDir); err != nil {
		log.Fatal(err)
	}

	meta, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	err = ioutil.WriteFile(filepath.Join(outputDir, "npmpackage.json"), meta, 0644)
	if err != nil {
		log.Fatal(err)
	}
}
