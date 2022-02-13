package lib

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

func FetchPackage(s string) (NpmPackage, error) {
	var npmp NpmPackage
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://registry.npmjs.org/%s", s), nil)
	if err != nil {
		return npmp, err
	}
	req.Header.Set("Accept", "application/vnd.npm.install-v1+json")

	r, err := client.Do(req)
	if err != nil {
		return npmp, err
	}

	defer r.Body.Close()

	err = json.NewDecoder(r.Body).Decode(&npmp)
	if err == io.EOF {
		err = nil
	}

	return npmp, err
}

func FetchPackageVersion(pack, version string) (Version, error) {
	version = normalizeSemver(version)
	npmpkg, err := FetchPackage(pack)
	if err != nil {
		return Version{}, err
	}

	npmv, found := npmpkg.Versions.ByVersion(version)
	if !found {
		return npmv, fmt.Errorf("version %q not found for package %q", version, pack)
	}
	return npmv, nil
}

type Dependencies []Dependency

func (vs *Dependencies) UnmarshalJSON(b []byte) error {
	var m map[string]string
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}

	for k, v := range m {
		d := Dependency{
			Name:         k,
			VersionRange: v,
		}

		*vs = append(*vs, d)
	}

	vsv := *vs
	sort.Slice(vsv, func(i, j int) bool {
		vi, vj := vsv[i], vsv[j]
		return vi.Name < vj.Name
	})

	return nil
}

type Dependency struct {
	Name         string
	VersionRange string
}

type Dist struct {
	ShaSum  string `json:"shasum"`
	Tarball string `json:"tarball"`
}

type DistTags struct {
	Latest string
}

func (tags *DistTags) UnmarshalJSON(b []byte) error {
	var m map[string]string
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	tags.Latest = normalizeSemver(m["latest"])
	return nil
}

type NpmPackage struct {
	Name     string   `json:"name"`
	DistTags DistTags `json:"dist-tags"`
	Versions Versions `json:"versions"`
}

type Version struct {
	Name         string       `json:"name"`
	Version      string       `json:"version"`
	Dependencies Dependencies `json:"dependencies"`
	Dist         Dist         `json:"dist"`
}

type Versions []Version

func (vs Versions) ByVersion(v string) (ver Version, found bool) {
	for _, ver = range vs {
		if ver.Version == v {
			return ver, true
		}
	}
	return
}

func (vs *Versions) UnmarshalJSON(b []byte) error {
	var m map[string]Version
	err := json.Unmarshal(b, &m)
	if err != nil {
		return err
	}
	for _, version := range m {
		version.Version = normalizeSemver(version.Version)
		*vs = append(*vs, version)
	}

	vsv := *vs
	sort.Slice(vsv, func(i, j int) bool {
		vi, vj := vsv[i], vsv[j]
		cmp := semver.Compare(vi.Version, vj.Version)
		if cmp != 0 {
			return cmp < 0
		}
		return vi.Version < vj.Version
	})
	return nil
}

func DownloadTarballAndUnpack(dist Dist, outputDir string) (err error) {
	f := &bytes.Buffer{}

	resp, err := http.Get(dist.Tarball)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	h := sha1.New()
	out := io.MultiWriter(f, h)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	shasumFile := hex.EncodeToString(h.Sum(nil)[:])
	if shasumFile != dist.ShaSum {
		return errors.New("shasum mismatch")
	}

	return untar(outputDir, f)
}

func untar(dst string, r io.Reader) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		switch {
		case err == io.EOF:
			return nil
		case err != nil:
			return err
		case header == nil:
			continue
		}

		target := filepath.Join(dst, header.Name)

		switch header.Typeflag {
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, 0o755); err != nil {
					return err
				}
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}

			f, err := os.Create(target)
			if err != nil {
				return err
			}

			if _, err := io.Copy(f, tr); err != nil {
				return err
			}
			f.Close()
		}
	}
}

func normalizeSemver(s string) string {
	// Make the version Go semver compatible.
	if !strings.HasPrefix(s, "v") {
		s = "v" + s
	}
	return s
}
