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
		return upload.Packages(uploadOptions)
	},
}

var uploadOptions = &upload.Options{}

func init() {
	rootCmd.AddCommand(uploadCmd)
	uploadCmd.Flags().StringVarP(&uploadOptions.Owner, "owner", "o", "", "github username or organization")
	uploadCmd.Flags().StringVarP(&uploadOptions.Repo, "repo", "r", "", "github repository")
	uploadCmd.Flags().StringVarP(&uploadOptions.Path, "path", "p", "", "Path to Helm Artifacts")
	uploadCmd.Flags().StringVarP(&uploadOptions.Token, "token", "t", "", "Github Auth Token")
	uploadCmd.Flags().BoolVar(&uploadOptions.Recursive, "recursive", false, "recursively find artifacts")
	uploadCmd.MarkFlagRequired("owner")
	uploadCmd.MarkFlagRequired("repo")
	uploadCmd.MarkFlagRequired("path")
	uploadCmd.MarkFlagRequired("token")
}
