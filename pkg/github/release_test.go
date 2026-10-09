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
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
