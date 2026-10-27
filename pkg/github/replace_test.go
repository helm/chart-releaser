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
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReplaceReleaseAssets(t *testing.T) {
	for _, tt := range []struct {
		name       string
		provenance bool
		failure    string
		wantError  bool
	}{
		{name: "signed", provenance: true},
		{name: "remove stale provenance"},
		{name: "failed second upload", provenance: true, failure: "upload", wantError: true},
		{name: "failed backup rename", provenance: true, failure: "backup", wantError: true},
		{name: "failed promotion", provenance: true, failure: "promote", wantError: true},
		{name: "retained backup", provenance: true, failure: "cleanup"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			type asset struct {
				ID    int64  `json:"id"`
				Name  string `json:"name"`
				Bytes string `json:"-"`
			}
			assets := map[int64]*asset{
				1: {ID: 1, Name: "chart.tgz", Bytes: "old archive"},
				2: {ID: 2, Name: "chart.tgz.prov", Bytes: "old provenance"},
				3: {ID: 3, Name: "notes.txt", Bytes: "unrelated"},
			}
			var mu sync.Mutex
			var requests []string
			nextID := int64(10)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				mu.Lock()
				defer mu.Unlock()
				requests = append(requests, r.Method+" "+r.URL.Path)
				w.Header().Set("Content-Type", "application/json")
				switch {
				case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "/tags/"):
					_, _ = io.WriteString(w, `{"id":42,"tag_name":"chart","body":"keep notes","prerelease":true}`)
				case r.Method == http.MethodGet && r.URL.Path == "/repos/owner/repo/releases/42/assets":
					// The provenance is deliberately on the second page.
					if r.URL.Query().Get("page") == "2" {
						_ = json.NewEncoder(w).Encode([]*asset{assets[2], assets[3]})
					} else {
						w.Header().Set("Link", `<?page=2>; rel="next"`)
						_ = json.NewEncoder(w).Encode([]*asset{assets[1]})
					}
				case r.Method == http.MethodPost && r.URL.Path == "/repos/owner/repo/releases/42/assets":
					if tt.failure == "upload" && nextID == 11 {
						w.WriteHeader(http.StatusUnprocessableEntity)
						return
					}
					body, err := io.ReadAll(r.Body)
					require.NoError(t, err)
					a := &asset{ID: nextID, Name: r.URL.Query().Get("name"), Bytes: string(body)}
					assets[nextID] = a
					nextID++
					w.WriteHeader(http.StatusCreated)
					_ = json.NewEncoder(w).Encode(a)
				case strings.HasPrefix(r.URL.Path, "/repos/owner/repo/releases/assets/"):
					id, err := strconv.ParseInt(filepath.Base(r.URL.Path), 10, 64)
					require.NoError(t, err)
					a := assets[id]
					if a == nil {
						w.WriteHeader(http.StatusNotFound)
						return
					}
					if r.Method == http.MethodDelete {
						if tt.failure == "cleanup" && id == 1 {
							w.WriteHeader(http.StatusForbidden)
							return
						}
						delete(assets, id)
						w.WriteHeader(http.StatusNoContent)
						return
					}
					require.Equal(t, http.MethodPatch, r.Method)
					var update asset
					require.NoError(t, json.NewDecoder(r.Body).Decode(&update))
					if (tt.failure == "backup" && id == 2 && strings.HasSuffix(update.Name, "-backup")) ||
						(tt.failure == "promote" && id == 11 && update.Name == "chart.tgz.prov") {
						w.WriteHeader(http.StatusForbidden)
						return
					}
					for otherID, other := range assets {
						if id != otherID && other.Name == update.Name {
							w.WriteHeader(http.StatusUnprocessableEntity)
							return
						}
					}
					a.Name = update.Name
					_ = json.NewEncoder(w).Encode(a)
				default:
					t.Errorf("unexpected request %s %s", r.Method, r.URL)
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer srv.Close()
			archive := filepath.Join(t.TempDir(), "chart.tgz")
			require.NoError(t, os.WriteFile(archive, []byte("new archive"), 0o600))
			input := &Release{Name: "chart", Assets: []*Asset{{Path: archive}}}
			if tt.provenance {
				require.NoError(t, os.WriteFile(archive+".prov", []byte("new provenance"), 0o600))
				input.Assets = append(input.Assets, &Asset{Path: archive + ".prov"})
			}
			client := NewClient("owner", "repo", "", srv.URL, srv.URL)
			exists, err := client.ReplaceReleaseAssets(context.Background(), input)
			require.True(t, exists)
			mu.Lock()
			defer mu.Unlock()
			if tt.wantError {
				require.Error(t, err)
				require.Len(t, assets, 3)
				assert.Equal(t, "old archive", assets[1].Bytes)
				assert.Equal(t, "chart.tgz", assets[1].Name)
				assert.Equal(t, "old provenance", assets[2].Bytes)
				assert.Equal(t, "chart.tgz.prov", assets[2].Name)
			} else {
				require.NoError(t, err)
				require.NotNil(t, assets[10])
				assert.Equal(t, "chart.tgz", assets[10].Name)
				assert.Equal(t, "new archive", assets[10].Bytes)
				assert.Nil(t, assets[2])
				if tt.provenance {
					require.NotNil(t, assets[11])
					assert.Equal(t, "chart.tgz.prov", assets[11].Name)
					assert.Equal(t, "new provenance", assets[11].Bytes)
				}
				if tt.failure == "cleanup" {
					require.NotNil(t, assets[1])
					assert.True(t, strings.HasSuffix(assets[1].Name, "-backup"))
				} else {
					assert.Nil(t, assets[1])
				}
			}
			assert.Equal(t, "notes.txt", assets[3].Name)
			assert.Equal(t, "unrelated", assets[3].Bytes)
			for _, request := range requests {
				assert.NotEqual(t, "PATCH /repos/owner/repo/releases/42", request, "release metadata must be preserved")
			}
		})
	}
}

func TestReplaceReleaseAssetsPreflight(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		exists bool
		err    string
	}{
		{name: "missing", status: 404},
		{name: "auth error", status: 403, err: "403"},
		{name: "immutable", status: 200, body: `{"id":42,"immutable":true}`, exists: true, err: "immutable"},
		{name: "invalid file", status: 200, body: `{"id":42}`, exists: true, err: "no such file"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodGet, r.Method)
				assert.Equal(t, "/repos/owner/repo/releases/tags/chart", r.URL.Path)
				w.WriteHeader(tt.status)
				_, _ = fmt.Fprint(w, tt.body)
			}))
			defer srv.Close()
			client := NewClient("owner", "repo", "", srv.URL, srv.URL)
			exists, err := client.ReplaceReleaseAssets(context.Background(), &Release{Name: "chart", Assets: []*Asset{{Path: filepath.Join(t.TempDir(), "missing.tgz")}}})
			assert.Equal(t, tt.exists, exists)
			if tt.err != "" {
				require.ErrorContains(t, err, tt.err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
