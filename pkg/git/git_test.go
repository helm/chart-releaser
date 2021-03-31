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

package git

import (
	"os"
	"os/exec"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGit_GetPushURL(t *testing.T) {
	curDir, _ := os.Getwd()
	repoPath := t.TempDir()
	repoDirErr := os.Chdir(repoPath)
	if repoDirErr != nil {
		t.Error(repoDirErr.Error())
	}

	t.Cleanup(func() {
		chdirErr := os.Chdir(curDir)
		if chdirErr != nil {
			t.Error(chdirErr.Error())
		}
	})

	_, initErr := exec.Command("git", "init").Output()
	if initErr != nil {
		t.Error(initErr.Error())
	}

	tests := []struct {
		name    string
		repo    string
		remote  string
		url     string
		token   string
		pushUrl string
	}{
		{
			name:    "Public GitHub",
			repo:    "publicrepo",
			remote:  "public",
			url:     "https://github.com/org/publicrepo",
			token:   "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",
			pushUrl: "https://x-access-token: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@github.com/org/publicrepo",
		},
		{
			name:    "GitHub Enterprise",
			repo:    "privaterepo",
			remote:  "enterprise",
			url:     "https://github.example.com/org/privaterepo",
			token:   "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",
			pushUrl: "https://x-access-token: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@github.example.com/org/privaterepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, addErr := exec.Command("git", "remote", "add", tt.remote, tt.url).Output()
			if addErr != nil {
				t.Error(addErr.Error())
			}

			g := Git{}
			pushUrl, pushErr := g.GetPushURL(tt.remote, tt.token)

			require.Empty(t, pushErr)
			require.EqualValues(t, pushUrl, tt.pushUrl)
		})
	}
}
