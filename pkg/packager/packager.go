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
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/mitchellh/go-homedir"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/registry"
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
	helmClient := action.NewPackage()
	helmClient.DependencyUpdate = true
	helmClient.Destination = p.config.PackagePath
	if p.config.Sign {
		// expand the ~ to the full home dir
		if strings.HasPrefix(p.config.KeyRing, "~") {
			dir, err := homedir.Dir()
			if err != nil {
				panic(err)
			}

			p.config.KeyRing = strings.ReplaceAll(p.config.KeyRing, "~", dir)
		}

		helmClient.Sign = true
		helmClient.Key = p.config.Key
		helmClient.Keyring = p.config.KeyRing
		helmClient.PassphraseFile = p.config.PassphraseFile
	}

	settings := cli.New()
	getters := getter.All(settings)
	registryClient, err := registry.NewClient()
	if err != nil {
		return err
	}

	for i := 0; i < len(p.paths); i++ {
		path, err := filepath.Abs(p.paths[i])
		if err != nil {
			return err
		}
		if _, err := os.Stat(p.paths[i]); err != nil {
			return err
		}

		downloadManager := &downloader.Manager{
			Out:              io.Discard,
			ChartPath:        path,
			Keyring:          helmClient.Keyring,
			Getters:          getters,
			Debug:            settings.Debug,
			RepositoryConfig: settings.RepositoryConfig,
			RepositoryCache:  settings.RepositoryCache,
			RegistryClient:   registryClient,
		}
		if err := downloadManager.Build(); err != nil {
			return err
		}
		packageRun, err := helmClient.Run(path, nil)
		if err != nil {
			fmt.Printf("Failed to package chart in %s (%s)\n", path, err.Error())
			return err
		}

		fmt.Printf("Successfully packaged chart in %s and saved it to: %s\n", path, packageRun)
	}
	return nil
}
