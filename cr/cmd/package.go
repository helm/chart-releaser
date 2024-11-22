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
	"github.com/helm/chart-releaser/pkg/config"
	"github.com/helm/chart-releaser/pkg/packager"
	"github.com/spf13/cobra"
)

// packageCmd represents the package command
var packageCmd = &cobra.Command{
	Use:   "package [CHART_PATH] [...]",
	Short: "Package Helm charts",
	Long: `This command packages a chart into a versioned chart archive file. If a path
is given, this will look at that path for a chart (which must contain a
Chart.yaml file) and then package that directory.


If you wish to use advanced packaging options such as creating signed
packages or updating chart dependencies please use "helm package" instead.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var err error
		if len(args) == 0 {
			args = append(args, ".")
		}
		config, err := config.LoadConfiguration(cfgFile, cmd, getRequiredPackageArgs())
		if err != nil {
			return err
		}

		p := packager.NewPackager(config, args)
		return p.CreatePackages()

	},
}

func getRequiredPackageArgs() []string {
	return []string{"package-path"}
}

func init() {
	rootCmd.AddCommand(packageCmd)
	packageCmd.Flags().StringP("package-path", "p", ".cr-release-packages", "Path to directory with chart packages")
	packageCmd.Flags().Bool("sign", false, "Use a PGP private key to sign this package")
	packageCmd.Flags().String("key", "", "Name of the key to use when signing")
	packageCmd.Flags().String("keyring", "~/.gnupg/pubring.gpg", "Location of a public keyring")
	packageCmd.Flags().String("passphrase-file", "", "Location of a file which contains the passphrase for the signing key. Use '-' in order to read from stdin")
}
