// Copyright The Helm Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package packager

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/helm/chart-releaser/pkg/config"
)

func TestPackager_CreatePackages(t *testing.T) {
	packagePath, _ := ioutil.TempDir(".", "packages")
	invalidPackagePath := filepath.Join(packagePath, "bad")
	file, _ := os.Create(invalidPackagePath)
	defer file.Close()
	defer os.RemoveAll(packagePath)
	tests := []struct {
		name        string
		chartPath   string
		packagePath string
		error       bool
	}{
		{
			"valid-chart-path",
			"testdata/test-chart",
			packagePath,
			false,
		},
		{
			"invalid-package-path",
			"testdata/test-chart",
			invalidPackagePath,
			true,
		},
		{
			"invalid-chart-path",
			"testdata/invalid-chart",
			packagePath,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Packager{
				paths:  strings.Split(tt.chartPath, ","),
				config: &config.Options{PackagePath: tt.packagePath},
			}
			err := p.CreatePackages()

			if tt.error {
				if err == nil {
					t.Error()
				}
			} else {
				if err != nil {
					t.Error()
				}
			}
		})
	}
}
