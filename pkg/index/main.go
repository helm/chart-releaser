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

package index

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/helm/chart-releaser/pkg/config"

	"github.com/helm/chart-releaser/pkg/github"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

//Create index.yaml file for a give git repo
func Create(config *config.Options) error {
	var indexFile = &repo.IndexFile{}
	var toAdd []string

	// Create a GitHub client
	ghc := github.NewClient(config.Owner, config.Repo, config.Token)

	// if path doesn't end with index.yaml we can try and fix it
	if path.Base(config.Path) != "index.yaml" {
		// if path is a directory then add index.yaml
		if stat, err := os.Stat(config.Path); err == nil && stat.IsDir() {
			config.Path = path.Join(config.Path, "index.yaml")
			// otherwise error out
		} else {
			fmt.Printf("path (%s) should be a directory or a file called index.yaml\n", config.Path)
			os.Exit(1)
		}
	}

	// Load up Index file (or create new one)
	if _, err := os.Stat(config.Path); err == nil {
		fmt.Printf("====> Using existing index at %s\n", config.Path)
		indexFile, err = repo.LoadIndexFile(config.Path)
		if err != nil {
			return err
		}
	} else {
		fmt.Printf("====> Create new index at %s\n", config.Path)
		indexFile = repo.NewIndexFile()
	}

	// Get list of releases for given github repo
	releases, err := ghc.ListReleases(context.TODO())
	if err != nil {
		return err
	}
	// Check if release has a package
	fmt.Println("--> Checking for releases with helm chart packages")
	for _, r := range releases {
		//fmt.Printf("found release %s\n", *r.TagName)
		var packageName, packageVersion, packageURL string
		for _, f := range r.Assets {
			tagParts := splitPackageNameAndVersion(*r.TagName)
			if len(tagParts) == 2 && *f.Name == fmt.Sprintf("%s-%s.tgz", tagParts[0], tagParts[1]) {
				p := strings.TrimSuffix(*f.Name, filepath.Ext(*f.Name))
				ps := splitPackageNameAndVersion(p)
				packageName, packageVersion = ps[0], ps[1]
				packageURL = *f.BrowserDownloadURL

				fmt.Printf("====> Found %s-%s.tgz\n", packageName, packageVersion)
				// check if index file already has an entry for current package
				if _, err := indexFile.Get(packageName, packageVersion); err != nil {
					toAdd = append(toAdd, packageURL)
				}
				break
			}
		}
	}
	for _, u := range toAdd {
		addToIndexFile(indexFile, u)
	}
	fmt.Printf("--> Updating index %s\n", config.Path)
	indexFile.SortEntries()
	return indexFile.WriteFile(config.Path, 0644)

}

func splitPackageNameAndVersion(pkg string) []string {
	delimIndex := strings.LastIndex(pkg, "-")
	return []string{pkg[0:delimIndex], pkg[delimIndex+1:]}
}

func addToIndexFile(indexFile *repo.IndexFile, url string) {
	// fetch package to temp url so we can extract metadata and stuff
	dir, err := ioutil.TempDir("", "chart-releaser")
	if err != nil {
		log.Fatal(err)
	}
	defer os.RemoveAll(dir)
	arch := path.Join(dir, path.Base(url))
	fmt.Printf("====> Downloading url %s\n", arch)
	err = downloadFile(arch, url)
	if err != nil {
		panic(err)
	}
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

// from https://golangcode.com/download-a-file-from-a-url/
// downloadFile will download a url to a local file. It's efficient because it will
// write as it downloads and not load the whole file into memory.
func downloadFile(filepath string, url string) error {

	// Create the file
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Get the data
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Write the body to file
	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return err
	}

	return nil
}
