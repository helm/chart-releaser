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

package cmd

import (
	"fmt"
	"os"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/helm/chart-releaser/pkg/git"
	"github.com/helm/chart-releaser/pkg/github"
	"github.com/helm/chart-releaser/pkg/releaser"
	"github.com/spf13/cobra"
)

// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "Update Helm repo index.yaml for the given GitHub repo",
	Long: `
Update a Helm chart repository index.yaml file based on a the
given GitHub repository's releases.
	`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		config, err := config.LoadConfiguration(cfgFile, cmd, getRequiredIndexArgs())
		if err != nil {
			return err
		}

		if len(config.ChartsRepo) > 0 {
			fmt.Fprintf(os.Stderr, "ATTENTION: Flag --charts-repo is deprecated. It does not have any effect.\n"+
				"The index.yaml is read from the '%s' branch instead.\n"+
				"Loading index.yaml directly from the charts repository lead to problems as there is a delay between\n"+
				"pushing to the GitHub pages branch until things appear online.\n"+
				"The flag will be removed with the next major release.", config.PagesBranch)
		}

		ghc := github.NewClient(config.Owner, config.GitRepo, config.Token, config.GitBaseURL, config.GitUploadURL)
		releaser := releaser.NewReleaser(config, ghc, &git.Git{})
		_, err = releaser.UpdateIndexFile()
		return err
	},
}

func getRequiredIndexArgs() []string {
	return []string{"owner", "git-repo"}
}

func init() {
	rootCmd.AddCommand(indexCmd)
	flags := indexCmd.Flags()
	flags.StringP("owner", "o", "", "GitHub username or organization")
	flags.StringP("git-repo", "r", "", "GitHub repository")
	flags.StringP("charts-repo", "c", "", "The URL to the charts repository")
	_ = flags.MarkHidden("charts-repo")
	flags.StringP("index-path", "i", ".cr-index/index.yaml", "Path to index file")
	flags.StringP("package-path", "p", ".cr-release-packages", "Path to directory with chart packages")
	flags.StringP("token", "t", "", "GitHub Auth Token (only needed for private repos)")
	flags.StringP("git-base-url", "b", "https://api.github.com/", "GitHub Base URL (only needed for private GitHub)")
	flags.StringP("git-upload-url", "u", "https://uploads.github.com/", "GitHub Upload URL (only needed for private GitHub)")
	flags.String("pages-branch", "gh-pages", "The GitHub pages branch")
	flags.String("pages-index-path", "index.yaml", "The GitHub pages index path")
	flags.String("remote", "origin", "The Git remote used when creating a local worktree for the GitHub Pages branch")
	flags.Bool("push", false, "Push index.yaml to the GitHub Pages branch (must not be set if --pr is set)")
	flags.Bool("pr", false, "Create a pull request for index.yaml against the GitHub Pages branch (must not be set if --push is set)")
	flags.String("release-name-template", "{{ .Name }}-{{ .Version }}", "Go template for computing release names, using chart metadata")
	flags.Bool("packages-with-index", false, "Host the package files in the GitHub Pages branch")
}
