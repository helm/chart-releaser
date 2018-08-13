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
	"github.com/paulczar/charthub/pkg/index"
	"github.com/spf13/cobra"
)

// indexCmd represents the index command
var indexCmd = &cobra.Command{
	Use:   "index",
	Short: "creates helm repo index.yaml for given github repo",
	Long: `
Creates a Helm Chart Repository index.yaml file based on a the
given github repository's releases.
	`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return index.Create(indexOptions)
	},
}

var indexOptions = &index.Options{}

func init() {
	rootCmd.AddCommand(indexCmd)
	indexCmd.Flags().StringVarP(&indexOptions.Owner, "owner", "o", "", "github username or organization")
	indexCmd.Flags().StringVarP(&indexOptions.Repo, "repo", "r", "", "github repository")
	indexCmd.Flags().StringVarP(&indexOptions.Path, "path", "p", "", "Path to index file")
	indexCmd.Flags().StringVarP(&indexOptions.Token, "token", "t", "", "Github Auth Token (only needed for private repos)")
	indexCmd.MarkFlagRequired("owner")
	indexCmd.MarkFlagRequired("repo")
	indexCmd.MarkFlagRequired("path")
}
