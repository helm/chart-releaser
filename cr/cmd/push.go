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
	"github.com/helm/chart-releaser/pkg/pusher"
	"github.com/spf13/cobra"
)

var pushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push Helm chart packages to an OCI registry",
	Long: `Push Helm chart packages to an OCI registry.

For every *.tgz file found under --package-path, the command pushes the chart
to <registry-url>/<chart-name>:<chart-version>. If a sibling *.tgz.prov
provenance file is present, it is pushed automatically.`,
	RunE: func(cmd *cobra.Command, _ []string) error {
		cfg, err := config.LoadConfiguration(cfgFile, cmd, getRequiredPushArgs())
		if err != nil {
			return err
		}
		return pusher.NewPusher(cfg).PushPackages()
	},
}

func getRequiredPushArgs() []string {
	// registry-url is validated inside the pusher so we can give a helpful
	// scheme-specific error message. The reflect-based required-flag check in
	// config.LoadConfiguration cannot resolve the RegistryURL field (the
	// generic kebab-case helper lowercases the URL acronym).
	return nil
}

func init() {
	rootCmd.AddCommand(pushCmd)
	pushCmd.Flags().StringP("registry-url", "r", "", "OCI registry URL (e.g. oci://ghcr.io/myorg/charts)")
	pushCmd.Flags().StringP("package-path", "p", ".cr-release-packages", "Path to directory with chart packages")
	pushCmd.Flags().StringP("username", "u", "", "Registry username (falls back to ~/.docker/config.json if unset)")
	pushCmd.Flags().String("password", "", "Registry password (falls back to ~/.docker/config.json if unset)")
	pushCmd.Flags().Bool("skip-existing", false, "Skip pushing chart versions that already exist in the registry")
	pushCmd.Flags().Bool("plain-http", false, "Use insecure HTTP connections")
	pushCmd.Flags().Bool("insecure-skip-tls-verify", false, "Skip TLS certificate verification")
	pushCmd.Flags().String("ca-file", "", "Verify registry certificate using this CA bundle")
	pushCmd.Flags().String("cert-file", "", "Identify registry client using this TLS certificate file")
	pushCmd.Flags().String("key-file", "", "Identify registry client using this TLS key file")
}
