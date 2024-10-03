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

package packager

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/incontact/chart-releaser/pkg/config"
)

func TestPackager_CreatePackages(t *testing.T) {
	packagePath, _ := os.MkdirTemp(".", "packages")
	invalidPackagePath := filepath.Join(packagePath, "bad")
	file, _ := os.Create(invalidPackagePath)
	t.Cleanup(func() {
		file.Close()
		os.RemoveAll(packagePath)
	})

	tests := []struct {
		name      string
		chartPath string
		options   *config.Options
		error     bool
	}{
		{
			name:      "valid-chart-path",
			chartPath: "testdata/test-chart",
			options:   &config.Options{PackagePath: packagePath},
			error:     false,
		},
		{
			name:      "invalid-package-path",
			chartPath: "testdata/test-chart",
			options:   &config.Options{PackagePath: invalidPackagePath},
			error:     true,
		},
		{
			name:      "invalid-chart-path",
			chartPath: "testdata/invalid-chart",
			options:   &config.Options{PackagePath: packagePath},
			error:     true,
		},
		{
			name:      "valid-chart-path-with-provenance",
			chartPath: "testdata/test-chart",
			options: &config.Options{
				PackagePath:    packagePath,
				Sign:           true,
				Key:            "Chart Releaser Test Key <no-reply@example.com>",
				KeyRing:        "testdata/testkeyring.gpg",
				PassphraseFile: "testdata/passphrase-file.txt",
			},
			error: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				os.Remove(filepath.Join(packagePath, "test-chart-0.1.0.tgz"))
				os.Remove(filepath.Join(packagePath, "test-chart-0.1.0.tgz.prov"))
			})

			p := &Packager{
				paths:  []string{tt.chartPath},
				config: tt.options,
			}
			err := p.CreatePackages()
			if tt.error {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.FileExists(t, filepath.Join(tt.options.PackagePath, "test-chart-0.1.0.tgz"))
				if tt.options.Sign {
					assert.FileExists(t, filepath.Join(tt.options.PackagePath, "test-chart-0.1.0.tgz.prov"))
				}
			}
		})
	}
}
