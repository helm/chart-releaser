// Copyright © 2018 NAME HERE <EMAIL ADDRESS>
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
	"github.com/paulczar/charthub/pkg/config"
	"github.com/paulczar/charthub/pkg/upload"
	"github.com/spf13/cobra"
)

// uploadCmd represents the upload command
var uploadCmd = &cobra.Command{
	Use:   "upload",
	Short: "Uploads Helm Chart packages to github releases",
	Long:  `Uploads Helm Chart packages to github releases`,
	PreRunE: func(cmd *cobra.Command, args []string) error {
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		options, err := config.LoadConfiguration(cfgFile, cmd, []string{"owner", "path", "repo", "token"})
		if err != nil {
			return err
		}
		return upload.Packages(options)
	},
}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringP("owner", "o", "", "github username or organization")
	uploadCmd.Flags().StringP("repo", "r", "", "github repository")
	uploadCmd.Flags().StringP("path", "p", "", "Path to Helm Artifacts")
	uploadCmd.Flags().StringP("token", "t", "", "Github Auth Token")
	uploadCmd.Flags().Bool("recursive", false, "recursively find artifacts")
}
