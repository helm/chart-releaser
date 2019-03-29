// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package releaser

import (
	"context"
	"fmt"
	"github.com/helm/chart-releaser/pkg/config"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/helm/chart-releaser/pkg/github"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

// GitHub contains the functions necessary for interacting with GitHub release
// objects
type GitHub interface {
	CreateRelease(ctx context.Context, input *github.Release) error
	GetRelease(ctx context.Context, tag string) (*github.Release, error)
}

type Releaser struct {
	config *config.Options
	github GitHub
}

func NewReleaser(config *config.Options, github GitHub) *Releaser {
	return &Releaser{
		config: config,
		github: github,
	}
}

//UpdateIndexFile index.yaml file for a give git repo
func (r *Releaser) UpdateIndexFile() error {
	var indexFile = &repo.IndexFile{}

	if _, err := os.Stat(r.config.IndexPath); err == nil {
		fmt.Printf("====> Using existing index at %s\n", r.config.IndexPath)
		indexFile, err = repo.LoadIndexFile(r.config.IndexPath)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("====> UpdateIndexFile new index at %s\n", r.config.IndexPath)
		indexFile = repo.NewIndexFile()
	}

	chartPackages, err := ioutil.ReadDir(r.config.PackagePath)
	if err != nil {
		return err
	}

	for _, chartPackage := range chartPackages {
		tag := strings.TrimSuffix(chartPackage.Name(), filepath.Ext(chartPackage.Name()))

		release, err := r.github.GetRelease(context.TODO(), tag)
		if err != nil {
			return err
		}

		for _, asset := range release.Assets {
			downloadUrl, _ := url.Parse(asset.URL)
			name := path.Base(downloadUrl.Path)
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			tagParts := r.splitPackageNameAndVersion(baseName)
			packageName, packageVersion := tagParts[0], tagParts[1]
			fmt.Printf("====> Found %s-%s.tgz\n", packageName, packageVersion)
			if _, err := indexFile.Get(packageName, packageVersion); err != nil {
				if err := r.addToIndexFile(indexFile, downloadUrl.String()); err != nil{
					return err
				}
				break
			}
		}
	}

	fmt.Printf("--> Updating index %s\n", r.config.IndexPath)
	indexFile.SortEntries()
	return indexFile.WriteFile(r.config.IndexPath, 0644)
}

func (r *Releaser) splitPackageNameAndVersion(pkg string) []string {
	delimIndex := strings.LastIndex(pkg, "-")
	return []string{pkg[0:delimIndex], pkg[delimIndex+1:]}
}

func (r *Releaser) addToIndexFile(indexFile *repo.IndexFile, url string) error {
	arch := path.Join(r.config.PackagePath, path.Base(url))

	// extract chart metadata
	fmt.Printf("====> Extracting chart metadata from %s\n", arch)
	c, err := chartutil.Load(arch)
	if err != nil {
		return errors.Wrapf(err, "%s is not a helm chart package", arch)
	}
	// calculate hash
	fmt.Printf("====> Calculating Hash for %s\n", arch)
	hash, err := provenance.DigestFile(arch)
	if err != nil {
		return err
	}

	// remove url name from url as helm's index library
	// adds it in during .Add
	// there should be a better way to handle this :(
	s := strings.Split(url, "/")
	s = s[:len(s)-1]

	// Add to index
	indexFile.Add(c.Metadata, path.Base(arch), strings.Join(s, "/"), hash)
	return nil
}

// CreateReleases finds and uploads helm chart packages to github
func (r *Releaser) CreateReleases() error {
	packages, err := r.getListOfPackages(r.config.PackagePath)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		return errors.Errorf("No charts found at %s.\n", r.config.PackagePath)
	}

	for _, p := range packages {
		baseName := strings.TrimSuffix(p, filepath.Ext(p))

		release := &github.Release{
			Name: baseName,
			Assets: []*github.Asset{
				{Path: p},
			},
		}
		provFile := fmt.Sprintf("%s.prov", p)
		if _, err := os.Stat(provFile); err == nil {
			asset := &github.Asset{Path: provFile}
			release.Assets = append(release.Assets, asset)
		}

		// TODO add changelog
		if err := r.github.CreateRelease(context.TODO(), release); err != nil {
			return errors.Wrap(err, "error creating GitHub release")
		}
	}

	return nil
}

func (r *Releaser) getListOfPackages(dir string) ([]string, error) {
	return filepath.Glob(filepath.Join(dir, "*.tgz"))
}
