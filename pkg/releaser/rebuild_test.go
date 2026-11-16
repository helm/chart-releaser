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
	"context"
	"os"
	"path/filepath"
	"slices"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/provenance"
	"helm.sh/helm/v3/pkg/repo"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/helm/chart-releaser/pkg/github"
)

// rebuildTestGitHub serves releases whose assets are read from local files.
type rebuildTestGitHub struct {
	FakeGitHub
	releases []*github.Release
}

func (g *rebuildTestGitHub) ListReleases(_ context.Context) ([]*github.Release, error) {
	return slices.Clone(g.releases), nil
}

func (g *rebuildTestGitHub) DownloadReleaseAsset(_ context.Context, asset *github.Asset, path string) error {
	return copyFile(asset.Path, path)
}

func saveChart(t *testing.T, dir, name, version string) string {
	t.Helper()
	versionDir := filepath.Join(dir, version)
	require.NoError(t, os.MkdirAll(versionDir, 0o755))
	archive, err := chartutil.Save(&chart.Chart{Metadata: &chart.Metadata{APIVersion: "v2", Name: name, Version: version}}, versionDir)
	require.NoError(t, err)
	return archive
}

func chartRelease(tag string, published time.Time, archives ...string) *github.Release {
	release := &github.Release{Name: tag, PublishedAt: published}
	for _, archive := range archives {
		release.Assets = append(release.Assets, &github.Asset{
			Path: archive,
			URL:  "https://github.com/owner/repo/releases/download/" + tag + "/" + filepath.Base(archive),
		})
	}
	return release
}

func TestReleaser_UpdateIndexFileRebuild(t *testing.T) {
	charts := t.TempDir()
	v010 := saveChart(t, charts, "test-chart", "0.1.0")
	v020 := saveChart(t, charts, "test-chart", "0.2.0")
	other := saveChart(t, charts, "other-chart", "1.0.0")
	notChart := filepath.Join(charts, "source.tgz")
	require.NoError(t, os.WriteFile(notChart, []byte("not a chart"), 0o600))
	notes := filepath.Join(charts, "notes.txt")
	require.NoError(t, os.WriteFile(notes, []byte("notes"), 0o600))

	first := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	second := first.Add(48 * time.Hour)
	third := second.Add(48 * time.Hour)
	// Listed newest first, like the GitHub API does.
	fakeGitHub := &rebuildTestGitHub{releases: []*github.Release{
		chartRelease("other-chart-1.0.0", third, other, notes),
		chartRelease("test-chart-0.2.0", second, v020, notChart),
		chartRelease("test-chart-0.1.0", first, v010),
	}}

	indexPath := filepath.Join(t.TempDir(), "index.yaml")
	fakeGit := new(FakeGit)
	fakeGit.On("RemoveWorktree", mock.Anything, mock.Anything).Return(nil)
	r := &Releaser{
		config: &config.Options{IndexPath: indexPath, PackagePath: t.TempDir(), Rebuild: true},
		github: fakeGitHub,
		git:    fakeGit,
	}

	updated, err := r.UpdateIndexFile()
	require.NoError(t, err)
	assert.True(t, updated)

	indexFile, err := repo.LoadIndexFile(indexPath)
	require.NoError(t, err)
	assert.Len(t, indexFile.Entries, 2)
	for _, tt := range []struct {
		name, version, archive, tag string
		created                     time.Time
	}{
		{"test-chart", "0.1.0", v010, "test-chart-0.1.0", first},
		{"test-chart", "0.2.0", v020, "test-chart-0.2.0", second},
		{"other-chart", "1.0.0", other, "other-chart-1.0.0", third},
	} {
		entry, err := indexFile.Get(tt.name, tt.version)
		require.NoError(t, err)
		assert.True(t, tt.created.Equal(entry.Created), "%s-%s is created when its release was published, got %s", tt.name, tt.version, entry.Created)
		digest, err := provenance.DigestFile(tt.archive)
		require.NoError(t, err)
		assert.Equal(t, digest, entry.Digest)
		assert.Equal(t, []string{"https://github.com/owner/repo/releases/download/" + tt.tag + "/" + filepath.Base(tt.archive)}, entry.URLs)
	}

	// Rebuilding the same releases again gives the same index, so there is nothing to update.
	contents, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	fakeGit.indexFile = indexPath
	updated, err = r.UpdateIndexFile()
	require.NoError(t, err)
	assert.False(t, updated)
	newContents, err := os.ReadFile(indexPath)
	require.NoError(t, err)
	assert.Equal(t, string(contents), string(newContents))

	// Versions without a release are dropped from the existing index.
	fakeGitHub.releases = fakeGitHub.releases[1:]
	updated, err = r.UpdateIndexFile()
	require.NoError(t, err)
	assert.True(t, updated)
	indexFile, err = repo.LoadIndexFile(indexPath)
	require.NoError(t, err)
	assert.False(t, indexFile.Has("other-chart", "1.0.0"))
	assert.True(t, indexFile.Has("test-chart", "0.1.0"))
	assert.True(t, indexFile.Has("test-chart", "0.2.0"))
}

func TestReleaser_UpdateIndexFileRebuildKeepsFirstRelease(t *testing.T) {
	charts := t.TempDir()
	original := saveChart(t, charts, "test-chart", "0.1.0")
	republished := filepath.Join(t.TempDir(), filepath.Base(original))
	_, err := chartutil.Save(&chart.Chart{Metadata: &chart.Metadata{APIVersion: "v2", Name: "test-chart", Version: "0.1.0", Description: "republished"}}, filepath.Dir(republished))
	require.NoError(t, err)

	first := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	fakeGitHub := &rebuildTestGitHub{releases: []*github.Release{
		chartRelease("test-chart-0.1.0-again", first.Add(time.Hour), republished),
		chartRelease("test-chart-0.1.0", first, original),
	}}
	indexPath := filepath.Join(t.TempDir(), "index.yaml")
	fakeGit := new(FakeGit)
	fakeGit.On("RemoveWorktree", mock.Anything, mock.Anything).Return(nil)
	r := &Releaser{
		config: &config.Options{IndexPath: indexPath, Rebuild: true},
		github: fakeGitHub,
		git:    fakeGit,
	}

	updated, err := r.UpdateIndexFile()
	require.NoError(t, err)
	assert.True(t, updated)

	// Like publishing the releases one by one, the version from the oldest release is kept.
	indexFile, err := repo.LoadIndexFile(indexPath)
	require.NoError(t, err)
	require.Len(t, indexFile.Entries["test-chart"], 1)
	digest, err := provenance.DigestFile(original)
	require.NoError(t, err)
	assert.Equal(t, digest, indexFile.Entries["test-chart"][0].Digest)
	assert.True(t, first.Equal(indexFile.Entries["test-chart"][0].Created))
}

func TestReleaser_UpdateIndexFileCreatedFromRelease(t *testing.T) {
	published := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	indexPath := filepath.Join(t.TempDir(), "index.yaml")
	fakeGit := new(FakeGit)
	fakeGit.On("RemoveWorktree", mock.Anything, mock.Anything).Return(nil)
	r := &Releaser{
		config: &config.Options{IndexPath: indexPath, PackagePath: "testdata/release-packages-prerelease"},
		github: &FakeGitHubWithURLEncoding{release: &github.Release{
			Name:        "test-chart-0.2.0-beta2",
			PublishedAt: published,
			Assets: []*github.Asset{{
				Path: "testdata/release-packages-prerelease/test-chart-0.2.0-beta2.tgz",
				URL:  "https://myrepo/charts/test-chart-0.2.0-beta2.tgz",
			}},
		}},
		git: fakeGit,
	}

	updated, err := r.UpdateIndexFile()
	require.NoError(t, err)
	assert.True(t, updated)

	indexFile, err := repo.LoadIndexFile(indexPath)
	require.NoError(t, err)
	entry, err := indexFile.Get("test-chart", "0.2.0-beta2")
	require.NoError(t, err)
	assert.True(t, published.Equal(entry.Created), "a new version is created when its release was published, got %s", entry.Created)
}
