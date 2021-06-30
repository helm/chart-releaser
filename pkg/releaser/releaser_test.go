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
	"bufio"
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/helm/chart-releaser/pkg/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/helm/chart-releaser/pkg/config"
)

type FakeGitHub struct {
	mock.Mock
	release *github.Release
}

type FakeGit struct {
	mock.Mock
}

type MockClient struct {
	statusCode int
	file       string
}

func (m *MockClient) Get(url string) (*http.Response, error) {
	if m.statusCode == http.StatusOK {
		file, _ := os.Open(m.file)
		reader := bufio.NewReader(file)
		return &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(reader)}, nil
	} else {
		return &http.Response{StatusCode: http.StatusNotFound, Body: ioutil.NopCloser(nil)}, nil
	}
}

func (m *MockClient) GetWithToken(url string, token string) (*http.Response, error) {
	if m.statusCode == http.StatusOK {
		file, _ := os.Open(m.file)
		reader := bufio.NewReader(file)
		return &http.Response{StatusCode: http.StatusOK, Body: ioutil.NopCloser(reader)}, nil
	} else {
		return &http.Response{StatusCode: http.StatusNotFound, Body: ioutil.NopCloser(nil)}, nil
	}
}

func (f *FakeGitHub) CreateRelease(ctx context.Context, input *github.Release) error {
	f.Called(ctx, input)
	f.release = input
	return nil
}

func (f *FakeGitHub) GetRelease(ctx context.Context, tag string) (*github.Release, error) {
	release := &github.Release{
		Name:        "testdata/release-packages/test-chart-0.1.0",
		Description: "A Helm chart for Kubernetes",
		Assets: []*github.Asset{
			{
				Path: "testdata/release-packages/test-chart-0.1.0.tgz",
				URL:  "https://myrepo/charts/test-chart-0.1.0.tgz",
			},
		},
	}
	return release, nil
}

func (f *FakeGitHub) CreatePullRequest(owner string, repo string, message string, head string, base string) (string, error) {
	f.Called(owner, repo, message, head, base)
	return "https://github.com/owner/repo/pull/42", nil
}

func (f *FakeGit) AddWorktree(workingDir string, committish string) (string, error) {
	f.Called(workingDir, committish)
	dir, err := ioutil.TempDir("testdata", "chart-releaser-")
	if err != nil {
		return "", err
	}
	return dir, nil
}

func (f *FakeGit) RemoveWorktree(workingDir string, path string) error {
	f.Called(workingDir, path)
	return os.RemoveAll(workingDir)
}

func (f *FakeGit) Add(workingDir string, args ...string) error {
	f.Called(workingDir, args)
	if len(args) == 0 {
		return fmt.Errorf("no args specified")
	}
	return nil
}

func (f *FakeGit) Commit(workingDir string, message string) error {
	f.Called(workingDir, message)
	return nil
}

func (f *FakeGit) Push(workingDir string, args ...string) error {
	f.Called(workingDir, args)
	return nil
}

func (f *FakeGit) GetPushURL(remote string, token string) (string, error) {
	f.Called(remote, token)
	pushURLWithToken := fmt.Sprintf("https://x-access-token:%s@github.com/owner/repo", token)
	return pushURLWithToken, nil
}

func TestReleaser_UpdateIndexFile(t *testing.T) {
	indexDir, _ := ioutil.TempDir(".", "index")
	defer os.RemoveAll(indexDir)

	fakeGitHub := new(FakeGitHub)

	tests := []struct {
		name     string
		exists   bool
		releaser *Releaser
	}{
		{
			"index-file-exists",
			true,
			&Releaser{
				config: &config.Options{
					IndexPath:   "testdata/index/index.yaml",
					PackagePath: "testdata/release-packages",
				},
				github:     fakeGitHub,
				httpClient: &MockClient{http.StatusOK, "testdata/repo/index.yaml"},
			},
		},
		{
			"index-file-does-not-exist",
			false,
			&Releaser{
				config: &config.Options{
					IndexPath:   filepath.Join(indexDir, "index.yaml"),
					PackagePath: "testdata/release-packages",
				},
				github:     fakeGitHub,
				httpClient: &MockClient{http.StatusNotFound, ""},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var sha256 string
			if tt.exists {
				sha256, _ = provenance.DigestFile(tt.releaser.config.IndexPath)
			}
			update, err := tt.releaser.UpdateIndexFile()
			assert.NoError(t, err)
			assert.Equal(t, update, !tt.exists)
			if tt.exists {
				newSha256, _ := provenance.DigestFile(tt.releaser.config.IndexPath)
				assert.Equal(t, sha256, newSha256)
			} else {
				_, err := os.Stat(tt.releaser.config.IndexPath)
				assert.NoError(t, err)
			}
		})
	}
}

func TestReleaser_UpdateIndexFileGenerated(t *testing.T) {
	indexDir, _ := ioutil.TempDir(".", "index")
	defer os.RemoveAll(indexDir)

	fakeGitHub := new(FakeGitHub)

	tests := []struct {
		name     string
		releaser *Releaser
	}{
		{
			"index-file-exists",
			&Releaser{
				config: &config.Options{
					IndexPath:   filepath.Join(indexDir, "index.yaml"),
					PackagePath: "testdata/release-packages",
				},
				github:     fakeGitHub,
				httpClient: &MockClient{http.StatusOK, "testdata/empty-repo/index.yaml"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexFile, _ := repo.LoadIndexFile("testdata/empty-repo/index.yaml")
			generated := indexFile.Generated
			update, err := tt.releaser.UpdateIndexFile()
			assert.NoError(t, err)
			assert.True(t, update)
			newIndexFile, _ := repo.LoadIndexFile(tt.releaser.config.IndexPath)
			newGenerated := newIndexFile.Generated
			assert.True(t, newGenerated.After(generated))
		})
	}
}

func TestReleaser_splitPackageNameAndVersion(t *testing.T) {
	tests := []struct {
		name     string
		pkg      string
		expected []string
	}{
		{
			"no-hyphen",
			"foo",
			nil,
		},
		{
			"one-hyphen",
			"foo-1.2.3",
			[]string{"foo", "1.2.3"},
		},
		{
			"two-hyphens",
			"foo-bar-1.2.3",
			[]string{"foo-bar", "1.2.3"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &Releaser{}
			if tt.expected == nil {
				assert.Panics(t, func() {
					r.splitPackageNameAndVersion(tt.pkg)
				}, "slice bounds out of range")
			} else {
				actual := r.splitPackageNameAndVersion(tt.pkg)
				assert.Equal(t, tt.expected, actual)
			}
		})
	}
}

func TestReleaser_addToIndexFile(t *testing.T) {
	tests := []struct {
		name              string
		chart             string
		version           string
		releaser          *Releaser
		packagesWithIndex bool
		error             bool
	}{
		{
			"invalid-package",
			"does-not-exist",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					PackagesWithIndex: false,
				},
			},
			false,
			true,
		},
		{
			"valid-package",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					PackagesWithIndex: false,
				},
			},
			false,
			false,
		},
		{
			"valid-package-with-index",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					PackagesWithIndex: true,
				},
			},
			true,
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexFile := repo.NewIndexFile()
			url := fmt.Sprintf("https://myrepo/charts/%s-%s.tgz", tt.chart, tt.version)
			err := tt.releaser.addToIndexFile(indexFile, url)
			if tt.error {
				assert.Error(t, err)
				assert.False(t, indexFile.Has(tt.chart, tt.version))
			} else {
				assert.True(t, indexFile.Has(tt.chart, tt.version))

				indexEntry, _ := indexFile.Get(tt.chart, tt.version)
				if tt.packagesWithIndex {
					assert.Equal(t, filepath.Base(url), indexEntry.URLs[0])
				} else {
					assert.Equal(t, url, indexEntry.URLs[0])
				}
			}
		})
	}
}

func TestReleaser_CreateReleases(t *testing.T) {
	tests := []struct {
		name     string
		chart    string
		version  string
		Releaser *Releaser
		error    bool
	}{
		{
			"invalid-package-path",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/does-not-exist",
					Commit:            "",
					PackagesWithIndex: false,
				},
			},
			true,
		},
		{
			"valid-package-path",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					Commit:            "",
					PackagesWithIndex: false,
				},
			},
			false,
		},
		{
			"valid-package-path-with-commit",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					Commit:            "5e239bd19fbefb9eb0181ecf0c7ef73b8fe2753c",
					PackagesWithIndex: false,
				},
			},
			false,
		},
		{
			"valid-package-path-with-commit-package-with-index",
			"test-chart",
			"0.1.0",
			&Releaser{
				config: &config.Options{
					PackagePath:       "testdata/release-packages",
					Commit:            "5e239bd19fbefb9eb0181ecf0c7ef73b8fe2753c",
					PackagesWithIndex: true,
					Push:              true,
				},
			},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fakeGitHub := new(FakeGitHub)
			fakeGitHub.On("CreateRelease", mock.Anything, mock.Anything).Return(nil)
			tt.Releaser.github = fakeGitHub
			fakeGit := new(FakeGit)
			fakeGit.On("AddWorktree", mock.Anything, mock.Anything).Return("/tmp/chart-releaser-012345678", nil)
			fakeGit.On("RemoveWorktree", mock.Anything, mock.Anything).Return(nil)
			fakeGit.On("Add", mock.Anything, mock.Anything).Return(nil)
			fakeGit.On("Commit", mock.Anything, mock.Anything).Return(nil)
			fakeGit.On("Push", mock.Anything, mock.Anything).Return(nil)
			pushURL := fmt.Sprintf("https://x-access-token:%s@github.com/owner/repo", tt.Releaser.config.Token)
			fakeGit.On("GetPushURL", mock.Anything, mock.Anything).Return(pushURL, nil)
			tt.Releaser.git = fakeGit
			tt.Releaser.config.ReleaseNameTemplate = "{{ .Name }}-{{ .Version }}"
			err := tt.Releaser.CreateReleases()
			if tt.error {
				assert.Error(t, err)
				assert.Nil(t, fakeGitHub.release)
				fakeGitHub.AssertNumberOfCalls(t, "CreateRelease", 0)
			} else {
				assert.NoError(t, err)
				releaseName := fmt.Sprintf("%s-%s", tt.chart, tt.version)
				assetPath := fmt.Sprintf("%s/%s-%s.tgz", tt.Releaser.config.PackagePath, tt.chart, tt.version)
				releaseDescription := "A Helm chart for Kubernetes"
				assert.Equal(t, releaseName, fakeGitHub.release.Name)
				assert.Equal(t, releaseDescription, fakeGitHub.release.Description)
				assert.Len(t, fakeGitHub.release.Assets, 1)
				assert.Equal(t, assetPath, fakeGitHub.release.Assets[0].Path)
				assert.Equal(t, tt.Releaser.config.Commit, fakeGitHub.release.Commit)
				fakeGitHub.AssertNumberOfCalls(t, "CreateRelease", 1)
			}
		})
	}
}
