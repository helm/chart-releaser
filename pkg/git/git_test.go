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
		pushURL string
	}{
		{
			name:    "Public GitHub",
			repo:    "publicrepo",
			remote:  "public",
			url:     "https://github.com/org/publicrepo",
			token:   "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",
			pushURL: "https://x-access-token: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@github.com/org/publicrepo",
		},
		{
			name:    "GitHub Enterprise",
			repo:    "privaterepo",
			remote:  "enterprise",
			url:     "https://github.example.com/org/privaterepo",
			token:   "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",
			pushURL: "https://x-access-token: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@github.example.com/org/privaterepo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, addErr := exec.Command("git", "remote", "add", tt.remote, tt.url).Output()
			if addErr != nil {
				t.Error(addErr.Error())
			}

			g := Git{}
			pushURL, pushErr := g.GetPushURL(tt.remote, tt.token)

			require.Empty(t, pushErr)
			require.EqualValues(t, pushURL, tt.pushURL)
		})
	}
}

func TestBuildPushURL(t *testing.T) {
	tests := []struct {
		name      string
		remoteURL string
		want      string
		wantErr   bool
	}{
		{name: "https", remoteURL: "https://github.com/org/repo", want: "https://x-access-token:tok@github.com/org/repo"},
		{name: "https with .git", remoteURL: "https://github.com/org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "https enterprise with port", remoteURL: "https://github.example.com:8443/org/repo", want: "https://x-access-token:tok@github.example.com:8443/org/repo"},
		{name: "https with credentials in the url", remoteURL: "https://user:pass@github.com/org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "http", remoteURL: "http://github.example.com/org/repo", want: "http://x-access-token:tok@github.example.com/org/repo"},
		{name: "scp-like ssh", remoteURL: "git@github.com:org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "scp-like ssh enterprise", remoteURL: "git@github.example.com:org/repo.git", want: "https://x-access-token:tok@github.example.com/org/repo.git"},
		{name: "scp-like ssh without user", remoteURL: "github.com:org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "ssh url", remoteURL: "ssh://git@github.com/org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "ssh url with port", remoteURL: "ssh://git@github.example.com:2222/org/repo.git", want: "https://x-access-token:tok@github.example.com/org/repo.git"},
		{name: "git url", remoteURL: "git://github.com/org/repo.git", want: "https://x-access-token:tok@github.com/org/repo.git"},
		{name: "local path", remoteURL: "/srv/git/repo.git", wantErr: true},
		{name: "empty", remoteURL: "", wantErr: true},
		{name: "scp-like without path", remoteURL: "git@github.com:", wantErr: true},
		{name: "url without host", remoteURL: "https:///org/repo", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := buildPushURL(tt.remoteURL, "tok")
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestGit_GetPushURL_SSHRemote(t *testing.T) {
	t.Chdir(t.TempDir())
	require.NoError(t, exec.Command("git", "init").Run())
	require.NoError(t, exec.Command("git", "remote", "add", "origin", "git@github.com:org/repo.git").Run())

	g := Git{}
	pushURL, err := g.GetPushURL("origin", "tok")
	require.NoError(t, err)
	require.Equal(t, "https://x-access-token:tok@github.com/org/repo.git", pushURL)
}
