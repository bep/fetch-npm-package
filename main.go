package main

import (
	"encoding/json"
	"fmt"
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

	if err := fetchPackage(os.Args[1], os.Args[2], os.Args[3]); err != nil {
		log.Fatal(err)
	}
}

func fetchPackage(packageName, version, outputDir string) error {
	if _, err := os.Stat(outputDir); os.IsNotExist(err) {
		return fmt.Errorf("output directory %q does not exist", outputDir)
	}
	packageDir := filepath.Join(outputDir, "package")
	if _, err := os.Stat(packageDir); err == nil {
		return fmt.Errorf("package dir %q already exists", packageDir)
	}

	log.Printf("Fetching package %s@%s", packageName, version)

	v, err := lib.FetchPackageVersion(packageName, version)
	if err != nil {
		return err
	}

	if err := lib.DownloadTarballAndUnpack(v.Dist, outputDir); err != nil {
		log.Fatal(err)
	}

	meta, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filepath.Join(outputDir, "npmpackage.json"), meta, 0644)
}
