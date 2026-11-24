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

package pusher

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
	"oras.land/oras-go/v2/registry/remote/errcode"

	"github.com/helm/chart-releaser/pkg/config"
)

func TestPushPackages_RegistryURLRequired(t *testing.T) {
	p := &Pusher{
		config: &config.Options{RegistryURL: "", PackagePath: t.TempDir()},
		out:    &bytes.Buffer{},
	}
	err := p.PushPackages()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "'--registry-url' is required")
}

func TestPushPackages_ValidatesScheme(t *testing.T) {
	tests := []struct {
		name        string
		registryURL string
	}{
		{"http", "http://example.com/charts"},
		{"https", "https://example.com/charts"},
		{"no scheme", "example.com/charts"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Pusher{
				config: &config.Options{RegistryURL: tt.registryURL, PackagePath: t.TempDir()},
				out:    &bytes.Buffer{},
			}
			err := p.PushPackages()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "registry-url must start with")
		})
	}
}

func TestPushPackages_NoPackagesFound(t *testing.T) {
	p := &Pusher{
		config: &config.Options{
			RegistryURL: "oci://example.com/charts",
			PackagePath: t.TempDir(),
		},
		out: &bytes.Buffer{},
	}
	err := p.PushPackages()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no chart packages found")
}

func TestPushPackages_PushFailsAgainstUnreachableRegistry(t *testing.T) {
	packageDir := t.TempDir()
	tgz := packageTestChart(t, packageDir)

	p := &Pusher{
		config: &config.Options{
			RegistryURL: "oci://127.0.0.1:1/charts",
			PackagePath: packageDir,
			PlainHTTP:   true,
		},
		out: &bytes.Buffer{},
	}
	err := p.PushPackages()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "pushing "+tgz)
}

func TestChartExists(t *testing.T) {
	chartPackage := packageTestChart(t, t.TempDir())
	for _, status := range []int{
		http.StatusOK,
		http.StatusNotFound,
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, http.MethodHead, r.Method)
				assert.Equal(t, "/v2/charts/test-chart/manifests/0.1.0", r.URL.Path)
				if status == http.StatusOK {
					w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
					w.Header().Set("Content-Length", "2")
					w.Header().Set("Docker-Content-Digest", fmt.Sprintf("sha256:%x", sha256.Sum256([]byte("{}"))))
				}
				w.WriteHeader(status)
			}))
			defer server.Close()
			client, err := registry.NewClient(
				registry.ClientOptPlainHTTP(),
				registry.ClientOptHTTPClient(server.Client()),
				registry.ClientOptCredentialsFile(filepath.Join(t.TempDir(), "config.json")),
			)
			require.NoError(t, err)
			remote := ociScheme + strings.TrimPrefix(server.URL, "http://") + "/charts"

			exists, err := chartExists(client, remote, chartPackage)
			assert.Equal(t, status == http.StatusOK, exists)
			if status == http.StatusOK || status == http.StatusNotFound {
				require.NoError(t, err)
			} else {
				var responseError *errcode.ErrorResponse
				require.ErrorAs(t, err, &responseError)
				assert.Equal(t, status, responseError.StatusCode)
			}
		})
	}
}

func TestChartExistsTransportError(t *testing.T) {
	chartPackage := packageTestChart(t, t.TempDir())
	wantErr := errors.New("registry transport failed")
	client, err := registry.NewClient(
		registry.ClientOptPlainHTTP(),
		registry.ClientOptHTTPClient(&http.Client{Transport: registryErrorTransport{err: wantErr}}),
		registry.ClientOptCredentialsFile(filepath.Join(t.TempDir(), "config.json")),
	)
	require.NoError(t, err)

	exists, err := chartExists(client, "oci://registry.example/charts", chartPackage)
	assert.False(t, exists)
	assert.ErrorIs(t, err, wantErr)
}

type registryErrorTransport struct {
	err error
}

func (r registryErrorTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, r.err
}

func TestPushPackages_SkipExistingRegistryError(t *testing.T) {
	t.Setenv("HELM_CONFIG_HOME", t.TempDir())
	t.Setenv("DOCKER_CONFIG", t.TempDir())
	packageDir := t.TempDir()
	chartPackage := packageTestChart(t, packageDir)
	for _, status := range []int{
		http.StatusUnauthorized,
		http.StatusForbidden,
		http.StatusInternalServerError,
	} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			var writes atomic.Int32
			server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch {
				case r.Method != http.MethodGet && r.Method != http.MethodHead:
					writes.Add(1)
					w.WriteHeader(http.StatusBadRequest)
				case r.URL.Path == "/v2/charts/test-chart/manifests/0.1.0":
					w.WriteHeader(status)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer server.Close()
			p := &Pusher{
				config: &config.Options{
					RegistryURL:           ociScheme + strings.TrimPrefix(server.URL, "https://") + "/charts",
					PackagePath:           packageDir,
					SkipExisting:          true,
					InsecureSkipTLSVerify: true,
				},
				out: &bytes.Buffer{},
			}

			err := p.PushPackages()
			require.Error(t, err)
			assert.Contains(t, err.Error(), "checking "+chartPackage+" in registry")
			assert.Zero(t, writes.Load(), "must not push after an unsuccessful existence check")
			var responseError *errcode.ErrorResponse
			require.ErrorAs(t, err, &responseError)
			assert.Equal(t, status, responseError.StatusCode)
		})
	}
}

// packageTestChart packages testdata/test-chart into dest and returns the
// resulting .tgz path. It uses helm's own packager so the artifact is a
// real chart archive a registry could accept.
func packageTestChart(t *testing.T, dest string) string {
	t.Helper()
	packageClient := action.NewPackage()
	packageClient.Destination = dest
	out, err := packageClient.Run("testdata/test-chart", nil)
	require.NoError(t, err)
	return filepath.Clean(out)
}
