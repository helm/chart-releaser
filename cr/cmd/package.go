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
	"fmt"
	"os"
	"path/filepath"

	"github.com/helm/chart-releaser/pkg/config"
	"github.com/spf13/cobra"
	"github.com/ulule/deepcopier"
	"helm.sh/helm/v3/pkg/action"
)

// packageCmd represents the package command
var packageCmd = &cobra.Command{
	Use:   "package [/path/to/chart1] [/path/to/chart/2]",
	Short: "Package Helm charts",
	Long: `Package Helm charts ready for release.
If you wish to use advanced packaging
options such as creating signed packages or
updating chart dependencies please use
"helm package" instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var settings map[string]interface{}
		var err error

		config, err := config.LoadConfiguration(cfgFile, cmd, getRequiredPackageArgs())
		if err != nil {
			return err
		}
		client := action.NewPackage()
		// valueOpts := &values.Options{}

		if config.PackagePath == "" {
			config.PackagePath = "."
		}
		client.Destination = config.PackagePath
		charts := args
		if len(charts) == 0 {
			charts[0] = "."
		}
		for i := 0; i < len(charts); i++ {
			path, err := filepath.Abs(charts[i])
			if err != nil {
				return err
			}
			if _, err := os.Stat(args[0]); err != nil {
				return err
			}
			deepcopier.Copy(config).To(settings)
			packageRun, err := client.Run(path, settings)

			if err != nil {
				return err
			}

			fmt.Printf("Successfully packaged chart in %s and saved it to: %s\n", path, packageRun)
		}
		return nil
	},
}

func getRequiredPackageArgs() []string {
	return []string{"package-path"}
}

func init() {
	rootCmd.AddCommand(packageCmd)
	packageCmd.Flags().StringP("package-path", "p", ".cr-release-packages", "Path to directory with chart packages")
}
