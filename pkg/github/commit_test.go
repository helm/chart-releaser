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

package github

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateSignedCommit(t *testing.T) {
	for _, tt := range []struct {
		name, basePath, response, wantError string
		status                              int
	}{
		{name: "GitHub", response: `{"data":{"createCommitOnBranch":{"commit":{"oid":"new-head"}}}}`},
		{name: "Enterprise", basePath: "/api/v3/", response: `{"data":{"createCommitOnBranch":{"commit":{"oid":"new-head"}}}}`},
		{name: "stale head", response: `{"errors":[{"message":"Expected branch to point to old-head"}]}`, wantError: "Expected branch"},
		{name: "permission denied", status: http.StatusForbidden, response: `{"message":"Resource not accessible"}`, wantError: "403"},
		{name: "no commit", response: `{"data":{"createCommitOnBranch":null}}`, wantError: "returned no commit"},
		{name: "malformed response", response: `{`, wantError: "unexpected EOF"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				wantPath := "/graphql"
				if tt.basePath != "" {
					wantPath = "/api/graphql"
				}
				assert.Equal(t, wantPath, r.URL.Path)
				assert.Equal(t, http.MethodPost, r.Method)
				assert.Equal(t, "Bearer token", r.Header.Get("Authorization"))
				var payload struct {
					Query     string `json:"query"`
					Variables struct {
						Input struct {
							Branch       map[string]string `json:"branch"`
							ExpectedHead string            `json:"expectedHeadOid"`
							Message      map[string]string `json:"message"`
							FileChanges  struct {
								Additions []struct{ Path, Contents string } `json:"additions"`
							} `json:"fileChanges"`
						} `json:"input"`
					} `json:"variables"`
				}
				require.NoError(t, json.NewDecoder(r.Body).Decode(&payload))
				assert.Contains(t, payload.Query, "createCommitOnBranch(input: $input)")
				input := payload.Variables.Input
				assert.Equal(t, map[string]string{"repositoryNameWithOwner": "owner/repo", "refName": "gh-pages"}, input.Branch)
				assert.Equal(t, "old-head", input.ExpectedHead)
				assert.Equal(t, map[string]string{"headline": "Update charts/index.yaml", "body": "Details"}, input.Message)
				require.Len(t, input.FileChanges.Additions, 1)
				assert.Equal(t, "charts/index.yaml", input.FileChanges.Additions[0].Path)
				contents, err := base64.StdEncoding.DecodeString(input.FileChanges.Additions[0].Contents)
				require.NoError(t, err)
				assert.Equal(t, "apiVersion: v1\n# 界\n", string(contents))
				if tt.status != 0 {
					w.WriteHeader(tt.status)
				}
				_, _ = w.Write([]byte(tt.response))
			}))
			defer srv.Close()
			client := NewClient("owner", "repo", "token", srv.URL+tt.basePath, srv.URL)
			err := client.CreateSignedCommit(context.Background(), "gh-pages", "old-head", "charts/index.yaml", "Update charts/index.yaml\n\nDetails", []byte("apiVersion: v1\n# 界\n"))
			if tt.wantError != "" {
				require.ErrorContains(t, err, tt.wantError)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
