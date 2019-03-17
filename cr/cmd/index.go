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

package cmd

import (
	"github.com/helm/chart-releaser/pkg/config"
	"github.com/helm/chart-releaser/pkg/github"
	"github.com/helm/chart-releaser/pkg/releaser"
	"github.com/spf13/cobra"
)

// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "UpdateIndexFile Helm repo index.yaml for the given GitHub repo",
	Long: `
UpdateIndexFile a Helm chart repository index.yaml file based on a the
given GitHub repository's releases.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		config, err := config.LoadConfiguration(cfgFile, cmd, getRequiredIndexArgs())
		if err != nil {
			return err
		}
		ghc := github.NewClient(config.Owner, config.Repo, config.Token)
		releaser := releaser.NewReleaser(config, ghc)
		return releaser.UpdateIndexFile()
	},
}

func getRequiredIndexArgs() []string {
	return []string{"owner", "path", "repo"}
}

func init() {
	rootCmd.AddCommand(indexCmd)
	flags := indexCmd.Flags()
	flags.StringP("owner", "o", "", "github username or organization")
	flags.StringP("repo", "r", "", "github repository")
	flags.StringP("path", "p", "", "Path to index file")
	flags.StringP("token", "t", "", "Github Auth Token (only needed for private repos)")
}
