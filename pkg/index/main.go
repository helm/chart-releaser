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

	"github.com/paulczar/charthub/pkg/github"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/provenance"
	"k8s.io/helm/pkg/repo"
)

// Options to be passed in from cli
type Options struct {
	Owner string
	Repo  string
	Path  string
	Token string
}

//Create index.yaml file for a give git repo
func Create(config *Options) error {
	var ghc github.GitHub
	var err error
	var ctx = context.TODO()
	var indexFile = &repo.IndexFile{}
	var toAdd []string

	// Create a GitHub client
	ghc, err = github.NewGitHubClient(config.Owner, config.Repo, config.Token)
	if err != nil {
		fmt.Println("failed to log into github")
		os.Exit(1)
	}

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
	releases, err := ghc.ListReleases(ctx)
	if err != nil {
		return err
	}
	// Check if release has a package
	fmt.Println("--> Checking for releases with helm chart packages")
	for _, r := range releases {
		//fmt.Printf("found release %s\n", *r.TagName)
		var hasPackage = false
		var packageName, packageVersion, packageURL string
		for _, f := range r.Assets {
			m := "-" + *r.TagName + ".tgz"
			if strings.Contains(*f.Name, m) {
				hasPackage = true
				p := strings.TrimSuffix(*f.Name, filepath.Ext(*f.Name))
				ps := strings.Split(p, "-")
				packageName, packageVersion = ps[0], ps[1]
				packageURL = *f.BrowserDownloadURL
				continue
			}
		}
		if hasPackage {
			fmt.Printf("====> Found %s-%s.tgz\n", packageName, packageVersion)
			// check if index file already has an entry for current package
			if _, err := indexFile.Get(packageName, packageVersion); err != nil {
				toAdd = append(toAdd, packageURL)
			}
		}
	}
	for _, u := range toAdd {
		// fetch package to temp file so we can extract metadata and stuff
		dir, err := ioutil.TempDir("", "charthub")
		if err != nil {
			log.Fatal(err)
		}
		defer os.RemoveAll(dir)
		arch := path.Join(dir, path.Base(u))
		fmt.Printf("====> Downloading file %s\n", arch)
		err = downloadFile(arch, u)
		if err != nil {
			panic(err)
		}
		// extract chart metadata
		fmt.Printf("====> Extracting chart metadata from %s\n", arch)
		c, err := chartutil.Load(arch)
		if err != nil {
			// weird, must not be a chart package
			fmt.Printf("====> %s is not a helm chart package\n", arch)
			continue
		}
		// calculate hash
		fmt.Printf("====> Calculating Hash for %s\n", arch)
		hash, err := provenance.DigestFile(arch)
		if err != nil {
			return nil
		}

		// remove file name from url as helm's index library
		// adds it in during .Add
		// there should be a better way to handle this :(
		s := strings.Split(u, "/")
		s = s[:len(s)-1]
		u = strings.Join(s, "/")

		// Add to index
		indexFile.Add(c.Metadata, path.Base(arch), u, hash)
	}
	fmt.Printf("--> Updating index %s", config.Path)
	indexFile.SortEntries()
	return indexFile.WriteFile(config.Path, 0644)

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
