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
	ListReleases(ctx context.Context) ([]*github.Release, error)
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
	// if path doesn't end with index.yaml we can try and fix it
	if path.Base(r.config.Path) != "index.yaml" {
		// if path is a directory then add index.yaml
		if stat, err := os.Stat(r.config.Path); err == nil && stat.IsDir() {
			r.config.Path = path.Join(r.config.Path, "index.yaml")
			// otherwise error out
		} else {
			fmt.Printf("path (%s) should be a directory or a file called index.yaml\n", r.config.Path)
			os.Exit(1)
		}
	}

	var indexFile = &repo.IndexFile{}
	// Load up Index file (or create new one)
	if _, err := os.Stat(r.config.Path); err == nil {
		fmt.Printf("====> Using existing index at %s\n", r.config.Path)
		indexFile, err = repo.LoadIndexFile(r.config.Path)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("====> UpdateIndexFile new index at %s\n", r.config.Path)
		indexFile = repo.NewIndexFile()
	}

	releases, err := r.github.ListReleases(context.TODO())
	if err != nil {
		return err
	}

	fmt.Println("--> Checking for releases with helm chart packages")
	for _, rel := range releases {
		for _, asset := range rel.Assets {
			downloadUrl, _ := url.Parse(asset.URL)
			name := path.Base(downloadUrl.Path)
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			tagParts := r.splitPackageNameAndVersion(baseName)
			packageName, packageVersion := tagParts[0], tagParts[1]
			fmt.Printf("====> Found %s-%s.tgz\n", packageName, packageVersion)
			if _, err := indexFile.Get(packageName, packageVersion); err != nil {
				r.addToIndexFile(indexFile, downloadUrl.String())
			}
			break
		}
	}

	fmt.Printf("--> Updating index %s\n", r.config.Path)
	indexFile.SortEntries()
	return indexFile.WriteFile(r.config.Path, 0644)
}

func (r *Releaser) splitPackageNameAndVersion(pkg string) []string {
	delimIndex := strings.LastIndex(pkg, "-")
	return []string{pkg[0:delimIndex], pkg[delimIndex+1:]}
}

func (r *Releaser) addToIndexFile(indexFile *repo.IndexFile, url string) {
	// fetch package to temp url so we can extract metadata and stuff
	//dir, err := ioutil.TempDir("", "chart-releaser")
	//if err != nil {
	//	log.Fatal(err)
	//}
	//defer os.RemoveAll(dir)
	// TODO dir
	arch := path.Join(".", path.Base(url))

	// extract chart metadata
	fmt.Printf("====> Extracting chart metadata from %s\n", arch)
	c, err := chartutil.Load(arch)
	if err != nil {
		// weird, must not be a chart package
		fmt.Printf("====> %s is not a helm chart package\n", arch)
		return
	}
	// calculate hash
	fmt.Printf("====> Calculating Hash for %s\n", arch)
	hash, err := provenance.DigestFile(arch)
	if err != nil {
		return
	}

	// remove url name from url as helm's index library
	// adds it in during .Add
	// there should be a better way to handle this :(
	s := strings.Split(url, "/")
	s = s[:len(s)-1]

	// Add to index
	indexFile.Add(c.Metadata, path.Base(arch), strings.Join(s, "/"), hash)
}

// CreateReleases finds and uploads helm chart packages to github
func (r *Releaser) CreateReleases() error {
	packages, err := r.getListOfPackages(r.config.Path, r.config.Recursive)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		return errors.Errorf("No charts found at %s, try --recursive or a different path.\n", r.config.Path)
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

func (r *Releaser) getListOfPackages(dir string, recurse bool) ([]string, error) {
	archives, err := filepath.Glob(filepath.Join(dir, "*.tgz"))
	if err != nil {
		return nil, err
	}
	if recurse {
		moreArchives, err := filepath.Glob(filepath.Join(dir, "**/*.tgz"))
		if err != nil {
			return nil, err
		}
		archives = append(archives, moreArchives...)
	}
	return archives, nil
}
