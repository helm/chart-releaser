// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package releaser

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Songmu/retry"
	"github.com/blang/semver"

	"text/template"

	"helm.sh/helm/v3/pkg/chart"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart/loader"

	"github.com/helm/chart-releaser/pkg/config"

	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/helm/chart-releaser/pkg/github"
)

// GitHub contains the functions necessary for interacting with GitHub release
// objects
type GitHub interface {
	CreateRelease(ctx context.Context, input *github.Release) error
	GetRelease(ctx context.Context, tag string) (*github.Release, error)
	CreatePullRequest(owner string, repo string, message string, head string, base string) (string, error)
	GetLatestChartRelease(ctx context.Context, prefix string) (*github.Release, error)
	GenerateReleaseNotes(ctx context.Context, latestRelease *github.Release, chart *chart.Chart) (string, error)
}

type Git interface {
	AddWorktree(workingDir string, commitIsh string) (string, error)
	RemoveWorktree(workingDir string, path string) error
	Add(workingDir string, args ...string) error
	Commit(workingDir string, message string) error
	Push(workingDir string, args ...string) error
	Pull(workingDir string, args ...string) error
	GetPushURL(remote string, token string) (string, error)
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

const chartAssetFileExtension = ".tgz"

func init() {
	rand.New(rand.NewSource(time.Now().UnixNano())) // nolint: gosec
}

type Releaser struct {
	config *config.Options
	github GitHub
	git    Git
}

func NewReleaser(config *config.Options, github GitHub, git Git) *Releaser {
	return &Releaser{
		config: config,
		github: github,
		git:    git,
	}
}

// UpdateIndexFile updates the index.yaml file for a given Git repo
func (r *Releaser) UpdateIndexFile() (bool, error) {
	// if index-path doesn't end with index.yaml we can try and fix it
	if filepath.Base(r.config.IndexPath) != "index.yaml" {
		// if path is a directory then add index.yaml
		if stat, err := os.Stat(r.config.IndexPath); err == nil && stat.IsDir() {
			r.config.IndexPath = filepath.Join(r.config.IndexPath, "index.yaml")
			// otherwise error out
		} else {
			fmt.Printf("index-path (%s) should be a directory or a file called index.yaml\n", r.config.IndexPath)
			os.Exit(1)
		}
	}

	fmt.Printf("Loading index file from git repository %s\n", r.config.IndexPath)
	worktree, err := r.git.AddWorktree("", r.config.Remote+"/"+r.config.PagesBranch)
	if err != nil {
		return false, err
	}
	defer r.git.RemoveWorktree("", worktree) // nolint: errcheck

	// if pages-index-path doesn't end with index.yaml we can try and fix it
	if filepath.Base(r.config.PagesIndexPath) != "index.yaml" {
		// if path is a directory then add index.yaml
		if stat, err := os.Stat(filepath.Join(worktree, r.config.PagesIndexPath)); err == nil && stat.IsDir() {
			r.config.PagesIndexPath = filepath.Join(r.config.PagesIndexPath, "index.yaml")
			// otherwise error out
		} else {
			fmt.Printf("pages-index-path (%s) should be a directory or a file called index.yaml\n", r.config.PagesIndexPath)
			os.Exit(1) // nolint: gocritic
		}
	}
	indexYamlPath := filepath.Join(worktree, r.config.PagesIndexPath)

	var indexFile *repo.IndexFile
	_, err = os.Stat(indexYamlPath)
	if err == nil { // nolint: gocritic
		indexFile, err = repo.LoadIndexFile(indexYamlPath)
		if err != nil {
			return false, err
		}
	} else if errors.Is(err, os.ErrNotExist) {
		indexFile = repo.NewIndexFile()
	} else {
		return false, err
	}

	// We have to explicitly glob for *.tgz files only. If GPG signing is enabled,
	// this would also return *.tgz.prov files otherwise, which we don't want here.
	chartPackages, err := filepath.Glob(r.config.PackagePath + "/*.tgz")
	if err != nil {
		return false, err
	}

	var update bool
	for _, chartPackage := range chartPackages {
		ch, err := loader.LoadFile(chartPackage)
		if err != nil {
			return false, err
		}
		releaseName, err := r.computeReleaseName(ch)
		if err != nil {
			return false, err
		}

		var release *github.Release
		if err := retry.Retry(3, 3*time.Second, func() error {
			rel, err := r.github.GetRelease(context.TODO(), releaseName)
			if err != nil {
				return err
			}
			release = rel
			return nil
		}); err != nil {
			return false, err
		}

		for _, asset := range release.Assets {
			downloadURL, _ := url.Parse(asset.URL)
			name := filepath.Base(downloadURL.Path)
			// Ignore any other files added in the release by the users.
			if filepath.Ext(name) != chartAssetFileExtension {
				continue
			}
			baseName := strings.TrimSuffix(name, filepath.Ext(name))
			tagParts := r.splitPackageNameAndVersion(baseName)
			packageName, packageVersion := tagParts[0], tagParts[1]
			fmt.Printf("Found %s-%s.tgz\n", packageName, packageVersion)
			if _, err := indexFile.Get(packageName, packageVersion); err != nil {
				if err := r.addToIndexFile(indexFile, downloadURL.String()); err != nil {
					return false, err
				}
				update = true
				break
			}
		}
	}

	if !update {
		fmt.Printf("Index %s did not change\n", r.config.IndexPath)
		return false, nil
	}

	fmt.Printf("Updating index %s\n", r.config.IndexPath)
	indexFile.SortEntries()

	indexFile.Generated = time.Now()

	if err := indexFile.WriteFile(r.config.IndexPath, 0644); err != nil {
		return false, err
	}

	if !r.config.Push && !r.config.PR {
		return true, nil
	}

	if err := copyFile(r.config.IndexPath, indexYamlPath); err != nil {
		return false, err
	}

	if err := r.git.Pull(worktree, r.config.Remote, r.config.PagesBranch); err != nil {
		return false, err
	}

	if err := r.git.Add(worktree, indexYamlPath); err != nil {
		return false, err
	}

	if err := r.git.Commit(worktree, fmt.Sprintf("Update %s", r.config.PagesIndexPath)); err != nil {
		return false, err
	}

	if err := r.pushToPagesBranch(worktree); err != nil {
		return false, err
	}

	return true, nil
}

func (r *Releaser) computeReleaseName(chart *chart.Chart) (string, error) {
	tmpl, err := template.New("gotpl").Parse(r.config.ReleaseNameTemplate)
	if err != nil {
		return "", err
	}

	var buffer bytes.Buffer
	if err := tmpl.Execute(&buffer, chart.Metadata); err != nil {
		return "", err
	}

	releaseName := buffer.String()
	return releaseName, nil
}

func (r *Releaser) getReleaseNotes(chart *chart.Chart) (string, error) {
	if r.config.ReleaseNotesFile != "" {
		for _, f := range chart.Files {
			if f.Name == r.config.ReleaseNotesFile {
				return string(f.Data), nil
			}
		}
		fmt.Printf("The release note file %q, is not present in the chart package\n", r.config.ReleaseNotesFile)
	}
	if r.config.GenerateReleaseNotes {
		latestRelease, err := r.github.GetLatestChartRelease(context.TODO(), chart.Metadata.Name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get latest release for chart %s", chart.Metadata.Name)
		}
		nextVersion := semver.MustParse(chart.Metadata.Version)
		versions := []semver.Version{nextVersion, latestRelease.SemVer}
		semver.Sort(versions)
		highest := versions[len(versions)-1]
		// skip generating notes if there's already a higher version in GitHub
		if nextVersion.String() == highest.String() {
			notes, err := r.github.GenerateReleaseNotes(context.TODO(), latestRelease, chart)
			if err != nil {
				return "", errors.Wrapf(err, "failed to generate release notes for chart %s", chart.Metadata.Name)
			}
			return notes, nil
		}
	}
	if r.config.GenerateReleaseNotes {
		latestRelease, err := r.github.GetLatestChartRelease(context.TODO(), chart.Metadata.Name)
		if err != nil {
			return "", errors.Wrapf(err, "failed to get latest release for chart %s", chart.Metadata.Name)
		}
		nextVersion := semver.MustParse(chart.Metadata.Version)
		versions := []semver.Version{nextVersion, latestRelease.SemVer}
		semver.Sort(versions)
		highest := versions[len(versions)-1]
		// skip generating notes if there's already a higher version in GitHub
		if nextVersion.String() == highest.String() {
			notes, err := r.github.GenerateReleaseNotes(context.TODO(), latestRelease, chart)
			if err != nil {
				return "", errors.Wrapf(err, "failed to generate release notes for chart %s", chart.Metadata.Name)
			}
			return notes, nil
		}
	}
	return chart.Metadata.Description, nil
}

func (r *Releaser) splitPackageNameAndVersion(pkg string) []string {
	delimIndex := strings.LastIndex(pkg, "-")
	return []string{pkg[0:delimIndex], pkg[delimIndex+1:]}
}

func (r *Releaser) addToIndexFile(indexFile *repo.IndexFile, url string) error {
	arch := filepath.Join(r.config.PackagePath, filepath.Base(url))

	// extract chart metadata
	fmt.Printf("Extracting chart metadata from %s\n", arch)
	c, err := loader.LoadFile(arch)
	if err != nil {
		return errors.Wrapf(err, "%s is not a helm chart package", arch)
	}
	// calculate hash
	fmt.Printf("Calculating Hash for %s\n", arch)
	hash, err := provenance.DigestFile(arch)
	if err != nil {
		return err
	}

	// remove url name from url as helm's index library
	// adds it in during .Add
	// there should be a better way to handle this :(
	s := strings.Split(url, "/")
	s = s[:len(s)-1]

	if r.config.PackagesWithIndex {
		// the chart will be stored in the same repo as
		// the index file so let's make the path relative
		s = s[:0]
	}

	// Add to index
	return indexFile.MustAdd(c.Metadata, filepath.Base(arch), strings.Join(s, "/"), hash)
}

// CreateReleases finds and uploads Helm chart packages to GitHub
func (r *Releaser) CreateReleases() error {
	packages, err := r.getListOfPackages(r.config.PackagePath)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		return errors.Errorf("no charts found at %s", r.config.PackagePath)
	}

	for _, p := range packages {
		ch, err := loader.LoadFile(p)
		if err != nil {
			return err
		}
		releaseName, err := r.computeReleaseName(ch)
		if err != nil {
			return err
		}
		notes, err := r.getReleaseNotes(ch)
		if err != nil {
			return err
		}

		release := &github.Release{
			Name:        releaseName,
			Description: notes,
			Assets: []*github.Asset{
				{Path: p},
			},
			Commit:     r.config.Commit,
			MakeLatest: strconv.FormatBool(r.config.MakeReleaseLatest),
		}
		provFile := fmt.Sprintf("%s.prov", p)
		if _, err := os.Stat(provFile); err == nil {
			asset := &github.Asset{Path: provFile}
			release.Assets = append(release.Assets, asset)
		}
		if r.config.SkipExisting {
			existingRelease, _ := r.github.GetRelease(context.TODO(), releaseName)
			if existingRelease != nil {
				continue
			}
		}
		if err := r.github.CreateRelease(context.TODO(), release); err != nil {
			return errors.Wrapf(err, "error creating GitHub release %s", releaseName)
		}

		if r.config.PackagesWithIndex {
			worktree, err := r.git.AddWorktree("", r.config.Remote+"/"+r.config.PagesBranch)
			if err != nil {
				return err
			}
			defer r.git.RemoveWorktree("", worktree) //nolint: errcheck

			pkgTargetPath := filepath.Join(worktree, filepath.Base(p))
			if err := copyFile(p, pkgTargetPath); err != nil {
				return err
			}

			if err := r.git.Add(worktree, pkgTargetPath); err != nil {
				return err
			}

			if err := r.git.Commit(worktree, fmt.Sprintf("Publishing chart package for %s", releaseName)); err != nil {
				return err
			}

			if err := r.pushToPagesBranch(worktree); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *Releaser) getListOfPackages(dir string) ([]string, error) {
	return filepath.Glob(filepath.Join(dir, "*.tgz"))
}

func (r *Releaser) pushToPagesBranch(worktree string) error {
	pushURL, err := r.git.GetPushURL(r.config.Remote, r.config.Token)
	if err != nil {
		return err
	}

	if r.config.Push {
		fmt.Printf("Pushing to branch %q\n", r.config.PagesBranch)
		if err := r.git.Push(worktree, pushURL, "HEAD:refs/heads/"+r.config.PagesBranch); err != nil {
			return err
		}
	} else if r.config.PR {
		branch := fmt.Sprintf("chart-releaser-%s", randomString(16))

		fmt.Printf("Pushing to branch %q\n", branch)
		if err := r.git.Push(worktree, pushURL, "HEAD:refs/heads/"+branch); err != nil {
			return err
		}
		fmt.Printf("Creating pull request against branch %q\n", r.config.PagesBranch)
		prURL, err := r.github.CreatePullRequest(r.config.Owner, r.config.GitRepo, "Update index.yaml", branch, r.config.PagesBranch)
		if err != nil {
			return err
		}
		fmt.Println("Pull request created:", prURL)
	}

	return nil
}

func copyFile(srcFile string, dstFile string) error {
	source, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer destination.Close()

	_, err = io.Copy(destination, source)
	return err
}

func randomString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))] // nolint: gosec
	}
	return string(b)
}
