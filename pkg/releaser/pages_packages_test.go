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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
	chartgit "github.com/helm/chart-releaser/pkg/git"
	"github.com/helm/chart-releaser/pkg/github"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type pagesTestGit struct {
	chartgit.Git
	repo, remote string
}

func (g *pagesTestGit) AddWorktree(_ string, ref string) (string, error) {
	return g.Git.AddWorktree(g.repo, ref)
}

func (g *pagesTestGit) RemoveWorktree(_ string, path string) error {
	return g.Git.RemoveWorktree(g.repo, path)
}

func (g *pagesTestGit) GetPushURL(_, _ string) (string, error) {
	// The fixture uses a local bare repository instead of a GitHub URL.
	return g.remote, nil
}

type pagesTestGitHub struct {
	FakeGitHub
	exists bool
}

func (g *pagesTestGitHub) GetRelease(ctx context.Context, tag string) (*github.Release, error) {
	if !g.exists {
		return nil, nil
	}
	return g.FakeGitHub.GetRelease(ctx, tag)
}

func TestReleaser_CreateReleasesSkipExistingPages(t *testing.T) {
	// Disable signing only for commits in these temporary test repositories.
	t.Setenv("GIT_CONFIG_COUNT", "3")
	t.Setenv("GIT_CONFIG_KEY_0", "commit.gpgsign")
	t.Setenv("GIT_CONFIG_VALUE_0", "false")
	t.Setenv("GIT_CONFIG_KEY_1", "user.name")
	t.Setenv("GIT_CONFIG_VALUE_1", "Chart releaser test")
	t.Setenv("GIT_CONFIG_KEY_2", "user.email")
	t.Setenv("GIT_CONFIG_VALUE_2", "test@example.invalid")

	for _, tt := range []struct {
		name                                            string
		releaseExists, packageExists, packagesWithIndex bool
	}{
		{name: "restore missing Pages package", releaseExists: true, packagesWithIndex: true},
		{name: "preserve existing Pages package", releaseExists: true, packageExists: true, packagesWithIndex: true},
		{name: "skip release without Pages packages", releaseExists: true},
		{name: "publish new release and Pages package", packagesWithIndex: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			runGit := func(dir string, args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = dir
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
				return strings.TrimSpace(string(output))
			}
			remote := filepath.Join(root, "pages.git")
			runGit(root, "init", "--bare", remote)
			repo := filepath.Join(root, "repo")
			runGit(root, "clone", remote, repo)
			runGit(repo, "checkout", "-b", "gh-pages")
			require.NoError(t, os.WriteFile(filepath.Join(repo, "index.yaml"), []byte("apiVersion: v1\nentries: {}\n"), 0o600))
			const packageName = "test-chart-0.1.0.tgz"
			const existingPackage = "already published; must not be overwritten"
			if tt.packageExists {
				require.NoError(t, os.WriteFile(filepath.Join(repo, packageName), []byte(existingPackage), 0o600))
			}
			runGit(repo, "add", ".")
			runGit(repo, "commit", "-m", "Initial Pages files")
			runGit(repo, "push", "origin", "gh-pages")
			before := runGit(root, "--git-dir", remote, "rev-parse", "gh-pages")

			gh := &pagesTestGitHub{exists: tt.releaseExists}
			if !tt.releaseExists {
				gh.On("CreateRelease", mock.Anything, mock.Anything).Return(nil).Once()
			}
			r := &Releaser{
				config: &config.Options{
					PackagePath: "testdata/release-packages", ReleaseNameTemplate: "{{ .Name }}-{{ .Version }}",
					SkipExisting: true, PackagesWithIndex: tt.packagesWithIndex, Push: tt.packagesWithIndex,
					Remote: "origin", PagesBranch: "gh-pages", GenerateReleaseNotes: tt.releaseExists,
					PackageCommitMessage: "chore: publish {{ .ReleaseName }}",
				},
				github: gh,
				git:    &pagesTestGit{repo: repo, remote: remote},
			}
			require.NoError(t, r.CreateReleases())
			after := runGit(root, "--git-dir", remote, "rev-parse", "gh-pages")
			if tt.packagesWithIndex {
				got := runGit(root, "--git-dir", remote, "show", "gh-pages:"+packageName)
				if tt.packageExists {
					assert.Equal(t, existingPackage, got)
					assert.Equal(t, before, after, "already published package must not produce an empty commit")
				} else {
					want, err := os.ReadFile(filepath.Join("testdata/release-packages", packageName))
					require.NoError(t, err)
					assert.Equal(t, strings.TrimSpace(string(want)), got)
					assert.NotEqual(t, before, after)
					assert.Equal(t, "chore: publish test-chart-0.1.0", runGit(root, "--git-dir", remote, "log", "-1", "--format=%s", "gh-pages"))
				}
				gh.exists = true
				require.NoError(t, r.CreateReleases(), "rerunning must succeed without an empty commit")
				assert.Equal(t, after, runGit(root, "--git-dir", remote, "rev-parse", "gh-pages"))
			} else {
				assert.Equal(t, before, after)
			}
			gh.AssertNumberOfCalls(t, "CreateRelease", map[bool]int{true: 0, false: 1}[tt.releaseExists])
			gh.AssertNumberOfCalls(t, "GenerateReleaseNotes", 0)
			gh.AssertNumberOfCalls(t, "ListTagsWithPrefix", 0)
			gh.AssertExpectations(t)
		})
	}
}
