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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/repo"
)

func TestSignedIndexCommit(t *testing.T) {
	for _, tt := range []struct {
		name, path             string
		unchanged, fail, merge bool
	}{
		{name: "new index", path: "index.yaml"},
		{name: "merge existing chart", path: "index.yaml", merge: true},
		{name: "nested index", path: "charts/index.yaml"},
		{name: "directory path", path: "charts/"},
		{name: "unchanged", path: "index.yaml", unchanged: true},
		{name: "stale branch", path: "index.yaml", fail: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			runGit := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", args...)
				cmd.Dir = root
				output, err := cmd.CombinedOutput()
				require.NoError(t, err, "%s", output)
				return strings.TrimSpace(string(output))
			}
			runGit("init", "-b", "gh-pages")
			require.NoError(t, os.Mkdir(filepath.Join(root, "charts"), 0o755))
			require.NoError(t, os.WriteFile(filepath.Join(root, "charts", "keep.txt"), []byte("keep"), 0o600))
			if tt.unchanged {
				require.NoError(t, copyFile("testdata/repo/index.yaml", filepath.Join(root, "index.yaml")))
			}
			if tt.merge {
				contents, err := os.ReadFile("testdata/repo/index.yaml")
				require.NoError(t, err)
				require.NoError(t, os.WriteFile(filepath.Join(root, "index.yaml"), []byte(strings.ReplaceAll(string(contents), "test-chart", "existing-chart")), 0o600))
			}
			runGit("add", ".")
			runGit("-c", "commit.gpgsign=false", "-c", "user.name=Test", "-c", "user.email=test@example.invalid", "commit", "-m", "Initial index")
			head := runGit("rev-parse", "HEAD")
			// Point the local tracking ref at the snapshot used to generate the index.
			runGit("update-ref", "refs/remotes/origin/gh-pages", head)
			gh := new(FakeGitHub)
			indexPath := filepath.Join(t.TempDir(), "index.yaml")
			wantPath := tt.path
			if tt.path == "charts/" {
				wantPath = "charts/index.yaml"
			}
			var apiError error
			if tt.fail {
				apiError = errors.New("expectedHeadOid does not match")
			}
			if !tt.unchanged {
				gh.On("CreateSignedCommit", mock.Anything, "gh-pages", head, wantPath, "Publish "+wantPath+"\n\nDetails", mock.Anything).
					Run(func(args mock.Arguments) {
						contents, err := os.ReadFile(indexPath)
						require.NoError(t, err)
						assert.Equal(t, contents, args.Get(5))
						index, err := repo.LoadIndexFile(indexPath)
						require.NoError(t, err)
						assert.True(t, index.Has("test-chart", "0.1.0"))
						if tt.merge {
							assert.True(t, index.Has("existing-chart", "0.1.0"))
						}
					}).Return(apiError).Once()
			}
			r := &Releaser{
				config: &config.Options{
					Push: true, SignCommits: true, Token: "token", Remote: "origin", PagesBranch: "gh-pages",
					IndexPath: indexPath, PagesIndexPath: tt.path, PackagePath: "testdata/release-packages",
					IndexCommitMessage: "Publish {{ .PagesIndexPath }}\n\nDetails",
				},
				github: gh, git: &pagesTestGit{repo: root},
			}
			updated, err := r.UpdateIndexFile()
			if tt.fail {
				require.ErrorContains(t, err, "refresh it and rerun")
				require.ErrorIs(t, err, apiError)
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, !tt.unchanged && !tt.fail, updated)
			gh.AssertExpectations(t)
			if tt.unchanged {
				gh.AssertNotCalled(t, "CreateSignedCommit", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
			}
			// The API path must neither create local commits nor push.
			assert.Equal(t, head, runGit("rev-parse", "HEAD"))
			assert.Equal(t, head, runGit("rev-parse", "origin/gh-pages"))
			assert.Equal(t, "", runGit("status", "--porcelain"))
			assert.NotContains(t, runGit("worktree", "list"), "detached HEAD")
		})
	}
}

func TestSignedIndexCommitValidation(t *testing.T) {
	for _, opts := range []*config.Options{
		{SignCommits: true},
		{SignCommits: true, Push: true, PR: true, Token: "token"},
		{SignCommits: true, Push: true},
	} {
		r := &Releaser{config: opts}
		_, err := r.UpdateIndexFile()
		require.Error(t, err)
	}
}
