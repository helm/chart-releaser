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
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/pkg/errors"

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

type HttpClient interface {
	Get(url string) (*http.Response, error)
}

type DefaultHttpClient struct {
}

func (c *DefaultHttpClient) Get(url string) (resp *http.Response, err error) {
	return http.Get(url)
}

type Releaser struct {
	config *config.Options
	github GitHub
	httpClient HttpClient
}

func NewReleaser(config *config.Options, github GitHub) *Releaser {
	return &Releaser{
		config: config,
		github: github,
		httpClient: &DefaultHttpClient{},
	}
}

// UpdateIndexFile index.yaml file for a give git repo
func (r *Releaser) UpdateIndexFile() (bool, error) {
	// if path doesn't end with index.yaml we can try and fix it
	if filepath.Base(r.config.IndexPath) != "index.yaml" {
		// if path is a directory then add index.yaml
		if stat, err := os.Stat(r.config.IndexPath); err == nil && stat.IsDir() {
			r.config.IndexPath = filepath.Join(r.config.IndexPath, "index.yaml")
			// otherwise error out
		} else {
			fmt.Printf("path (%s) should be a directory or a file called index.yaml\n", r.config.IndexPath)
			os.Exit(1)
		}
	}

	var indexFile *repo.IndexFile

	resp, err := r.httpClient.Get(fmt.Sprintf("%s/index.yaml", r.config.ChartsRepo))
	if err != nil {
		return false, err
	}

	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		out, err := os.Create(r.config.IndexPath)
		if err != nil {
			return false, err
		}
		defer out.Close()

		_, err = io.Copy(out, resp.Body)
		if err != nil {
			return false, err
		}

		fmt.Printf("====> Using existing index at %s\n", r.config.IndexPath)
		indexFile, err = repo.LoadIndexFile(r.config.IndexPath)
		if err != nil {
			return false, err
		}
	} else {
		fmt.Printf("====> UpdateIndexFile new index at %s\n", r.config.IndexPath)
		indexFile = repo.NewIndexFile()
	}

	chartPackages, err := ioutil.ReadDir(r.config.PackagePath)
	if err != nil {
		return false, err
	}

	var update bool
	for _, chartPackage := range chartPackages {
		tag := strings.TrimSuffix(chartPackage.Name(), filepath.Ext(chartPackage.Name()))

		release, err := r.github.GetRelease(context.TODO(), tag)
		if err != nil {
			return false, err
		}

		for _, asset := range release.Assets {
			downloadUrl, _ := url.Parse(asset.URL)
			name := filepath.Base(downloadUrl.Path)
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			tagParts := r.splitPackageNameAndVersion(baseName)
			packageName, packageVersion := tagParts[0], tagParts[1]
			fmt.Printf("====> Found %s-%s.tgz\n", packageName, packageVersion)
			if _, err := indexFile.Get(packageName, packageVersion); err != nil {
				if err := r.addToIndexFile(indexFile, downloadUrl.String()); err != nil {
					return false, err
				}
				update = true
				break
			}
		}
	}

	if update {
		fmt.Printf("--> Updating index %s\n", r.config.IndexPath)
		indexFile.SortEntries()
		return true, indexFile.WriteFile(r.config.IndexPath, 0644)
	} else {
		fmt.Printf("--> Index %s did not change\n", r.config.IndexPath)
	}

	return false, nil
}

func (r *Releaser) splitPackageNameAndVersion(pkg string) []string {
	delimIndex := strings.LastIndex(pkg, "-")
	return []string{pkg[0:delimIndex], pkg[delimIndex+1:]}
}

func (r *Releaser) addToIndexFile(indexFile *repo.IndexFile, url string) error {
	arch := filepath.Join(r.config.PackagePath, filepath.Base(url))

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
	indexFile.Add(c.Metadata, filepath.Base(arch), strings.Join(s, "/"), hash)
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
		baseName := filepath.Base(strings.TrimSuffix(p, filepath.Ext(p)))
		chart, err := chartutil.Load(p)
		if err != nil {
			return err
		}
		release := &github.Release{
			Name:        baseName,
			Description: chart.Metadata.Description,
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
