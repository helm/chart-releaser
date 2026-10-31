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

package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadConfigurationPreRelease(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantLatest bool
		wantPre    bool
		wantErr    bool
	}{
		{name: "defaults", args: nil, wantLatest: true, wantPre: false},
		{name: "pre-release disables latest", args: []string{"--pre-release"}, wantLatest: false, wantPre: true},
		{name: "pre-release with latest explicitly disabled", args: []string{"--pre-release", "--make-release-latest=false"}, wantLatest: false, wantPre: true},
		{name: "pre-release with latest explicitly enabled", args: []string{"--pre-release", "--make-release-latest=true"}, wantErr: true},
		{name: "latest explicitly enabled without pre-release", args: []string{"--make-release-latest=true"}, wantLatest: true, wantPre: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: "upload"}
			cmd.Flags().Bool("make-release-latest", true, "")
			cmd.Flags().Bool("pre-release", false, "")
			require.NoError(t, cmd.ParseFlags(tt.args))

			opts, err := LoadConfiguration("", cmd, nil)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantLatest, opts.MakeReleaseLatest)
			assert.Equal(t, tt.wantPre, opts.PreRelease)
		})
	}
}

func TestLoadConfigurationUseAPIAssetURL(t *testing.T) {
	for _, tt := range []struct {
		name, env string
		args      []string
		want      bool
	}{
		{name: "disabled by default"},
		{name: "flag", args: []string{"--use-api-asset-url"}, want: true},
		{name: "environment", env: "true", want: true},
		{name: "flag overrides environment", env: "true", args: []string{"--use-api-asset-url=false"}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("CR_USE_API_ASSET_URL", tt.env)
			cmd := &cobra.Command{Use: "index"}
			cmd.Flags().Bool("use-api-asset-url", false, "")
			require.NoError(t, cmd.ParseFlags(tt.args))
			opts, err := LoadConfiguration("", cmd, nil)
			require.NoError(t, err)
			assert.Equal(t, tt.want, opts.UseAPIAssetURL)
		})
	}
}

func TestLoadConfigurationReplacement(t *testing.T) {
	for _, source := range []string{"flags", "environment", "file"} {
		t.Run(source, func(t *testing.T) {
			cmd := &cobra.Command{Use: "upload"}
			cmd.Flags().Bool("skip-existing", false, "")
			cmd.Flags().Bool("replace-existing", false, "")
			cfgFile := ""
			switch source {
			case "flags":
				require.NoError(t, cmd.ParseFlags([]string{"--skip-existing", "--replace-existing"}))
			case "environment":
				t.Setenv("CR_SKIP_EXISTING", "true")
				t.Setenv("CR_REPLACE_EXISTING", "true")
			case "file":
				cfgFile = filepath.Join(t.TempDir(), "cr.yaml")
				require.NoError(t, os.WriteFile(cfgFile, []byte("skip-existing: true\nreplace-existing: true\n"), 0o600))
			}
			_, err := LoadConfiguration(cfgFile, cmd, nil)
			require.ErrorContains(t, err, "specify either --skip-existing or --replace-existing")
		})
	}
}

func TestLoadConfigurationSignCommits(t *testing.T) {
	for _, source := range []string{"flags", "environment", "file"} {
		for _, tt := range []struct {
			name            string
			push, pr, token bool
			wantError       string
		}{
			{name: "valid", push: true, token: true},
			{name: "requires push", token: true, wantError: "requires --push"},
			{name: "rejects PR", pr: true, token: true, wantError: "cannot be used with --pr"},
			{name: "requires token", push: true, wantError: "--token is required"},
		} {
			t.Run(source+"/"+tt.name, func(t *testing.T) {
				cmd := &cobra.Command{Use: "index"}
				cmd.Flags().Bool("sign-commits", false, "")
				cmd.Flags().Bool("push", tt.push, "")
				cmd.Flags().Bool("pr", tt.pr, "")
				token := ""
				if tt.token {
					token = "token"
				}
				cmd.Flags().String("token", token, "")
				cfgFile := ""
				switch source {
				case "flags":
					require.NoError(t, cmd.ParseFlags([]string{"--sign-commits"}))
				case "environment":
					t.Setenv("CR_SIGN_COMMITS", "true")
				case "file":
					cfgFile = filepath.Join(t.TempDir(), "cr.yaml")
					require.NoError(t, os.WriteFile(cfgFile, []byte("sign-commits: true\n"), 0o600))
				}
				opts, err := LoadConfiguration(cfgFile, cmd, nil)
				if tt.wantError != "" {
					require.ErrorContains(t, err, tt.wantError)
				} else {
					require.NoError(t, err)
					assert.True(t, opts.SignCommits)
				}
			})
		}
	}
}

func TestSignCommitsSharedConfiguration(t *testing.T) {
	cfgFile := filepath.Join(t.TempDir(), "cr.yaml")
	require.NoError(t, os.WriteFile(cfgFile, []byte("sign-commits: true\n"), 0o600))
	for _, name := range []string{"package", "upload"} {
		t.Run(name, func(t *testing.T) {
			_, err := LoadConfiguration(cfgFile, &cobra.Command{Use: name}, nil)
			require.NoError(t, err, "index signing settings must not affect other commands")
		})
	}
}
