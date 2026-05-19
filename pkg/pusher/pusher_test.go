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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/action"

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
