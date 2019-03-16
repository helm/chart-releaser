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

package upload

import (
	"context"
	"fmt"
	"github.com/helm/chart-releaser/pkg/config"
	"github.com/helm/chart-releaser/pkg/github"
	"github.com/pkg/errors"
	"os"
	"path/filepath"
	"strings"
)

// CreateReleases finds and uploads helm chart packages to github
func CreateReleases(config *config.Options) error {
	packages, err := getListOfPackages(config.Path, config.Recursive)
	if err != nil {
		return err
	}

	if len(packages) == 0 {
		return errors.Errorf("No charts found at %s, try --recursive or a different path.\n", config.Path)
	}

	ghc := github.NewClient(config.Owner, config.Repo, config.Token)

	for _, p := range packages {
		baseName := strings.TrimSuffix(p, filepath.Ext(p))

		release := &github.Release{
			Name: baseName,
			Assets: []*github.Asset{
				{Path: p},
			},
		}
		provFile := fmt.Sprintf("%s.prov", p)
		if _, err := os.Stat(provFile); err == nil {
			asset := &github.Asset{Path: provFile}
			release.Assets = append(release.Assets, asset)
		}

		if err := ghc.CreateRelease(context.TODO(), release); err != nil {
			return errors.Wrap(err, "error creating GitHub release")
		}
	}

	return nil
}

func getListOfPackages(dir string, recurse bool) ([]string, error) {
	archives, err := filepath.Glob(filepath.Join(dir, "*.tgz"))
	if err != nil {
		return nil, err
	}
	if recurse {
		moreArchives, err := filepath.Glob(filepath.Join(dir, "**/*.tgz"))
		if err != nil {
			return nil, err
		}
		archives = append(archives, moreArchives...)
	}
	return archives, nil
}
