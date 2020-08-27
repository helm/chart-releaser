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
	"fmt"
	"os"
	"path/filepath"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/ulule/deepcopier"
	"helm.sh/helm/v3/pkg/action"
)

// Packager exposes the packager object
type Packager struct {
	config *config.Options
	paths  []string
}

// NewPackager returns a configured Packager
func NewPackager(config *config.Options, paths []string) *Packager {
	return &Packager{
		config: config,
		paths:  paths,
	}
}

// CreatePackages creates Helm chart packages
func (p *Packager) CreatePackages() error {
	var settings map[string]interface{}

	helmClient := action.NewPackage()

	helmClient.Destination = p.config.PackagePath

	for i := 0; i < len(p.paths); i++ {
		path, err := filepath.Abs(p.paths[i])
		if err != nil {
			return err
		}
		if _, err := os.Stat(p.paths[i]); err != nil {
			return err
		}
		deepcopier.Copy(p.config).To(settings)
		packageRun, err := helmClient.Run(path, settings)

		if err != nil {
			fmt.Printf("Failed to package chart in %s (%s)\n", path, err.Error())
			return err
		}

		fmt.Printf("Successfully packaged chart in %s and saved it to: %s\n", path, packageRun)
	}
	return nil
}
