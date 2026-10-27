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
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"
)

func TestCreateReleasesReplacement(t *testing.T) {
	for _, tt := range []struct {
		name   string
		exists bool
		err    error
	}{
		{name: "replace", exists: true},
		{name: "create missing"},
		{name: "failure stops processing", exists: true, err: errors.New("immutable")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			gh := new(FakeGitHub)
			gh.On("ReplaceReleaseAssets", mock.Anything, mock.Anything).Return(tt.exists, tt.err).Once()
			if !tt.exists && tt.err == nil {
				gh.On("CreateRelease", mock.Anything, mock.Anything).Return(nil).Once()
			}
			r := NewReleaser(&config.Options{
				PackagePath: "testdata/release-packages", ReleaseNameTemplate: "{{ .Name }}-{{ .Version }}", ReplaceExisting: true,
			}, gh, new(FakeGit))
			err := r.CreateReleases()
			if tt.err != nil {
				require.ErrorIs(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
			gh.AssertExpectations(t)
		})
	}
	t.Run("conflicting flags", func(t *testing.T) {
		r := NewReleaser(&config.Options{SkipExisting: true, ReplaceExisting: true}, new(FakeGitHub), new(FakeGit))
		require.ErrorContains(t, r.CreateReleases(), "specify either")
	})
}

func TestReplaceIndexEntry(t *testing.T) {
	for _, tt := range []struct {
		name   string
		pages  bool
		useAPI bool
	}{
		{name: "release assets"},
		{name: "API asset URLs", useAPI: true},
		{name: "Pages packages", pages: true},
		{name: "Pages packages with API flag", pages: true, useAPI: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			packageDir := t.TempDir()
			metadata := &chart.Metadata{APIVersion: "v2", Name: "example", Version: "1.0.0+new", Description: "updated"}
			archive, err := chartutil.Save(&chart.Chart{Metadata: metadata}, packageDir)
			require.NoError(t, err)
			digest, err := provenance.DigestFile(archive)
			require.NoError(t, err)
			index := repo.NewIndexFile()
			for _, version := range []string{"1.0.0+old", "1.0.0+new", "1.0.0+new", "0.9.0"} {
				require.NoError(t, index.MustAdd(&chart.Metadata{APIVersion: "v2", Name: "example", Version: version, Description: "old"}, "old.tgz", "https://old.invalid", "old-digest"))
			}
			r := NewReleaser(&config.Options{PackagePath: packageDir, ReplaceExisting: true, PackagesWithIndex: tt.pages, UseAPIAssetURL: tt.useAPI}, nil, nil)
			assetURL := "https://github.com/owner/repo/releases/download/example-1.0.0%2Bnew/example-1.0.0%2Bnew.tgz"
			apiURL := "https://api.github.com/repos/owner/repo/releases/assets/123"
			updated, err := r.maybeAddToIndexFile(index, assetURL, apiURL)
			require.NoError(t, err)
			require.True(t, updated)
			require.Len(t, index.Entries["example"], 3)
			for _, entry := range index.Entries["example"] {
				if entry.Version == metadata.Version {
					assert.Equal(t, digest, entry.Digest)
					assert.Equal(t, "updated", entry.Description)
					switch {
					case tt.pages:
						assert.Equal(t, []string{"example-1.0.0+new.tgz"}, entry.URLs)
					case tt.useAPI:
						assert.Equal(t, []string{apiURL}, entry.URLs)
					default:
						assert.Equal(t, []string{strings.ReplaceAll(assetURL, "%2B", "+")}, entry.URLs)
					}
				} else {
					assert.Equal(t, "old-digest", entry.Digest)
				}
			}
			updated, err = r.maybeAddToIndexFile(index, assetURL, apiURL)
			require.NoError(t, err)
			assert.False(t, updated, "identical replacement must not change index timestamps or produce duplicate entries")
			if tt.useAPI && !tt.pages {
				newAPIURL := "https://api.github.com/repos/owner/repo/releases/assets/456"
				updated, err = r.maybeAddToIndexFile(index, assetURL, newAPIURL)
				require.NoError(t, err)
				assert.True(t, updated, "a replacement asset ID must refresh the API URL even when bytes are unchanged")
				for _, entry := range index.Entries["example"] {
					if entry.Version == metadata.Version {
						assert.Equal(t, []string{newAPIURL}, entry.URLs)
					}
				}
			}
		})
	}
}

func TestCommitReplacementPackage(t *testing.T) {
	for _, signed := range []bool{false, true} {
		t.Run(map[bool]string{false: "remove stale provenance", true: "replace provenance"}[signed], func(t *testing.T) {
			sourceDir, pagesDir := t.TempDir(), t.TempDir()
			archive := filepath.Join(sourceDir, "chart.tgz")
			require.NoError(t, os.WriteFile(archive, []byte("new archive"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(pagesDir, "chart.tgz"), []byte("old archive"), 0o600))
			require.NoError(t, os.WriteFile(filepath.Join(pagesDir, "chart.tgz.prov"), []byte("old provenance"), 0o600))
			if signed {
				require.NoError(t, os.WriteFile(archive+".prov", []byte("new provenance"), 0o600))
			}
			git := new(FakeGit)
			git.On("Add", pagesDir, mock.Anything).Return(nil).Once()
			git.On("Commit", pagesDir, "Publishing chart package for chart").Return(nil).Once()
			r := NewReleaser(&config.Options{ReplaceExisting: true}, nil, git)
			require.NoError(t, r.commitPackage(pagesDir, archive, "chart"))
			actual, err := os.ReadFile(filepath.Join(pagesDir, "chart.tgz"))
			require.NoError(t, err)
			assert.Equal(t, "new archive", string(actual))
			actual, err = os.ReadFile(filepath.Join(pagesDir, "chart.tgz.prov"))
			if signed {
				require.NoError(t, err)
				assert.Equal(t, "new provenance", string(actual))
			} else {
				require.ErrorIs(t, err, os.ErrNotExist)
			}
			require.NoError(t, r.commitPackage(pagesDir, archive, "chart"), "identical replacement must not create an empty commit")
			git.AssertExpectations(t)
		})
	}
}
