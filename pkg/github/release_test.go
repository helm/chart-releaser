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
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"unicode/utf8"

	"github.com/google/go-github/v56/github"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClient_CreateRelease_BodyLimit(t *testing.T) {
	const limit = 125000
	for _, tt := range []struct {
		name            string
		description     string
		notes           string
		generate        bool
		generationError bool
		want            string
	}{
		{name: "empty"},
		{name: "short", description: "Chart description", want: "Chart description"},
		{name: "at limit", description: strings.Repeat("a", limit), want: strings.Repeat("a", limit)},
		{name: "over limit", description: strings.Repeat("a", limit) + "b", want: strings.Repeat("a", limit)},
		{name: "unicode at limit", description: strings.Repeat("界", limit), want: strings.Repeat("界", limit)},
		{name: "unicode over limit", description: strings.Repeat("界", limit-1) + "🚀extra", want: strings.Repeat("界", limit-1) + "🚀"},
		{name: "generated notes", description: "Chart", notes: "Changes", generate: true, want: "Chart\n\nChanges"},
		{name: "generated notes without description", notes: "Changes", generate: true, want: "Changes"},
		{name: "combined body over limit", description: "Chart", notes: strings.Repeat("a", limit), generate: true, want: "Chart\n\n" + strings.Repeat("a", limit-7)},
		{name: "generation fails", generate: true, generationError: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var generated, created, published bool
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method + " " + r.URL.Path {
				case "POST /repos/owner/repo/releases/generate-notes":
					generated = true
					var opts map[string]any
					require.NoError(t, json.NewDecoder(r.Body).Decode(&opts))
					assert.Equal(t, "chart-1.0.0", opts["tag_name"])
					assert.Equal(t, "commit-sha", opts["target_commitish"])
					assert.NotContains(t, opts, "previous_tag_name", "omit the previous tag to use GitHub's default comparison")
					if tt.generationError {
						w.WriteHeader(http.StatusInternalServerError)
						_, _ = w.Write([]byte(`{"message":"generation failed"}`))
						return
					}
					require.NoError(t, json.NewEncoder(w).Encode(map[string]string{"body": tt.notes}))
				case "POST /repos/owner/repo/releases":
					created = true
					var release github.RepositoryRelease
					require.NoError(t, json.NewDecoder(r.Body).Decode(&release))
					assert.Equal(t, tt.want, release.GetBody())
					assert.LessOrEqual(t, utf8.RuneCountInString(release.GetBody()), limit)
					assert.True(t, utf8.ValidString(release.GetBody()))
					assert.False(t, release.GetGenerateReleaseNotes())
					assert.True(t, release.GetDraft())
					w.WriteHeader(http.StatusCreated)
					_, _ = w.Write([]byte(`{"id":42}`))
				case "PATCH /repos/owner/repo/releases/42":
					published = true
					_, _ = w.Write([]byte(`{"id":42}`))
				default:
					t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()

			release := &Release{Name: "chart-1.0.0", Description: tt.description, GenerateReleaseNotes: tt.generate, Commit: "commit-sha"}
			err := NewClient("owner", "repo", "", srv.URL, srv.URL).CreateRelease(context.Background(), release)
			if tt.generationError {
				require.ErrorContains(t, err, "failed to generate release notes")
			} else {
				require.NoError(t, err)
			}
			assert.Equal(t, tt.generate, generated)
			assert.Equal(t, !tt.generationError, created)
			assert.Equal(t, !tt.generationError, published)
			assert.Equal(t, tt.description, release.Description, "preserve the caller's input")
			assert.Equal(t, tt.generate, release.GenerateReleaseNotes)
		})
	}
}

func TestClient_CreateRelease_DraftThenPublish(t *testing.T) {
	var (
		mu       sync.Mutex
		requests []string
		created  map[string]any
		patched  map[string]any
	)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		requests = append(requests, r.Method+" "+r.URL.Path)

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/releases":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&created))
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": 42}`))
		case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/releases/42/assets":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{"id": 1}`))
		case r.Method == http.MethodPatch && r.URL.Path == "/repos/owner/repo/releases/42":
			require.NoError(t, json.NewDecoder(r.Body).Decode(&patched))
			_, _ = w.Write([]byte(`{"id": 42}`))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	asset := filepath.Join(t.TempDir(), "chart-0.1.0.tgz")
	require.NoError(t, os.WriteFile(asset, []byte("chart"), 0o600))

	client := NewClient("owner", "repo", "token", srv.URL, srv.URL)
	err := client.CreateRelease(context.Background(), &Release{
		Name:       "chart-0.1.0",
		Assets:     []*Asset{{Path: asset}},
		MakeLatest: "false",
		PreRelease: true,
	})
	require.NoError(t, err)

	assert.Equal(t, []string{
		"POST /repos/owner/repo/releases",
		"POST /repos/owner/repo/releases/42/assets",
		"PATCH /repos/owner/repo/releases/42",
	}, requests)

	// The release has to be a draft while the assets are uploaded, so that
	// repositories with immutable releases accept them.
	assert.Equal(t, true, created["draft"])
	assert.Equal(t, true, created["prerelease"])
	assert.Equal(t, "false", created["make_latest"])

	assert.Equal(t, false, patched["draft"])
	assert.Equal(t, "false", patched["make_latest"])
}

func TestClient_GetReleaseAssetURLs(t *testing.T) {
	for _, tt := range []struct{ name, body, browserURL, apiURL string }{
		{name: "both URLs", body: `{"assets":[{"name":"chart.tgz","browser_download_url":"https://github.com/owner/repo/releases/download/v1/chart.tgz","url":"https://api.github.com/repos/owner/repo/releases/assets/42"}]}`, browserURL: "https://github.com/owner/repo/releases/download/v1/chart.tgz", apiURL: "https://api.github.com/repos/owner/repo/releases/assets/42"},
		{name: "omitted optional fields", body: `{"assets":[{}]}`},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/repos/owner/repo/releases/tags/v1", r.URL.Path)
				_, _ = w.Write([]byte(tt.body))
			}))
			defer srv.Close()
			release, err := NewClient("owner", "repo", "", srv.URL, srv.URL).GetRelease(context.Background(), "v1")
			require.NoError(t, err)
			require.Len(t, release.Assets, 1)
			assert.Equal(t, tt.browserURL, release.Assets[0].URL)
			assert.Equal(t, tt.apiURL, release.Assets[0].APIURL)
		})
	}
}
