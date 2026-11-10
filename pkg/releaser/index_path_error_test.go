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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type invalidIndexPathGit struct {
	Git
	path, cleanupMarker string
}

func (g *invalidIndexPathGit) AddWorktree(_, _ string) (string, error) {
	return g.path, os.MkdirAll(g.path, 0o700)
}

func (g *invalidIndexPathGit) RemoveWorktree(_, path string) error {
	if err := os.WriteFile(g.cleanupMarker, []byte("cleaned"), 0o600); err != nil {
		return err
	}
	return os.RemoveAll(path)
}

func TestInvalidIndexPathsReturnErrors(t *testing.T) {
	if name := os.Getenv("CR_TEST_INVALID_INDEX_CASE"); name != "" {
		root := os.Getenv("CR_TEST_INVALID_INDEX_ROOT")
		cfg := &config.Options{IndexPath: filepath.Join(root, "index.yaml"), PagesIndexPath: "index.yaml"}
		if name == "index-path" {
			cfg.IndexPath = filepath.Join(root, "wrong.yaml")
		} else {
			cfg.PagesIndexPath = "missing-directory"
		}
		r := NewReleaser(cfg, nil, &invalidIndexPathGit{
			path: filepath.Join(root, "worktree"), cleanupMarker: filepath.Join(root, "cleanup"),
		})
		updated, err := r.UpdateIndexFile()
		require.ErrorContains(t, err, name+" (")
		assert.False(t, updated)
		fmt.Println("validation returned to caller")
		return
	}

	for _, name := range []string{"index-path", "pages-index-path"} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			cmd := exec.Command(os.Args[0], "-test.run=^TestInvalidIndexPathsReturnErrors$")
			cmd.Env = append(os.Environ(), "CR_TEST_INVALID_INDEX_CASE="+name, "CR_TEST_INVALID_INDEX_ROOT="+root)
			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "validation must return an error instead of exiting the process: %s", output)
			assert.Contains(t, string(output), "validation returned to caller")
			if name == "pages-index-path" {
				assert.FileExists(t, filepath.Join(root, "cleanup"))
			} else {
				assert.NoFileExists(t, filepath.Join(root, "cleanup"))
			}
			assert.NoDirExists(t, filepath.Join(root, "worktree"))
		})
	}
}
